package gameplayer

import (
	"context"

	"github.com/kaifa/game-platform/pkg/models"
)

// GameTypeStats 游戏类型统计数据
type GameTypeStats struct {
	GameType     string  `json:"game_type"`
	TotalGames   int     `json:"total_games"`
	Wins         int     `json:"wins"`
	Losses       int     `json:"losses"`
	WinRate      float64 `json:"win_rate"`
	TotalBalance float64 `json:"total_balance"`
}

// TotalStats 总统计数据
type TotalStats struct {
	TotalGames   int     `json:"total_games"`
	TotalWins    int     `json:"total_wins"`
	TotalLosses  int     `json:"total_losses"`
	WinRate      float64 `json:"win_rate"`
	TotalBalance float64 `json:"total_balance"`
}

// Repository 定义游戏玩家统计数据访问接口
type Repository interface {
	// GetGameTypeStats 获取指定游戏类型的统计数据
	GetGameTypeStats(ctx context.Context, userID uint, gameType string) (*GameTypeStats, error)

	// GetTotalStats 获取用户总统计数据
	GetTotalStats(ctx context.Context, userID uint) (*TotalStats, error)

	// GetUserGameRecords 获取用户参与的游戏记录（用于计算名次）
	GetUserGameRecords(ctx context.Context, userID uint, gameType string) ([]models.GameRecord, error)

	// GetGamePlayer 获取玩家在指定游戏中的记录
	GetGamePlayer(ctx context.Context, roomID string, userID uint) (*models.GamePlayer, error)
}
