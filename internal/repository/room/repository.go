package room

import (
	"context"

	"github.com/kaifa/game-platform/pkg/models"
)

// Repository 定义房间相关的数据访问接口。
// 后续将把 pkg/services/room_service.go 中直接依赖数据库/Redis 的逻辑迁移到具体实现中。
// 当前仅作为解耦骨架，不参与实际业务调用。
type Repository interface {
	Create(ctx context.Context, room *models.GameRoom) error
	Update(ctx context.Context, room *models.GameRoom) error
	DeleteByRoomID(ctx context.Context, roomID string) error
	GetByRoomID(ctx context.Context, roomID string) (*models.GameRoom, error)
	List(ctx context.Context, filter ListFilter) ([]*models.GameRoom, error)
}

// ListFilter 描述房间列表查询的筛选条件。
type ListFilter struct {
	GameType string
	Status   int8
	OwnerID  uint
	Limit    int
	Offset   int
}
