package message

import (
	"context"
	"errors"

	messagerepo "github.com/kaifa/game-platform/internal/repository/message"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

// Service 定义消息业务服务接口
type Service interface {
	// GetUserMessages 获取用户消息列表
	GetUserMessages(ctx context.Context, userID uint, msgType string, isRead *bool, page, pageSize int) ([]models.UserMessage, int64, error)

	// GetUnreadCount 获取未读消息数量
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)

	// ReadMessage 标记消息为已读
	ReadMessage(ctx context.Context, id, userID uint) (*models.UserMessage, error)

	// BatchReadMessages 批量标记消息为已读
	BatchReadMessages(ctx context.Context, userID uint, ids []uint) error

	// DeleteMessage 删除用户消息
	DeleteMessage(ctx context.Context, id, userID uint) error

	// GetAnnouncements 获取公告列表
	GetAnnouncements(ctx context.Context) ([]models.Announcement, error)
}

type service struct {
	repo messagerepo.Repository
}

// New 创建消息服务实例
func New(repo messagerepo.Repository) Service {
	return &service{
		repo: repo,
	}
}

// GetUserMessages 获取用户消息列表
func (s *service) GetUserMessages(ctx context.Context, userID uint, msgType string, isRead *bool, page, pageSize int) ([]models.UserMessage, int64, error) {
	// ✅ 业务逻辑：参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// ✅ 通过 Repository 查询
	return s.repo.GetUserMessages(ctx, userID, msgType, isRead, offset, pageSize)
}

// GetUnreadCount 获取未读消息数量
func (s *service) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	// ✅ 通过 Repository 查询
	return s.repo.GetUnreadCount(ctx, userID)
}

// ReadMessage 标记消息为已读
func (s *service) ReadMessage(ctx context.Context, id, userID uint) (*models.UserMessage, error) {
	// ✅ 通过 Repository 查询消息
	message, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("消息不存在")
		}
		return nil, err
	}

	// ✅ 业务逻辑：如果未读，标记为已读
	if !message.IsRead {
		if err := s.repo.MarkAsRead(ctx, id, userID); err != nil {
			return nil, err
		}
		// 重新获取消息（更新后的）
		message, _ = s.repo.GetByID(ctx, id, userID)
	}

	return message, nil
}

// BatchReadMessages 批量标记消息为已读
func (s *service) BatchReadMessages(ctx context.Context, userID uint, ids []uint) error {
	// ✅ 业务逻辑：如果没有指定ID，标记所有消息为已读
	if len(ids) == 0 {
		return s.repo.MarkAllAsRead(ctx, userID)
	}

	// ✅ 通过 Repository 批量标记
	return s.repo.BatchMarkAsRead(ctx, userID, ids)
}

// DeleteMessage 删除用户消息
func (s *service) DeleteMessage(ctx context.Context, id, userID uint) error {
	// ✅ 通过 Repository 删除
	return s.repo.Delete(ctx, id, userID)
}

// GetAnnouncements 获取公告列表
func (s *service) GetAnnouncements(ctx context.Context) ([]models.Announcement, error) {
	// ✅ 业务逻辑：最多返回20条
	return s.repo.GetAnnouncements(ctx, 20)
}
