package mysql

import (
	"context"
	"time"

	messagerepo "github.com/kaifa/game-platform/internal/repository/message"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) messagerepo.Repository {
	return &MessageRepository{db: db}
}

// GetUserMessages 获取用户消息列表
func (r *MessageRepository) GetUserMessages(ctx context.Context, userID uint, msgType string, isRead *bool, offset, limit int) ([]models.UserMessage, int64, error) {
	var messages []models.UserMessage
	var total int64

	query := r.db.WithContext(ctx).Model(&models.UserMessage{}).Where("user_id = ?", userID)

	// 搜索条件
	if msgType != "" {
		query = query.Where("type = ?", msgType)
	}
	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// GetUnreadCount 获取未读消息数量
func (r *MessageRepository) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserMessage{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// GetByID 根据ID和用户ID获取消息
func (r *MessageRepository) GetByID(ctx context.Context, id, userID uint) (*models.UserMessage, error) {
	var message models.UserMessage
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

// MarkAsRead 标记消息为已读
func (r *MessageRepository) MarkAsRead(ctx context.Context, id, userID uint) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&models.UserMessage{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// BatchMarkAsRead 批量标记消息为已读
func (r *MessageRepository) BatchMarkAsRead(ctx context.Context, userID uint, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&models.UserMessage{}).
		Where("id IN ? AND user_id = ?", ids, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// MarkAllAsRead 标记用户所有消息为已读
func (r *MessageRepository) MarkAllAsRead(ctx context.Context, userID uint) error {
	now := time.Now().Unix()
	return r.db.WithContext(ctx).Model(&models.UserMessage{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// Delete 删除用户消息
func (r *MessageRepository) Delete(ctx context.Context, id, userID uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.UserMessage{}).Error
}

// Create 创建消息
func (r *MessageRepository) Create(ctx context.Context, message *models.UserMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetAnnouncements 获取有效的公告列表
func (r *MessageRepository) GetAnnouncements(ctx context.Context, limit int) ([]models.Announcement, error) {
	var announcements []models.Announcement
	now := time.Now().Unix()

	query := r.db.WithContext(ctx).Model(&models.Announcement{}).
		Where("status = ?", 1) // 只获取已发布的公告

	// 时间范围筛选
	query = query.Where("(start_time IS NULL OR start_time <= ?) AND (end_time IS NULL OR end_time >= ?)", now, now)

	// 获取列表
	if err := query.Order("priority DESC, created_at DESC").Limit(limit).Find(&announcements).Error; err != nil {
		return nil, err
	}

	return announcements, nil
}

var _ messagerepo.Repository = (*MessageRepository)(nil)
