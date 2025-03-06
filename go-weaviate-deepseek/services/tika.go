package services

import (
	"context"
	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/services/scrape"
	"os"

	"github.com/google/go-tika/tika"
)

// 读取doc/xls/pdf/ppt内容
func ReadByTika(path string) (string, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return "", err
	}

	client := tika.NewClient(nil, conf.TIKA_HOST)
	content, err := client.Parse(context.TODO(), f)
	if err != nil {
		return "", err
	}
	sc := scrape.GetSanitizer()
	content = sc.Sanitize(content)

	content = RE_CHUNK_NEWLINE.ReplaceAllString(content, "\n")
	content = RE_CHUNK_SPACE.ReplaceAllString(content, " ")

	// content = strings.TrimSpace(content)
	return content, nil
}
