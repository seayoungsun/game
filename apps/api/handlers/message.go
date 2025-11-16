package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	messagesvc "github.com/kaifa/game-platform/internal/service/message"
	"gorm.io/gorm"
)

var (
	messageService messagesvc.Service
)

// SetMessageService 注入消息服务实现
func SetMessageService(service messagesvc.Service) {
	messageService = service
}

func ensureMessageService(c *gin.Context) bool {
	if messageService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "消息服务未初始化"})
		return false
	}
	return true
}

// GetUserMessages 获取当前用户的消息列表
func GetUserMessages(c *gin.Context) {
	if !ensureMessageService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 搜索条件
	msgType := c.Query("type")
	var isRead *bool
	if isReadStr := c.Query("is_read"); isReadStr != "" {
		val := isReadStr == "true"
		isRead = &val
	}

	// ✅ 使用 MessageService
	messages, total, err := messageService.GetUserMessages(c.Request.Context(), userID.(uint), msgType, isRead, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取消息列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  messages,
			"total": total,
		},
	})
}

// GetUnreadMessageCount 获取未读消息数量
func GetUnreadMessageCount(c *gin.Context) {
	if !ensureMessageService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	// ✅ 使用 MessageService
	count, err := messageService.GetUnreadCount(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取未读消息数量失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"count": count,
		},
	})
}

// ReadMessage 标记消息为已读
func ReadMessage(c *gin.Context) {
	if !ensureMessageService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的消息ID",
		})
		return
	}

	// ✅ 使用 MessageService
	message, err := messageService.ReadMessage(c.Request.Context(), uint(id), userID.(uint))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "消息不存在" {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "消息不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取消息失败: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "已标记为已读",
		"data":    message,
	})
}

// BatchReadMessages 批量标记消息为已读
func BatchReadMessages(c *gin.Context) {
	if !ensureMessageService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	var req struct {
		IDs []uint `json:"ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// ✅ 使用 MessageService（支持空数组 = 标记所有）
	if err := messageService.BatchReadMessages(c.Request.Context(), userID.(uint), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "批量标记失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "已标记为已读",
	})
}

// DeleteUserMessage 删除用户消息
func DeleteUserMessage(c *gin.Context) {
	if !ensureMessageService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的消息ID",
		})
		return
	}

	// ✅ 使用 MessageService
	if err := messageService.DeleteMessage(c.Request.Context(), uint(id), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除消息失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// GetAnnouncements 获取公告列表（用户端）
func GetAnnouncements(c *gin.Context) {
	if !ensureMessageService(c) {
		return
	}

	// ✅ 使用 MessageService
	announcements, err := messageService.GetAnnouncements(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取公告列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": announcements,
	})
}
