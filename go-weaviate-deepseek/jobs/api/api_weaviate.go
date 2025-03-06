package api

import (
	"encoding/json"
	"fmt"
	"go-weaviate-deepseek/ext"
	"go-weaviate-deepseek/ext/weaviatelib"
	"go-weaviate-deepseek/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func lwea() *logrus.Entry {
	return ext.LF("weaviate")
}

func apiWeaviate(r *gin.Engine) {
	// create new db(schema)
	// {
	// 	"cls_name": "xxccc",
	// 	"desp": "desp",
	// 	"schema": "[{\"name\":\"title\",\"dataType\":[\"string\"]},{\"name\":\"captions\",\"dataType\":[\"text\"]},{\"name\":\"url\",\"dataType\":[\"string\"]},{\"name\":\"media_type\",\"dataType\":[\"string\"]}]"
	// }
	r.POST("/weaviate/create_db", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		clsName := doc.Get("cls_name").String()
		desp := doc.Get("desp").String()
		schema := doc.Get("schema").String() // schema is json string

		err := weaviatelib.DefineTextSchema(clsName, schema, desp)
		if ok := checkErr(err, ctx); !ok {
			return
		}
		ctx.JSON(http.StatusOK, ext.M{"status": "ok"})
	})

	// remove one db
	r.POST("/weaviate/remove_db", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		clsName := doc.Get("cls_name").String()

		err := weaviatelib.RemoveSchema(clsName)
		if ok := checkErr(err, ctx); !ok {
			return
		}
		ctx.JSON(http.StatusOK, ext.M{"status": "ok"})
	})

	// get db schema info
	r.GET("/weaviate/db_info", func(ctx *gin.Context) {
		// clsName := ctx.Query("cls_name")
		b, err := weaviatelib.GetSchema()
		if ok := checkErr(err, ctx); !ok {
			return
		}
		res := ext.M{}
		err = json.Unmarshal(b, &res)
		if ok := checkErr(err, ctx); !ok {
			return
		}
		ctx.JSON(http.StatusOK, ext.M{"status": "ok", "data": res})
	})

	r.POST("/weaviate/search", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		prompt := doc.Get("prompt").String()
		distance := doc.Get("distance").Float()
		clsName := doc.Get("cls_name").String()

		lwea().Printf("weaviate/search, clsName: %s, prompt: %s, distance: %f", clsName, prompt, distance)

		b, err := weaviatelib.Query(clsName, prompt, float32(distance))
		if ok := checkErr(err, ctx); !ok {
			return
		}
		fmt.Println("b:", string(b))
		texts := make([]gjson.Result, 0)
		getClsName := weaviatelib.GetClsName(clsName)
		gjson.ParseBytes(b).Get(getClsName).ForEach(func(k, v gjson.Result) bool {
			texts = append(texts, v)
			return true
		})

		lwea().Printf("/weaviate/search, result: %s", texts)
		ctx.JSON(http.StatusOK, ext.M{"status": "ok", "data": texts})
	})

	// for testing
	r.POST("/weaviate/scan", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		clsName := doc.Get("cls_name").String()

		res, err := weaviatelib.Scan(clsName)
		if ok := checkErr(err, ctx); !ok {
			return
		}
		ctx.JSON(http.StatusOK, ext.M{"status": "ok", "data": res})
	})

	// insert new data
	// {
	//  "cls_name": "xxx",
	// 	"type": "audio" | "video" | "pdf" | "word" | "url" | "one_url" | "text",
	// 	"data": "xx"
	// }
	// type url:
	// {
	// 	"cls_name": "aabbcc",
	// 	"type": "url",
	// 	"data": "{\"url\":\"https://eggman.tv\",\"domains\":\"eggman.tv,\"}"
	// }
	// type image:
	// {
	// 	"cls_name": "aabbcc",
	// 	"type": "image",
	// 	"data": "{\"base64\":\"xxxxxx\",\"url\":\"https://eggman.tv/a.png\"\"title\":\"a image desp\"}"
	// }
	r.POST("/weaviate/create", func(ctx *gin.Context) {
		str := readBody(ctx)
		i := services.ImportSource{}
		err := json.Unmarshal([]byte(str), &i)
		if ok := checkErr(err, ctx); !ok {
			return
		}
		go func() {
			trackid := ext.GenGlobalID()
			lwea().Printf("import start, %s, source type: %s, cls_name: %s", trackid, i.Type, i.ClsName)
			err := i.Do()
			if err != nil {
				lwea().Warn("weaviate create err:", err)
				return
			}
			lwea().Printf("import done, %s, source type: %s, cls_name: %s", trackid, i.Type, i.ClsName)
		}()

		ctx.JSON(http.StatusOK, ext.M{"status": "ok"})
	})

	r.POST("/weaviate/delete", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		id := doc.Get("id").String()
		clsName := doc.Get("cls_name").String()

		lwea().Printf("/weaviate/delete, id: %s", id)

		// 删除
		err := weaviatelib.DeleteByID(clsName, id)
		if ok := checkErr(err, ctx); !ok {
			return
		}

		ctx.JSON(http.StatusOK, ext.M{"status": "ok"})
	})

	r.POST("/weaviate/count", func(ctx *gin.Context) {
		str := readBody(ctx)
		doc := gjson.Parse(str)
		clsName := doc.Get("cls_name").String()

		count, err := weaviatelib.Count(clsName)
		if ok := checkErr(err, ctx); !ok {
			return
		}
		ctx.JSON(http.StatusOK, ext.M{"status": "ok", "data": count})
	})
}
