package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BasicAuth 基础认证
func BasicAuth(authUsername, authPassword string) gin.HandlerFunc {
	return func(gc *gin.Context) {
		username, password, hasAuth := gc.Request.BasicAuth()
		if !hasAuth {
			gc.Header("WWW-Authenticate", `Basic realm="Authentication required"`)
			gc.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if username != authUsername || password != authPassword {
			gc.Header("WWW-Authenticate", `Basic realm="Invalid credentials"`)
			gc.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		gc.Next()
	}
}
