package mysql

import (
	"context"
	"encoding/json"
	"fmt"

	gameplayerrepo "github.com/kaifa/game-platform/internal/repository/gameplayer"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

type GamePlayerRepository struct {
	db *gorm.DB
}

func NewGamePlayerRepository(db *gorm.DB) gameplayerrepo.Repository {
	return &GamePlayerRepository{db: db}
}

// GetGameTypeStats 获取指定游戏类型的统计数据
func (r *GamePlayerRepository) GetGameTypeStats(ctx context.Context, userID uint, gameType string) (*gameplayerrepo.GameTypeStats, error) {
	stats := &gameplayerrepo.GameTypeStats{
		GameType: gameType,
	}

	// 查询总游戏数
	var totalGames int64
	if err := r.db.WithContext(ctx).Table("game_players").
		Select("COUNT(DISTINCT game_players.room_id)").
		Joins("JOIN game_records ON game_players.room_id = game_records.room_id").
		Where("game_players.user_id = ? AND game_records.game_type = ?", userID, gameType).
		Count(&totalGames).Error; err != nil {
		return nil, err
	}
	stats.TotalGames = int(totalGames)

	if stats.TotalGames == 0 {
		return stats, nil
	}

	// 查询总余额变化
	var totalBalance float64
	if err := r.db.WithContext(ctx).Table("game_players").
		Select("COALESCE(SUM(game_players.balance), 0)").
		Joins("JOIN game_records ON game_players.room_id = game_records.room_id").
		Where("game_players.user_id = ? AND game_records.game_type = ?", userID, gameType).
		Scan(&totalBalance).Error; err != nil {
		return nil, err
	}
	stats.TotalBalance = totalBalance

	// 查询获胜次数（通过解析 JSON）
	stats.Wins = r.calculateWins(ctx, userID, gameType)

	// 计算失败次数和胜率
	stats.Losses = stats.TotalGames - stats.Wins
	if stats.TotalGames > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalGames) * 100
	}

	return stats, nil
}

// GetTotalStats 获取用户总统计数据
func (r *GamePlayerRepository) GetTotalStats(ctx context.Context, userID uint) (*gameplayerrepo.TotalStats, error) {
	stats := &gameplayerrepo.TotalStats{}

	// 查询总游戏数
	var totalGames int64
	if err := r.db.WithContext(ctx).Table("game_players").
		Select("COUNT(DISTINCT room_id)").
		Where("user_id = ?", userID).
		Count(&totalGames).Error; err != nil {
		return nil, err
	}
	stats.TotalGames = int(totalGames)

	if totalGames == 0 {
		return stats, nil
	}

	// 查询总余额变化
	var totalBalance float64
	if err := r.db.WithContext(ctx).Table("game_players").
		Select("COALESCE(SUM(balance), 0)").
		Where("user_id = ?", userID).
		Scan(&totalBalance).Error; err != nil {
		return nil, err
	}
	stats.TotalBalance = totalBalance

	// 查询总获胜次数（需要解析JSON）
	stats.TotalWins = r.calculateTotalWins(ctx, userID)

	// 计算胜率和失败次数
	stats.TotalLosses = stats.TotalGames - stats.TotalWins
	if stats.TotalGames > 0 {
		stats.WinRate = float64(stats.TotalWins) / float64(stats.TotalGames) * 100
	}

	return stats, nil
}

// GetUserGameRecords 获取用户参与的游戏记录
func (r *GamePlayerRepository) GetUserGameRecords(ctx context.Context, userID uint, gameType string) ([]models.GameRecord, error) {
	var records []models.GameRecord

	query := r.db.WithContext(ctx).
		Joins("JOIN game_players ON game_records.room_id = game_players.room_id").
		Where("game_players.user_id = ?", userID)

	if gameType != "" {
		query = query.Where("game_records.game_type = ?", gameType)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

// GetGamePlayer 获取玩家在指定游戏中的记录
func (r *GamePlayerRepository) GetGamePlayer(ctx context.Context, roomID string, userID uint) (*models.GamePlayer, error) {
	var player models.GamePlayer
	if err := r.db.WithContext(ctx).Where("room_id = ? AND user_id = ?", roomID, userID).First(&player).Error; err != nil {
		return nil, err
	}
	return &player, nil
}

// calculateWins 计算获胜次数（从 JSON 中解析名次）
func (r *GamePlayerRepository) calculateWins(ctx context.Context, userID uint, gameType string) int {
	records, err := r.GetUserGameRecords(ctx, userID, gameType)
	if err != nil {
		return 0
	}

	wins := 0
	for _, record := range records {
		// 检查用户是否参与该游戏
		_, err := r.GetGamePlayer(ctx, record.RoomID, userID)
		if err != nil {
			continue
		}

		// 解析结算结果，查找用户的排名
		if len(record.Result) > 0 {
			var resultData map[string]interface{}
			if err := json.Unmarshal(record.Result, &resultData); err == nil {
				if userResult, ok := resultData[fmt.Sprintf("%d", userID)].(map[string]interface{}); ok {
					if rank, ok := userResult["rank"].(float64); ok && rank == 1 {
						wins++
					}
				}
			}
		}
	}

	return wins
}

// calculateTotalWins 计算总获胜次数
func (r *GamePlayerRepository) calculateTotalWins(ctx context.Context, userID uint) int {
	records, err := r.GetUserGameRecords(ctx, userID, "")
	if err != nil {
		return 0
	}

	wins := 0
	for _, record := range records {
		// 检查用户是否参与
		_, err := r.GetGamePlayer(ctx, record.RoomID, userID)
		if err != nil {
			continue
		}

		// 解析结算结果
		if len(record.Result) > 0 {
			var resultData map[string]interface{}
			if err := json.Unmarshal(record.Result, &resultData); err == nil {
				if userResult, ok := resultData[fmt.Sprintf("%d", userID)].(map[string]interface{}); ok {
					if rank, ok := userResult["rank"].(float64); ok && rank == 1 {
						wins++
					}
				}
			}
		}
	}

	return wins
}

var _ gameplayerrepo.Repository = (*GamePlayerRepository)(nil)
