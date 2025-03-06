package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/ext"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func l() *logrus.Entry {
	return ext.LF("web-api")
}

// RunWebAPI run http sever
func RunWebAPI(c chan string) {
	gin.SetMode(gin.ReleaseMode)
	// gin.DisableConsoleColor()
	r := gin.New()

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("GIN[%s] %s %s %s %s %d %s \"%s\" %s\"\n",
			param.TimeStamp.Format(time.RFC3339),
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())
	r.Use(authHeaderMiddlewareWithoutPaths(
		"/",
		"/ds-ws",  // websocket connection
		"/ds-sse", // SSE connection
	))

	apiDeepSeekAliyun(r)
	apiWeaviate(r)
	apiWS(r)

	r.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hi, please access https://eggman.tv to start:)")
	})

	go r.Run(conf.WebAPIPort)
	l().Info("start Web API at:", conf.WebAPIPort)

	<-c
}

func readBody(ctx *gin.Context) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(ctx.Request.Body)
	return buf.String()
}

func checkErr(err error, ctx *gin.Context) bool {
	if err != nil {
		ctx.JSON(http.StatusOK, ext.M{
			"status": "error",
			"error":  err.Error(),
		})
		return false
	}
	return true
}
