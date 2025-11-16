package message

import (
	"context"

	"github.com/kaifa/game-platform/pkg/models"
)

// Repository 定义消息数据访问接口
type Repository interface {
	// GetUserMessages 获取用户消息列表
	GetUserMessages(ctx context.Context, userID uint, msgType string, isRead *bool, offset, limit int) ([]models.UserMessage, int64, error)

	// GetUnreadCount 获取未读消息数量
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)

	// GetByID 根据ID和用户ID获取消息
	GetByID(ctx context.Context, id, userID uint) (*models.UserMessage, error)

	// MarkAsRead 标记消息为已读
	MarkAsRead(ctx context.Context, id, userID uint) error

	// BatchMarkAsRead 批量标记消息为已读
	BatchMarkAsRead(ctx context.Context, userID uint, ids []uint) error

	// MarkAllAsRead 标记用户所有消息为已读
	MarkAllAsRead(ctx context.Context, userID uint) error

	// Delete 删除用户消息
	Delete(ctx context.Context, id, userID uint) error

	// Create 创建消息
	Create(ctx context.Context, message *models.UserMessage) error

	// GetAnnouncements 获取有效的公告列表
	GetAnnouncements(ctx context.Context, limit int) ([]models.Announcement, error)
}
