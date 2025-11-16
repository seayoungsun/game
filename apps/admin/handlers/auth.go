package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/pkg/services"
	"github.com/kaifa/game-platform/pkg/utils"
)

// Login 管理员登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 调用服务登录
	adminService := services.NewAdminService()
	admin, err := adminService.Login(req.Username, req.Password, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": err.Error(),
		})
		return
	}

	// 获取管理员权限
	permissions, err := adminService.GetAdminWithPermissions(admin.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取权限失败",
		})
		return
	}

	// 生成Token
	token, err := utils.GenerateAdminToken(admin.ID, admin.Username, permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成Token失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data": gin.H{
			"token": token,
			"admin": gin.H{
				"id":          admin.ID,
				"username":    admin.Username,
				"nickname":    admin.Nickname,
				"email":       admin.Email,
				"avatar":      admin.Avatar,
				"permissions": permissions,
			},
		},
	})
}

// Logout 管理员退出登录
func Logout(c *gin.Context) {
	// JWT是无状态的，客户端删除Token即可
	// 这里可以记录退出日志或使Token失效（需要Token黑名单机制）
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "退出成功",
	})
}

// GetProfile 获取当前管理员信息
func GetProfile(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	adminIDUint := adminID.(uint)

	adminService := services.NewAdminService()
	admin, err := adminService.GetAdminByID(adminIDUint)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "管理员不存在",
		})
		return
	}

	// 获取权限
	permissions, _ := adminService.GetAdminWithPermissions(adminIDUint)
	// 获取角色
	roles, _ := adminService.GetAdminRoles(adminIDUint)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"id":            admin.ID,
			"username":      admin.Username,
			"nickname":      admin.Nickname,
			"email":         admin.Email,
			"avatar":        admin.Avatar,
			"status":        admin.Status,
			"last_login_at": admin.LastLoginAt,
			"last_login_ip": admin.LastLoginIP,
			"permissions":   permissions,
			"roles":         roles,
		},
	})
}

// GetPermissions 获取当前管理员的权限列表
func GetPermissions(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	adminIDUint := adminID.(uint)

	adminService := services.NewAdminService()
	permissions, err := adminService.GetAdminWithPermissions(adminIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取权限失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": permissions,
	})
}
