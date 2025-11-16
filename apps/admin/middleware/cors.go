package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 允许的来源（可以根据需要配置）
		allowedOrigins := []string{
			"http://localhost:3000", // Vue开发服务器
			"http://localhost:5173", // Vite默认端口
			"http://localhost:8080", // 生产环境前端
			"http://localhost:8000", // Vue CLI默认端口
		}

		// 检查来源是否允许
		allowOrigin := ""
		if origin != "" {
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					allowOrigin = origin
					break
				}
			}
			// 如果未匹配，在开发环境下允许所有来源
			if allowOrigin == "" {
				allowOrigin = origin // 开发环境允许所有来源
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
