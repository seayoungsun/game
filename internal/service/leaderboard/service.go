package leaderboard

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	userrepo "github.com/kaifa/game-platform/internal/repository/user"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	UpdateLeaderboard(ctx context.Context, gameType string, scores map[uint]float64) error
	GetLeaderboard(ctx context.Context, gameType, period string, page, pageSize int) (*LeaderboardResponse, error)
	GetUserRank(ctx context.Context, gameType, period string, userID uint) (int, float64, error)
}

type service struct {
	redis    *redis.Client
	userRepo userrepo.Repository
}

func New(redisClient *redis.Client, userRepo userrepo.Repository) Service {
	return &service{redis: redisClient, userRepo: userRepo}
}

func (s *service) UpdateLeaderboard(ctx context.Context, gameType string, scores map[uint]float64) error {
	if s.redis == nil || len(scores) == 0 {
		return nil
	}
	for userID, score := range scores {
		member := fmt.Sprintf("%d", userID)
		totalKey := fmt.Sprintf("leaderboard:%s:total", gameType)
		if err := s.redis.ZAdd(ctx, totalKey, redis.Z{Member: member, Score: score}).Err(); err != nil {
			return fmt.Errorf("更新总榜失败: %w", err)
		}
		dayKey := fmt.Sprintf("leaderboard:%s:day:%s", gameType, time.Now().Format("2006-01-02"))
		if err := s.redis.ZAdd(ctx, dayKey, redis.Z{Member: member, Score: score}).Err(); err != nil {
			return fmt.Errorf("更新日榜失败: %w", err)
		}
		_ = s.redis.Expire(ctx, dayKey, 7*24*time.Hour)
		weekStart := getWeekStart(time.Now())
		weekKey := fmt.Sprintf("leaderboard:%s:week:%s", gameType, weekStart.Format("2006-01-02"))
		if err := s.redis.ZAdd(ctx, weekKey, redis.Z{Member: member, Score: score}).Err(); err != nil {
			return fmt.Errorf("更新周榜失败: %w", err)
		}
		_ = s.redis.Expire(ctx, weekKey, 30*24*time.Hour)
		monthKey := fmt.Sprintf("leaderboard:%s:month:%s", gameType, time.Now().Format("2006-01"))
		if err := s.redis.ZAdd(ctx, monthKey, redis.Z{Member: member, Score: score}).Err(); err != nil {
			return fmt.Errorf("更新月榜失败: %w", err)
		}
		_ = s.redis.Expire(ctx, monthKey, 90*24*time.Hour)
	}
	return nil
}

func (s *service) GetLeaderboard(ctx context.Context, gameType, period string, page, pageSize int) (*LeaderboardResponse, error) {
	if s.redis == nil {
		return nil, errors.New("排行榜功能未启用")
	}
	key, err := leaderboardKey(gameType, period)
	if err != nil {
		return nil, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1
	members, err := s.redis.ZRevRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("查询排行榜失败: %w", err)
	}
	total, err := s.redis.ZCard(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}
	resp := &LeaderboardResponse{
		GameType: gameType,
		Period:   period,
		Page:     page,
		PageSize: pageSize,
		Total:    int(total),
		Rankings: make([]RankingItem, 0, len(members)),
	}
	baseRank := int(start) + 1
	for i, member := range members {
		memberStr, ok := member.Member.(string)
		if !ok {
			continue
		}
		userID, err := strconv.ParseUint(memberStr, 10, 32)
		if err != nil {
			continue
		}
		var user models.User
		if s.userRepo != nil {
			if u, err := s.userRepo.GetByID(ctx, uint(userID)); err == nil {
				user = *u
			}
		}
		item := RankingItem{
			Rank:     baseRank + i,
			UserID:   uint(userID),
			Score:    member.Score,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			UID:      user.UID,
		}
		resp.Rankings = append(resp.Rankings, item)
	}
	resp.MyRank = -1
	return resp, nil
}

func (s *service) GetUserRank(ctx context.Context, gameType, period string, userID uint) (int, float64, error) {
	if s.redis == nil {
		return -1, 0, errors.New("排行榜功能未启用")
	}
	key, err := leaderboardKey(gameType, period)
	if err != nil {
		return -1, 0, err
	}
	member := fmt.Sprintf("%d", userID)
	rank, err := s.redis.ZRevRank(ctx, key, member).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, 0, nil
		}
		return -1, 0, fmt.Errorf("查询用户排名失败: %w", err)
	}
	score, err := s.redis.ZScore(ctx, key, member).Result()
	if err != nil {
		return -1, 0, fmt.Errorf("查询用户分数失败: %w", err)
	}
	return int(rank) + 1, score, nil
}

type LeaderboardResponse struct {
	GameType string        `json:"game_type"`
	Period   string        `json:"period"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
	MyRank   int           `json:"my_rank"`
	Rankings []RankingItem `json:"rankings"`
}

type RankingItem struct {
	Rank     int     `json:"rank"`
	UserID   uint    `json:"user_id"`
	UID      int64   `json:"uid"`
	Nickname string  `json:"nickname"`
	Avatar   string  `json:"avatar"`
	Score    float64 `json:"score"`
}

func leaderboardKey(gameType, period string) (string, error) {
	switch period {
	case "total":
		return fmt.Sprintf("leaderboard:%s:total", gameType), nil
	case "day":
		return fmt.Sprintf("leaderboard:%s:day:%s", gameType, time.Now().Format("2006-01-02")), nil
	case "week":
		weekStart := getWeekStart(time.Now())
		return fmt.Sprintf("leaderboard:%s:week:%s", gameType, weekStart.Format("2006-01-02")), nil
	case "month":
		return fmt.Sprintf("leaderboard:%s:month:%s", gameType, time.Now().Format("2006-01")), nil
	default:
		return "", errors.New("无效的排行榜类型")
	}
}

func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}
