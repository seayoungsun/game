package gamerecord

import (
	"context"

	"github.com/kaifa/game-platform/pkg/models"
)

// Repository 定义游戏记录相关的数据访问接口。
type Repository interface {
	ListRoomIDsByUser(ctx context.Context, userID uint) ([]string, error)
	CountRecordsByRoomIDs(ctx context.Context, roomIDs []string, gameType string) (int64, error)
	ListRecordsByRoomIDs(ctx context.Context, roomIDs []string, gameType string, offset, limit int) ([]models.GameRecord, error)
	GetRecordByID(ctx context.Context, recordID uint) (*models.GameRecord, error)
	ListRecordsByRoom(ctx context.Context, roomID string) ([]models.GameRecord, error)
	GetPlayerInRoom(ctx context.Context, roomID string, userID uint) (*models.GamePlayer, error)
	ListPlayersByRoom(ctx context.Context, roomID string) ([]models.GamePlayer, error)
	GetRoomByRoomID(ctx context.Context, roomID string) (*models.GameRoom, error)

	// CreateGameRecord 创建游戏记录
	CreateGameRecord(ctx context.Context, record *models.GameRecord) error

	// CreateGamePlayer 创建玩家对局记录
	CreateGamePlayer(ctx context.Context, player *models.GamePlayer) error

	// BatchCreateGamePlayers 批量创建玩家对局记录
	BatchCreateGamePlayers(ctx context.Context, players []*models.GamePlayer) error
}
