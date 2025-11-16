package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	usersvc "github.com/kaifa/game-platform/internal/service/user"
	userstatssvc "github.com/kaifa/game-platform/internal/service/userstats"
)

var (
	userService      usersvc.Service
	userStatsService userstatssvc.Service
)

// SetUserService 注入用户服务实现
func SetUserService(service usersvc.Service) {
	userService = service
}

// SetUserStatsService 注入用户统计服务实现
func SetUserStatsService(service userstatssvc.Service) {
	userStatsService = service
}

func ensureUserService(c *gin.Context) bool {
	if userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户服务未初始化"})
		return false
	}
	return true
}

func ensureUserStatsService(c *gin.Context) bool {
	if userStatsService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户统计服务未初始化"})
		return false
	}
	return true
}

// Register 用户注册
func Register(c *gin.Context) {
	if !ensureUserService(c) {
		return
	}
	var req usersvc.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	user, token, err := userService.Register(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "注册成功",
		"data": gin.H{
			"user":  user,
			"token": token,
		},
	})
}

// Login 用户登录
func Login(c *gin.Context) {
	if !ensureUserService(c) {
		return
	}
	var req usersvc.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	ip := c.ClientIP()
	user, token, err := userService.Login(c.Request.Context(), &req, ip)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data": gin.H{
			"user":  user,
			"token": token,
		},
	})
}

// Profile 获取用户信息
func Profile(c *gin.Context) {
	if !ensureUserService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	user, err := userService.GetUserByID(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": user})
}

// GetUserStats 获取用户游戏统计
func GetUserStats(c *gin.Context) {
	if !ensureUserStatsService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	// ✅ 使用新的 UserStatsService
	stats, err := userStatsService.GetUserStats(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    stats,
	})
}
