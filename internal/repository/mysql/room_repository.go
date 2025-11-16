package mysql

import (
	"context"

	roomrepo "github.com/kaifa/game-platform/internal/repository/room"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

// RoomRepository MySQL 实现。
type RoomRepository struct {
	db *gorm.DB
}

// NewRoomRepository 创建房间仓储实例。
func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(ctx context.Context, room *models.GameRoom) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *RoomRepository) Update(ctx context.Context, room *models.GameRoom) error {
	return r.db.WithContext(ctx).Save(room).Error
}

func (r *RoomRepository) DeleteByRoomID(ctx context.Context, roomID string) error {
	return r.db.WithContext(ctx).Where("room_id = ?", roomID).Delete(&models.GameRoom{}).Error
}

func (r *RoomRepository) GetByRoomID(ctx context.Context, roomID string) (*models.GameRoom, error) {
	var room models.GameRoom
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *RoomRepository) List(ctx context.Context, filter roomrepo.ListFilter) ([]*models.GameRoom, error) {
	query := r.db.WithContext(ctx).Model(&models.GameRoom{})

	if filter.GameType != "" {
		query = query.Where("game_type = ?", filter.GameType)
	}
	if filter.Status > 0 {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OwnerID > 0 {
		query = query.Where("creator_id = ?", filter.OwnerID)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var rooms []*models.GameRoom
	if err := query.Order("created_at DESC").Limit(limit).Offset(filter.Offset).Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

var _ roomrepo.Repository = (*RoomRepository)(nil)
