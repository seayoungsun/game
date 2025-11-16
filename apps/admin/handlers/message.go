package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

// GetAnnouncements 获取公告列表
func GetAnnouncements(c *gin.Context) {
	var announcements []models.Announcement
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := database.DB.Model(&models.Announcement{})

	// 搜索条件
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	query.Count(&total)

	// 获取列表
	if err := query.Offset(offset).Limit(pageSize).Order("priority DESC, created_at DESC").Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取公告列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  announcements,
			"total": total,
		},
	})
}

// GetAnnouncement 获取公告详情
func GetAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的公告ID",
		})
		return
	}

	var announcement models.Announcement
	if err := database.DB.First(&announcement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "公告不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取公告详情失败: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": announcement,
	})
}

// CreateAnnouncement 创建公告
func CreateAnnouncement(c *gin.Context) {
	adminID, _ := c.Get("admin_id")

	var req struct {
		Title       string `json:"title" binding:"required"`
		Content     string `json:"content" binding:"required"`
		Type        string `json:"type"`
		Priority    int    `json:"priority"`
		Status      int    `json:"status"`
		StartTime   *int64 `json:"start_time"`
		EndTime     *int64 `json:"end_time"`
		TargetUsers string `json:"target_users"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if req.Type == "" {
		req.Type = "info"
	}
	if req.Status == 0 {
		req.Status = 1
	}
	if req.TargetUsers == "" {
		req.TargetUsers = "all"
	}

	now := time.Now().Unix()
	announcement := models.Announcement{
		Title:       req.Title,
		Content:     req.Content,
		Type:        req.Type,
		Priority:    req.Priority,
		Status:      req.Status,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		TargetUsers: req.TargetUsers,
		CreatedBy:   adminID.(uint),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := database.DB.Create(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建公告失败: " + err.Error(),
		})
		return
	}

	// 如果公告已发布，发送给目标用户
	if req.Status == 1 {
		sendAnnouncementToUsers(&announcement)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    announcement,
	})
}

// UpdateAnnouncement 更新公告
func UpdateAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的公告ID",
		})
		return
	}

	var announcement models.Announcement
	if err := database.DB.First(&announcement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "公告不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取公告失败: " + err.Error(),
			})
		}
		return
	}

	var req struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		Type        string `json:"type"`
		Priority    int    `json:"priority"`
		Status      int    `json:"status"`
		StartTime   *int64 `json:"start_time"`
		EndTime     *int64 `json:"end_time"`
		TargetUsers string `json:"target_users"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	oldStatus := announcement.Status

	// 更新字段
	if req.Title != "" {
		announcement.Title = req.Title
	}
	if req.Content != "" {
		announcement.Content = req.Content
	}
	if req.Type != "" {
		announcement.Type = req.Type
	}
	if req.Priority != 0 || req.Priority == -1 {
		announcement.Priority = req.Priority
	}
	if req.Status != 0 {
		announcement.Status = req.Status
	}
	if req.StartTime != nil {
		announcement.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		announcement.EndTime = req.EndTime
	}
	if req.TargetUsers != "" {
		announcement.TargetUsers = req.TargetUsers
	}
	announcement.UpdatedAt = time.Now().Unix()

	if err := database.DB.Save(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新公告失败: " + err.Error(),
		})
		return
	}

	// 如果状态从未发布变为已发布，发送给目标用户
	if oldStatus != 1 && announcement.Status == 1 {
		sendAnnouncementToUsers(&announcement)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    announcement,
	})
}

// DeleteAnnouncement 删除公告
func DeleteAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的公告ID",
		})
		return
	}

	if err := database.DB.Delete(&models.Announcement{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除公告失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// sendAnnouncementToUsers 发送公告给目标用户
func sendAnnouncementToUsers(announcement *models.Announcement) {
	now := time.Now().Unix()

	// 检查时间范围
	if announcement.StartTime != nil && *announcement.StartTime > now {
		return // 未到开始时间
	}
	if announcement.EndTime != nil && *announcement.EndTime < now {
		return // 已过结束时间
	}

	var userIDs []uint

	if announcement.TargetUsers == "all" {
		// 发送给所有用户
		database.DB.Model(&models.User{}).Pluck("id", &userIDs)
	} else {
		// 发送给指定用户
		ids := strings.Split(announcement.TargetUsers, ",")
		for _, idStr := range ids {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
				userIDs = append(userIDs, uint(id))
			}
		}
	}

	if len(userIDs) == 0 {
		return
	}

	// 批量创建用户消息
	messages := make([]models.UserMessage, 0, len(userIDs))
	for _, userID := range userIDs {
		messages = append(messages, models.UserMessage{
			UserID:    userID,
			Type:      "system",
			Title:     announcement.Title,
			Content:   announcement.Content,
			IsRead:    false,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 分批插入（每批1000条）
	batchSize := 1000
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		database.DB.CreateInBatches(messages[i:end], batchSize)
	}
}

// GetUserMessages 获取用户消息列表（管理员查看所有消息）
func GetUserMessages(c *gin.Context) {
	var messages []models.UserMessage
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := database.DB.Model(&models.UserMessage{})

	// 搜索条件
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if msgType := c.Query("type"); msgType != "" {
		query = query.Where("type = ?", msgType)
	}
	if isRead := c.Query("is_read"); isRead != "" {
		query = query.Where("is_read = ?", isRead == "true")
	}

	// 获取总数
	query.Count(&total)

	// 获取列表
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&messages).Error; err != nil {
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

// SendUserMessage 发送用户消息（管理员操作）
func SendUserMessage(c *gin.Context) {
	var req struct {
		UserIDs   []uint `json:"user_ids" binding:"required"`
		Type      string `json:"type"`
		Title     string `json:"title" binding:"required"`
		Content   string `json:"content" binding:"required"`
		RelatedID string `json:"related_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if req.Type == "" {
		req.Type = "info"
	}

	now := time.Now().Unix()
	messages := make([]models.UserMessage, 0, len(req.UserIDs))

	for _, userID := range req.UserIDs {
		messages = append(messages, models.UserMessage{
			UserID:    userID,
			Type:      req.Type,
			Title:     req.Title,
			Content:   req.Content,
			RelatedID: req.RelatedID,
			IsRead:    false,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// 分批插入
	batchSize := 1000
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		if err := database.DB.CreateInBatches(messages[i:end], batchSize).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "发送消息失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "发送成功",
		"data": gin.H{
			"count": len(messages),
		},
	})
}

// DeleteUserMessage 删除用户消息
func DeleteUserMessage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的消息ID",
		})
		return
	}

	if err := database.DB.Delete(&models.UserMessage{}, id).Error; err != nil {
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

// BatchDeleteUserMessages 批量删除用户消息
func BatchDeleteUserMessages(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if err := database.DB.Where("id IN ?", req.IDs).Delete(&models.UserMessage{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "批量删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}
