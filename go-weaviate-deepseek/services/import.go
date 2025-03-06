package services

import (
	"go-weaviate-deepseek/ext"
	"go-weaviate-deepseek/services/scrape"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

const (
	minTextLength = 30
)

func lim() *logrus.Entry {
	return ext.LF("import")
}

type ImportSource struct {
	ClsName string `json:"cls_name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

func (i *ImportSource) Do() error {
	doc := gjson.Parse(i.Data)
	switch i.Type {
	case "text":
		return i.handleText(i.Data, ext.M{
			"title":      "",
			"url":        "",
			"media_type": "text",
		})
	case "url", "one_url":
		entryURL := doc.Get("url").String()
		domains := make([]string, 0)
		if doc.Get("domains").Exists() {
			domains = strings.Split(doc.Get("domains").String(), ",")
		}
		scraper := scrape.NewScraper(entryURL, domains)
		if i.Type == "one_url" {
			scraper.SetDepth(1)
		}

		res, err := scraper.Start()
		if err != nil {
			return err
		}
		lim().Printf("scrape url done, url: %s, start creating vector data", entryURL)
		for urlStr, v := range res {
			txt := cast.ToString(v["text"])
			err := i.handleText(txt, ext.M{
				"title":      v["title"],
				"url":        urlStr,
				"media_type": "url",
			})
			if err != nil {
				return err
			}
		}
	case "image":
		b64 := doc.Get("base64").String()
		title := doc.Get("title").String()
		urlStr := doc.Get("url").String()
		txt, err := ExtractTextFromImage(b64, true)
		if err != nil {
			return err
		}
		return i.handleText(ext.Oneline(txt), ext.M{
			"title":      title,
			"url":        urlStr,
			"media_type": "image",
		})
	}

	return nil
}

func (i *ImportSource) handleText(bigText string, addiAttrs ext.M) error {
	chunks := ChunkSplit(bigText, CHUNK_SIZE)
	var err error
	for _, ca := range chunks {
		if !isMeetMinLength(ca.Chunk) {
			lim().Printf("chunk length is less than %d, text: %s, skip save", minTextLength, ca.Chunk)
			continue
		}

		err = ca.CalVector()
		if err != nil {
			return err
		}
		err = ca.Save(i.ClsName, addiAttrs)
		if err != nil {
			lim().Errorln("save chunk err:", err)
			return err
		}
	}
	return nil
}

func isMeetMinLength(txt string) bool {
	return len([]rune(txt)) > minTextLength
}
