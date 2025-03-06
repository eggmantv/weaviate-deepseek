package api

import (
	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/ext"

	"github.com/gin-gonic/gin"
)

// authHeaderMiddlewareWithoutPaths 可以传递哪些路由不需要验证，默认都需要
func authHeaderMiddlewareWithoutPaths(withoutPaths ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		pa := ctx.FullPath()
		for _, pattern := range withoutPaths {
			if len(pattern) == 0 {
				continue
			}
			if pattern == pa {
				ctx.Next()
				return
			}
		}

		secret := ctx.GetHeader(conf.AuthHeaderKey)
		if secret != conf.AuthHeaderSecret {
			ctx.AbortWithStatusJSON(401, ext.M{
				"status": "error",
				"error":  "api secret is invalid",
			})
			return
		}
		ctx.Next()
	}
}
