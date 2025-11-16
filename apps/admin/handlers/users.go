package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
)

// GetUsers 获取用户列表
func GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	status, _ := strconv.Atoi(c.Query("status"))

	var users []models.User
	query := database.DB.Model(&models.User{})

	if search != "" {
		query = query.Where("phone LIKE ? OR nickname LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	offset := (page - 1) * pageSize
	var total int64
	query.Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":       users,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetUserDetail 获取用户详情
func GetUserDetail(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": user,
	})
}

// UpdateUser 更新用户信息
func UpdateUser(c *gin.Context) {
	// 需要 admin:users:update 权限
	userID := c.Param("id")

	var req struct {
		Nickname string `json:"nickname"`
		Status   int8   `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "用户不存在",
		})
		return
	}

	updates := make(map[string]interface{})
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Status > 0 {
		updates["status"] = req.Status
	}

	if len(updates) > 0 {
		database.DB.Model(&user).Updates(updates)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
	})
}
