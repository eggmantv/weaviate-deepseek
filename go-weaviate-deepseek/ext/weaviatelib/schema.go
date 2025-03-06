package weaviatelib

import (
	"context"
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate/entities/models"
)

var WeaviateURI string
var L *logrus.Entry

const (
	ClsRubyGPT string = "RubyGPT"
)

type VectorizerFuncDef func(string) ([]float32, error)

var VectorizerFunc VectorizerFuncDef

func init() {
	if len(os.Getenv("HOST_IP")) == 0 {
		WeaviateURI = "localhost:8070"
	} else {
		WeaviateURI = os.Getenv("HOST_IP") + ":8070"
	}
}

func Setup(l *logrus.Entry) {
	L = l
}

// ClearAllSchema remove all schema(dbs)
func ClearAllSchema() error {
	return GetClient().Schema().AllDeleter().
		Do(context.Background())
}

func RemoveSchema(clsName string) error {
	clsName = GetClsName(clsName)
	return GetClient().Schema().ClassDeleter().
		WithClassName(clsName).
		Do(context.Background())
}

func DefineTextSchema(clsName, schemaStr, desp string) error {
	clsName = GetClsName(clsName)
	client := GetClient()
	creator := client.Schema().ClassCreator()
	// properties := []*models.Property{
	// 	{
	// 		Name:     "title",
	// 		DataType: []string{"string"},
	// 	},
	// 	{
	// 		Name:     "captions",
	// 		DataType: []string{"text"},
	// 	},
	// 	{
	// 		Name:     "url",
	// 		DataType: []string{"string"},
	// 		// ModuleConfig: map[string]interface{}{
	// 		// 	"text2vec-openai": map[string]interface{}{
	// 		// 		"skip": true,
	// 		// 	},
	// 		// },
	// 	},
	// }
	properties := make([]*models.Property, 0)
	err := json.Unmarshal([]byte(schemaStr), &properties)
	if err != nil {
		return err
	}
	creator = creator.WithClass(&models.Class{
		Class:       clsName,
		Description: desp,   //"Captions extract from eggman videos using Azure STT service",
		Vectorizer:  "none", // text2vec-openai
		ModuleConfig: map[string]interface{}{
			"text2vec-openai": map[string]interface{}{
				"model":        "ada",
				"modelVersion": "002",
				"type":         "text",
			},
		},
		Properties: properties,
	})
	err = creator.Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// DefineImageSchema
// https://weaviate.io/blog/how-to-build-an-image-search-application-with-weaviate
func DefineImageSchema(clsName, schemaStr, desp string) error {
	clsName = GetClsName(clsName)
	client := GetClient()
	creator := client.Schema().ClassCreator()
	properties := make([]*models.Property, 0)
	err := json.Unmarshal([]byte(schemaStr), &properties)
	if err != nil {
		return err
	}
	creator = creator.WithClass(&models.Class{
		Class:       clsName,
		Description: desp,
		ModuleConfig: map[string]interface{}{
			"img2vec-neural": map[string]interface{}{
				"imageFields": []string{"image"},
			},
		},
		Vectorizer:      "img2vec-neural",
		VectorIndexType: "hnsw",
		Properties:      properties,
	})
	err = creator.Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func GetSchema() ([]byte, error) {
	client := GetClient()
	schema, err := client.Schema().Getter().Do(context.Background())
	if err != nil {
		return nil, err
	}
	scheB, err := schema.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return scheB, nil
}

func GetClient() *weaviate.Client {
	cfg := weaviate.Config{
		Host:   WeaviateURI,
		Scheme: "http",
	}
	client, err := weaviate.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	return client
}

// GetClsName weaviate 会自动把clsname首字母转换成大写，所以这里统一处理
func GetClsName(clsName string) string {
	// RubyChat需要特殊处理
	if clsName == ClsRubyGPT {
		return clsName
	}
	return "A" + clsName
}
