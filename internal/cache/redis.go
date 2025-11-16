package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/kaifa/game-platform/internal/config"
	"github.com/redis/go-redis/v9"
)

var (
	RDB *redis.Client
	ctx = context.Background()
)

// InitRedis 初始化Redis连接
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	RDB = rdb
	return rdb, nil
}

// Close 关闭Redis连接
func Close() error {
	if RDB != nil {
		return RDB.Close()
	}
	return nil
}

// Set 设置键值对（带过期时间）
func Set(key string, value interface{}, expiration time.Duration) error {
	return RDB.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func Get(key string) (string, error) {
	return RDB.Get(ctx, key).Result()
}

// Del 删除键
func Del(keys ...string) error {
	return RDB.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(key string) (bool, error) {
	count, err := RDB.Exists(ctx, key).Result()
	return count > 0, err
}

// Expire 设置过期时间
func Expire(key string, expiration time.Duration) error {
	return RDB.Expire(ctx, key, expiration).Err()
}

// SetNX 设置键值对（仅当键不存在时）
func SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	return RDB.SetNX(ctx, key, value, expiration).Result()
}

// Increment 自增
func Increment(key string) (int64, error) {
	return RDB.Incr(ctx, key).Result()
}

// HSet 哈希表设置
func HSet(key, field string, value interface{}) error {
	return RDB.HSet(ctx, key, field, value).Err()
}

// HGet 哈希表获取
func HGet(key, field string) (string, error) {
	return RDB.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func HGetAll(key string) (map[string]string, error) {
	return RDB.HGetAll(ctx, key).Result()
}

// ZAdd 有序集合添加
func ZAdd(key string, score float64, member string) error {
	return RDB.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// ZRange 有序集合范围查询
func ZRange(key string, start, stop int64) ([]string, error) {
	return RDB.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange 有序集合倒序范围查询
func ZRevRange(key string, start, stop int64) ([]string, error) {
	return RDB.ZRevRange(ctx, key, start, stop).Result()
}

// ZRevRangeWithScores 有序集合倒序范围查询（带分数）
func ZRevRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	return RDB.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

// ZRevRank 获取成员在有序集合中的排名（从高到低）
func ZRevRank(key, member string) (int64, error) {
	rank, err := RDB.ZRevRank(ctx, key, member).Result()
	if err == redis.Nil {
		return -1, nil // 成员不存在
	}
	return rank, err
}

// ZScore 获取成员的分数
func ZScore(key, member string) (float64, error) {
	return RDB.ZScore(ctx, key, member).Result()
}

// ZCard 获取有序集合的成员数量
func ZCard(key string) (int64, error) {
	return RDB.ZCard(ctx, key).Result()
}
