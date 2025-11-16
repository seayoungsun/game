package mysql

import (
	"context"

	gamerecordrepo "github.com/kaifa/game-platform/internal/repository/gamerecord"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

type GameRecordRepository struct {
	db *gorm.DB
}

func NewGameRecordRepository(db *gorm.DB) *GameRecordRepository {
	return &GameRecordRepository{db: db}
}

func (r *GameRecordRepository) ListRoomIDsByUser(ctx context.Context, userID uint) ([]string, error) {
	var roomIDs []string
	query := r.db.WithContext(ctx).Table("game_players").Select("room_id").Where("user_id = ?", userID)
	if err := query.Pluck("room_id", &roomIDs).Error; err != nil {
		return nil, err
	}
	return roomIDs, nil
}

func (r *GameRecordRepository) CountRecordsByRoomIDs(ctx context.Context, roomIDs []string, gameType string) (int64, error) {
	query := r.db.WithContext(ctx).Table("game_records").Where("room_id IN ?", roomIDs)
	if gameType != "" {
		query = query.Where("game_type = ?", gameType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *GameRecordRepository) ListRecordsByRoomIDs(ctx context.Context, roomIDs []string, gameType string, offset, limit int) ([]models.GameRecord, error) {
	query := r.db.WithContext(ctx).Where("room_id IN ?", roomIDs)
	if gameType != "" {
		query = query.Where("game_type = ?", gameType)
	}
	var records []models.GameRecord
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *GameRecordRepository) GetRecordByID(ctx context.Context, recordID uint) (*models.GameRecord, error) {
	var record models.GameRecord
	if err := r.db.WithContext(ctx).First(&record, recordID).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *GameRecordRepository) ListRecordsByRoom(ctx context.Context, roomID string) ([]models.GameRecord, error) {
	var records []models.GameRecord
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *GameRecordRepository) GetPlayerInRoom(ctx context.Context, roomID string, userID uint) (*models.GamePlayer, error) {
	var player models.GamePlayer
	if err := r.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, userID).First(&player).Error; err != nil {
		return nil, err
	}
	return &player, nil
}

func (r *GameRecordRepository) ListPlayersByRoom(ctx context.Context, roomID string) ([]models.GamePlayer, error) {
	var players []models.GamePlayer
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Find(&players).Error; err != nil {
		return nil, err
	}
	return players, nil
}

func (r *GameRecordRepository) GetRoomByRoomID(ctx context.Context, roomID string) (*models.GameRoom, error) {
	var room models.GameRoom
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// CreateGameRecord 创建游戏记录
func (r *GameRecordRepository) CreateGameRecord(ctx context.Context, record *models.GameRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// CreateGamePlayer 创建玩家对局记录
func (r *GameRecordRepository) CreateGamePlayer(ctx context.Context, player *models.GamePlayer) error {
	return r.db.WithContext(ctx).Create(player).Error
}

// BatchCreateGamePlayers 批量创建玩家对局记录
func (r *GameRecordRepository) BatchCreateGamePlayers(ctx context.Context, players []*models.GamePlayer) error {
	if len(players) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(players).Error
}

var _ gamerecordrepo.Repository = (*GameRecordRepository)(nil)
