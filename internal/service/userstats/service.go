package userstats

import (
	"context"

	gameplayerrepo "github.com/kaifa/game-platform/internal/repository/gameplayer"
)

// Service 定义用户统计业务服务接口
type Service interface {
	// GetUserStats 获取用户游戏统计
	GetUserStats(ctx context.Context, userID uint) (*UserStatsResponse, error)
}

type service struct {
	gamePlayerRepo gameplayerrepo.Repository
}

// New 创建用户统计服务实例
func New(gamePlayerRepo gameplayerrepo.Repository) Service {
	return &service{
		gamePlayerRepo: gamePlayerRepo,
	}
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	UserID uint                                    `json:"user_id"`
	Total  gameplayerrepo.TotalStats               `json:"total"` // 总统计
	Games  map[string]gameplayerrepo.GameTypeStats `json:"games"` // 各游戏类型统计
}

// GetUserStats 获取用户游戏统计
func (s *service) GetUserStats(ctx context.Context, userID uint) (*UserStatsResponse, error) {
	stats := &UserStatsResponse{
		UserID: userID,
		Games:  make(map[string]gameplayerrepo.GameTypeStats),
	}

	// ✅ 业务逻辑：获取所有游戏类型的统计
	gameTypes := []string{"running", "texas", "bull"}
	for _, gameType := range gameTypes {
		// ✅ 通过 Repository 查询
		gameStats, err := s.gamePlayerRepo.GetGameTypeStats(ctx, userID, gameType)
		if err != nil {
			continue
		}
		stats.Games[gameType] = *gameStats
	}

	// ✅ 通过 Repository 查询总统计
	totalStats, err := s.gamePlayerRepo.GetTotalStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.Total = *totalStats

	return stats, nil
}
