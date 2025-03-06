package api

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/ext"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"gopkg.in/resty.v1"
)

const (
	redisChatSSETicketPrefix = "chat:ticket:"
)

/*
目前我们的账号tokens限额是
gpt-3.5-turbo-16k	180,000
gpt-4	10,000

Price:

	INPUT					OUTPUT

gpt-3.5-turbo-16k $0.003 / 1K tokens	$0.004 / 1K tokens
*/
var chatModelConfigs = map[string]map[string]interface{}{
	"gpt-3.5-turbo-16k": {
		"legal_name": openai.GPT3Dot5Turbo16K,
		"max_tokens": 6000,
		"chunk_size": ext.M{
			"VAR":  6000,
			"URL":  6000,
			"FILE": 6000,
		},
	},
	"gpt-4": {
		"legal_name": openai.GPT4,
		"max_tokens": 1500,
		"chunk_size": ext.M{
			"VAR":  2000,
			"URL":  2000,
			"FILE": 2000,
		},
	},
	conf.AliDeepSeekModelName: {
		"legal_name": conf.AliDeepSeekModelName,
		"max_tokens": 1500,
		"chunk_size": ext.M{
			"VAR":  2000,
			"URL":  2000,
			"FILE": 2000,
		},
	},
}

func lada() *logrus.Entry {
	return ext.LF("deepseek_aliyun")
}

func apiDeepSeekAliyun(r *gin.Engine) {
	// curl -i -X POST -H "X_ZB_KEY: SMiCLZ8G06bOReA1o8D3UMxfUB2iax74cux6O17" -d '{"prompt": "给我讲一个笑话"}' "http://localhost:5011/deep_seek_ali/chat_now"
	// 不使用流式响应，实时返回信息，和上面的采用的模型一样
	r.POST("/deep_seek_ali/chat_now", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		pm := doc.Get("prompt").String()
		userUUID := doc.Get("user_uuid").String()
		hasContext := doc.Get("has_context").Bool()

		lada().Printf("chat_now, prompt: %s, user_uuid: %s", pm, userUUID)

		jobUUID := ext.GenUUID()
		rsp, err := ChatNow(jobUUID, pm, userUUID, hasContext)
		if err != nil {
			lada().Printf("err, /deep_seek_ali/chat_now: %v", err)
			ctx.JSON(http.StatusOK, ext.M{
				"status":   "error",
				"job_uuid": jobUUID,
				"error":    err.Error(),
			})
			return
		}
		usageB, _ := json.Marshal(rsp.Usage)
		lada().Printf("token usage: %s", string(usageB))
		ctx.JSON(http.StatusOK, ext.M{"status": "ok", "data": ext.M{
			"job_uuid": jobUUID,
			"prompt":   pm,
			"rsp":      rsp,
		}})
	})

	// curl -i -X POST -H "X_ZB_KEY: SMiCLZ8G06bOReA1o8D3UMxfUB2iax74cux6O17" -d '{"prompt": "给我讲一个笑话"}' "http://localhost:5011/deep_seek_ali/embedding"
	// 计算向量接口
	r.POST("/deep_seek_ali/embedding", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		pm := doc.Get("prompt").String()
		// userUUID := doc.Get("user_uuid").String()

		// jobUUID := ext.GenUUID()
		rsp, err := Embedding(pm)
		if err != nil {
			lada().Printf("err, embedding: %v", err)
			ctx.JSON(http.StatusOK, ext.M{
				"status": "error",
				// "job_uuid": jobUUID,
				"error": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, ext.M{"status": "ok", "data": ext.M{
			// "job_uuid": jobUUID,
			"prompt": pm,
			"rsp":    rsp,
		}})
	})
}

// ChatNow wait until getting all response from openai
func ChatNow(jobUUID, prompt, userUUID string, hasContext bool) (*openai.ChatCompletionResponse, error) {
	lada().Printf("chat now, handling, prompt: %s, hasContext: %v", prompt, hasContext)
	messageRows := make([]openai.ChatCompletionMessage, 0)
	if hasContext {
		err := json.Unmarshal([]byte(prompt), &messageRows)
		if err != nil {
			lada().Warn("unmarshal prompt json err:", err)
			return nil, err
		}
	} else {
		messageRows = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		}
	}

	requestBody := map[string]interface{}{
		"model":       conf.AliDeepSeekModelName,
		"max_tokens":  2000,
		"temperature": 0.7,
		"top_p":       1,
		// "frequency_penalty": 0,
		"presence_penalty": 0,
		"messages":         messageRows,
	}

	resp, err := resty.SetTimeout(time.Duration(3*time.Minute)).R().
		SetHeader("Authorization", "Bearer "+conf.AliDeepSeekAPIKey).
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		Post(conf.AliDeepSeeKBaseUrl + "/chat/completions")

	if err != nil {
		lada().Fatalf("Error /chat/completions request: %v", err)
	}

	bodyDoc := gjson.ParseBytes(resp.Body())
	if bodyDoc.Get("error").Exists() {
		lada().Errorf("Error /chat/completions request: %v", bodyDoc.Get("error").String())
		return nil, errors.New(bodyDoc.Get("error").String())
	}

	var d openai.ChatCompletionResponse
	json.Unmarshal(resp.Body(), &d)
	return &d, nil
}

