package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/pkg/utils"
)

// AdminAuthMiddleware 管理员认证中间件
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少认证Token",
			})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Token格式错误",
			})
			c.Abort()
			return
		}

		// 解析Token
		claims, err := utils.ParseAdminToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Token无效或已过期",
			})
			c.Abort()
			return
		}

		// 将管理员信息存储到上下文
		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Set("admin_username", claims.Username) // 兼容操作日志中间件
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}
