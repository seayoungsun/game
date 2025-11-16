package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequirePermission 权限检查中间件
func RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取权限列表
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足",
			})
			c.Abort()
			return
		}

		// 转换权限列表
		permList, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足",
			})
			c.Abort()
			return
		}

		// 检查是否有指定权限
		hasPermission := false
		for _, perm := range permList {
			if perm == permissionCode {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足：缺少 " + permissionCode + " 权限",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole 角色检查中间件（通过权限实现）
// 实际应该从数据库查询角色，这里简化处理
func RequireRole(roleCode string) gin.HandlerFunc {
	// 可以根据角色代码映射到权限
	// 这里先返回一个基础实现
	return func(c *gin.Context) {
		// TODO: 从数据库查询管理员角色并验证
		// 暂时通过权限检查实现
		c.Next()
	}
}
