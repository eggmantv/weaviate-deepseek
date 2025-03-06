package api

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/ext"
	"go-weaviate-deepseek/services"
	"go-weaviate-deepseek/services/scrape"

	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"gopkg.in/resty.v1"
)

type PromptParser struct {
	ExtHolders map[string]map[string]string
	ExtRes     map[string]string
	wait       sync.WaitGroup

	Res     []string // prompts array
	resLock sync.Mutex
	Configs ext.M // such as chat models to use
}

// For match: "<a>aa</a> hello world <|343_url|>https://eggman.tv</|343_url|>, haha, three litte <|343_url|>https://www.343cloud.com</|343_url|> birds."
// var urlMather = regexp.MustCompile(`<\|343_url\|>([^>]*)<\/\|343_url\|>`)
// For match: "hello world RMO_EXT{FILE:file-key}, genius"
var varMatcher = regexp.MustCompile(`RMO_VAR\{([^\{]*)\}`)
var extMatcher = regexp.MustCompile(`RMO_EXT\{([^\{]*)\}`)

func HasExtInPrompt(prompt string) bool {
	return extMatcher.MatchString(prompt)
}

// ParsePrompt return prompts those need to send in batching
func ParsePrompt(prompt, optionValues string) (*PromptParser, error) {
	pp := PromptParser{
		ExtHolders: make(map[string]map[string]string, 0),
		ExtRes:     make(map[string]string, 0),

		Res: make([]string, 0),
		// default configs
		Configs: ext.M{
			"chat_model": conf.AliDeepSeekModelName,
		},
	}
	valuesDoc := gjson.Parse(optionValues)
	// parse chat model configs
	chM := getOptionValue(valuesDoc, "_chat_model", "value")
	if len(chM) > 0 {
		pp.Configs["chat_model"] = chM
	}
	modelConfig, exists := chatModelConfigs[cast.ToString(pp.Configs["chat_model"])]
	if !exists {
		return nil, fmt.Errorf("chat_model %s is invalid", cast.ToString(pp.Configs["chat_model"]))
	}

	// find all variables, and replace each with uuid
	newPrompt := varMatcher.ReplaceAllStringFunc(prompt, func(ma string) string {
		uid := ext.GenUUID()
		res := varMatcher.FindAllStringSubmatch(ma, -1)

		pp.ExtHolders[uid] = map[string]string{
			"ext_type":     "VAR",
			"ext_var_name": res[0][1],
		}
		return uid
	})
	// find all extensions
	newPrompt = extMatcher.ReplaceAllStringFunc(newPrompt, func(ma string) string {
		uid := ext.GenUUID()
		res := extMatcher.FindAllStringSubmatch(ma, -1)
		extensions := strings.Split(res[0][1], ":")
		if len(extensions) != 2 {
			return ma // illegal format
		}

		pp.ExtHolders[uid] = map[string]string{
			"ext_type":     extensions[0],
			"ext_var_name": extensions[1],
		}
		return uid
	})

	// parse prompts
	for uuid, e := range pp.ExtHolders {
		extValue := getOptionValue(valuesDoc, e["ext_var_name"], "value")
		if len(extValue) == 0 {
			continue
		}

		switch e["ext_type"] {
		case "VAR":
			pp.resLock.Lock()
			pp.ExtRes[uuid] = extValue
			pp.resLock.Unlock()
		case "FILE":
			pp.wait.Add(1)
			go pp.handleFile(uuid, extValue)
		case "URL":
			pp.wait.Add(1)
			go pp.handleURL(uuid, extValue)
		default:
			ppml().Warnf("unknown ext_type: %s", e["ext_type"])
			return nil, fmt.Errorf("unknown variable type: %s", e["ext_type"])
		}
	}
	pp.wait.Wait()

	// Split chunk
	// Only one variable can trigger chunk spliting, otherwise will be meaningless
	autoSplit := getOptionValue(valuesDoc, "_auto_split", "value")
	chunkSizeConfig := modelConfig["chunk_size"].(ext.M)
	splitedUsed := false
	if autoSplit == "true" {
		for uuid, e := range pp.ExtHolders {
			if splitedUsed {
				break
			}
			v, exists := pp.ExtRes[uuid]
			if !exists {
				v = ""
			}
			cs := cast.ToInt(chunkSizeConfig[e["ext_type"]]) - ext.TokenLen(newPrompt)
			if ext.TokenLen(v) > cs {
				chunks := services.ChunkSplit(v, cs)
				for _, c := range chunks {
					cprompt := strings.ReplaceAll(newPrompt, uuid, c.Chunk)
					pp.Res = append(pp.Res, cprompt)
				}
				splitedUsed = true
			}
		}
	}

	// no variables and ext
	if len(pp.Res) == 0 {
		pp.Res = append(pp.Res, newPrompt)
	}
	// replace left variables
	for uuid := range pp.ExtHolders {
		for i, p := range pp.Res {
			v, exists := pp.ExtRes[uuid]
			if !exists {
				v = ""
			}
			pp.Res[i] = strings.ReplaceAll(p, uuid, v)
		}
	}

	ppml().Printf("prompt batch count: %d， autoSplit: %v", len(pp.Res), autoSplit)
	return &pp, nil
}

func (pp *PromptParser) handleURL(uuid, u string) {
	defer pp.wait.Done()

	scraper := scrape.NewScraper(u, []string{})
	scraper.SetDepth(1)
	res, err := scraper.Start()
	if err != nil {
		ppml().Warnf("scrape err, url: %s, err: %s", u, err.Error())
		return
	}
	ppml().Printf("scrape url in prompt done, url: %s", u)

	// Put all pages' content together
	allContents := ""
	for _, v := range res {
		allContents += cast.ToString(v["text"]) + ". "
	}

	pp.resLock.Lock()
	pp.ExtRes[uuid] = allContents
	pp.resLock.Unlock()
}

func (pp *PromptParser) handleFile(uuid, fileURL string) {
	defer pp.wait.Done()

	tmpFile, err := os.CreateTemp("/tmp", "redmoextfile-*")
	if err != nil {
		ppml().Warn("err, create tmp file, err:", err)
		return
	}
	defer tmpFile.Close()

	pp.resLock.Lock()
	defer pp.resLock.Unlock()
	ppml().Printf("start downloading file: %s, dst: %s", fileURL, tmpFile.Name())
	_, err = resty.New().
		SetTimeout(180 * time.Second).
		R().
		SetOutput(tmpFile.Name()).
		Get(fileURL)
	if err != nil {
		errMsg := fmt.Sprintf("err, download file, file: %s, err: %s", fileURL, err)
		pp.ExtRes[uuid] = errMsg
		ppml().Warnf(errMsg)
		return
	}
	defer os.Remove(tmpFile.Name())
	ppml().Printf("download finished, start read, file: %s, dst: %s", fileURL, tmpFile.Name())

	res, err := services.ReadByTika(tmpFile.Name())
	if err != nil {
		ppml().Warnf("err, tika read file, file: %s, err: %s", fileURL, err)
		return
	}
	ppml().Printf("read finished, file: %s, dst: %s, length: %d", fileURL, tmpFile.Name(), len(res))

	pp.ExtRes[uuid] = res
}

func getOptionValue(values gjson.Result, nameValue, targetKey string) string {
	var res string
	values.ForEach(func(k, v gjson.Result) bool {
		if v.Get("name").String() == nameValue {
			res = v.Get(targetKey).String()
			return false
		}
		return true
	})
	return res
}