func Embedding(prompt string) (*openai.EmbeddingResponse, error) {
	lada().Infof("embedding, prompt: %s", prompt)

	requestBody := map[string]interface{}{
		"model":           "text-embedding-v3",
		"input":           prompt,
		"encoding_format": "float",
	}

	resp, err := resty.SetTimeout(time.Duration(3*time.Minute)).R().
		SetHeader("Authorization", "Bearer "+conf.AliDeepSeekAPIKey).
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		Post(conf.AliDeepSeeKBaseUrl + "/embeddings")

	if err != nil {
		lada().Fatalf("Error embeddings request: %v", err)
	}

	bodyDoc := gjson.ParseBytes(resp.Body())
	// lada().Info("Embedding: ", bodyDoc)
	if bodyDoc.Get("error").Exists() {
		lada().Errorf("Error embeddings request: %v", bodyDoc.Get("error").String())
		return nil, errors.New(bodyDoc.Get("error").String())
	}

	var d openai.EmbeddingResponse
	json.Unmarshal(resp.Body(), &d)
	return &d, nil
}

// Vectorizer 用于直接计算
func Vectorizer(prompt string) ([]float32, error) {
	rsp, err := Embedding(prompt)
	if err != nil {
		return []float32{}, err
	}
	if len(rsp.Data) > 0 {
		return rsp.Data[0].Embedding, nil
	}
	return []float32{}, errors.New("empty embedding")
}

func doNotify(url string, jobRes ext.M) (*resty.Response, error) {
	body, _ := json.Marshal(jobRes)
	rsp, err := resty.New().
		SetTimeout(15*time.Second).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		R().
		SetBody(body).
		SetHeader(conf.AuthHeaderKey, conf.AuthHeaderSecret).
		Post(url)
	if !rsp.IsSuccess() {
		lada().Warnf("err, notify job error: %v, status: %d", err, rsp.StatusCode())
		return rsp, err
	}
	return rsp, nil
}

func notifyJob(url string, jobRes ext.M) *resty.Response {
	j := ext.GenGlobalID()
	if len(url) == 0 {
		// lada().Warnf("notify url is empty, skip notify, j: %s", j)
		return nil
	}

	wait := time.After(60 * time.Second)
	for {
		select {
		case <-wait:
			lada().Errorf("critical, notify job timeout, url: %s, j: %s", url, j)
			return nil
		default:
			rsp, err := doNotify(url, jobRes)
			if err != nil || rsp.StatusCode() != 200 {
				lada().Errorf("critical, notify job error, err: %v, url: %s, body: %s", err, url, string(rsp.Body()))
				time.Sleep(2 * time.Second)
				continue
			}
			return rsp
		}
	}
}

func notifyJobTo3rd(webHook string, res ext.M) {
	data := res["data"].(ext.M)
	if cast.ToString(res["status"]) == "ok" {
		notifyJob(webHook, ext.M{
			"status": res["status"],
			"data": ext.M{
				// "prompt":             data["prompt"],
				"parent_thread_uuid": data["parent_chat_uuid"],
				"thread_uuid":        data["chat_uuid"],
				"content":            data["content"],
				"is_finished":        data["is_finished"],
				// "prompt_tokens":    data["prompt_tokens"],
				// "content_tokens":   data["content_tokens"],
				// "web_hook":      data["web_hook"],
				"reason":     data["reason"],
				"chat_model": data["chat_model"],
				"chunks":     data["chunks"],
				// "prompt_chains": data["prompt_chains"], // may be blank
				"workflow": data["workflow"],
			},
		})
	} else {
		notifyJob(webHook, ext.M{
			"status": res["status"],
			"data": ext.M{
				"error":              data["error"],
				"thread_uuid":        data["chatUUID"],
				"parent_thread_uuid": data["parentChatUUID"],
				"chunks":             data["chunks"],
				"prompt_chains":      data["promptChains"],
				"workflow":           data["workflow"],
			},
		})
	}
}
