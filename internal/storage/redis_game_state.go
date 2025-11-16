package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kaifa/game-platform/pkg/models"
	"github.com/redis/go-redis/v9"
)

// RedisGameStateStorage Redis实现的游戏状态存储
type RedisGameStateStorage struct {
	redis *redis.Client
}

// NewRedisGameStateStorage 创建Redis游戏状态存储实例
func NewRedisGameStateStorage(redisClient *redis.Client) GameStateStorage {
	return &RedisGameStateStorage{
		redis: redisClient,
	}
}

// Get 获取游戏状态
func (r *RedisGameStateStorage) Get(ctx context.Context, roomID string) (*models.GameState, error) {
	if r.redis == nil {
		return nil, errors.New("Redis客户端未初始化")
	}

	key := fmt.Sprintf("game:%s", roomID)
	data, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("游戏状态不存在")
		}
		return nil, fmt.Errorf("获取游戏状态失败: %w", err)
	}

	var gameState models.GameState
	if err := gameState.FromJSON([]byte(data)); err != nil {
		return nil, fmt.Errorf("解析游戏状态失败: %w", err)
	}

	return &gameState, nil
}

// Save 保存游戏状态
func (r *RedisGameStateStorage) Save(ctx context.Context, state *models.GameState, expiration time.Duration) error {
	if r.redis == nil {
		return errors.New("Redis客户端未初始化")
	}

	key := fmt.Sprintf("game:%s", state.RoomID)
	data, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("序列化游戏状态失败: %w", err)
	}

	if err := r.redis.Set(ctx, key, string(data), expiration).Err(); err != nil {
		return fmt.Errorf("保存游戏状态失败: %w", err)
	}

	return nil
}

// Delete 删除游戏状态
func (r *RedisGameStateStorage) Delete(ctx context.Context, roomID string) error {
	if r.redis == nil {
		return errors.New("Redis客户端未初始化")
	}

	key := fmt.Sprintf("game:%s", roomID)
	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除游戏状态失败: %w", err)
	}

	return nil
}

// Exists 检查游戏状态是否存在
func (r *RedisGameStateStorage) Exists(ctx context.Context, roomID string) (bool, error) {
	if r.redis == nil {
		return false, errors.New("Redis客户端未初始化")
	}

	key := fmt.Sprintf("game:%s", roomID)
	count, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("检查游戏状态失败: %w", err)
	}

	return count > 0, nil
}
