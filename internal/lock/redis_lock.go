package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/internal/metrics"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisLock Redis 分布式锁实现
// 适用于多实例部署场景，保证跨实例的数据一致性
type RedisLock struct {
	redis *redis.Client
}

// NewRedisLock 创建 Redis 分布式锁实例
func NewRedisLock(redisClient *redis.Client) Lock {
	return &RedisLock{
		redis: redisClient,
	}
}

// TryLock 尝试获取锁（非阻塞）
func (l *RedisLock) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if l.redis == nil {
		return false, errors.New("Redis 客户端未初始化")
	}

	lockKey := fmt.Sprintf("lock:%s", key)
	lockValue := time.Now().UnixNano() // 使用时间戳作为锁的值

	// 使用 SET NX（不存在才设置）实现锁
	success, err := l.redis.SetNX(ctx, lockKey, lockValue, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("获取锁失败: %w", err)
	}

	if success {
		logger.Logger.Debug("成功获取分布式锁",
			zap.String("key", key),
			zap.Duration("ttl", ttl),
		)
	}

	return success, nil
}

// Lock 获取锁（阻塞，带重试）
func (l *RedisLock) Lock(ctx context.Context, key string, ttl time.Duration, maxRetries int, retryInterval time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		success, err := l.TryLock(ctx, key, ttl)
		if err != nil {
			return err
		}

		if success {
			return nil // 成功获取锁
		}

		// 未获取到锁，等待后重试
		if i < maxRetries-1 {
			select {
			case <-time.After(retryInterval):
				// 继续重试
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return errors.New("获取锁超时，请稍后重试")
}

// Unlock 释放锁
func (l *RedisLock) Unlock(ctx context.Context, key string) error {
	if l.redis == nil {
		return errors.New("Redis 客户端未初始化")
	}

	lockKey := fmt.Sprintf("lock:%s", key)

	// 删除锁
	err := l.redis.Del(ctx, lockKey).Err()
	if err != nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}

	logger.Logger.Debug("成功释放分布式锁",
		zap.String("key", key),
	)

	return nil
}

// WithLock 在锁保护下执行函数（带监控）
func (l *RedisLock) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	// ✅ 记录开始时间（用于监控等待时间）
	startTime := time.Now()

	// 默认重试3次，间隔50ms
	err := l.Lock(ctx, key, ttl, 3, 50*time.Millisecond)

	// ✅ 记录等待时间
	waitTime := time.Since(startTime)
	success := err == nil
	metrics.GetGlobalMetrics().RecordLockAcquire(key, success, waitTime)

	if err != nil {
		return fmt.Errorf("获取锁失败: %w", err)
	}

	// ✅ 记录持有锁的开始时间
	holdStartTime := time.Now()

	// 确保释放锁
	defer func() {
		// ✅ 记录持有时间
		holdTime := time.Since(holdStartTime)
		metrics.GetGlobalMetrics().RecordLockRelease(key, holdTime)

		if err := l.Unlock(context.Background(), key); err != nil {
			logger.Logger.Error("释放锁失败",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}()

	// 执行业务逻辑
	return fn()
}

// Refresh 刷新锁的过期时间
func (l *RedisLock) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	if l.redis == nil {
		return errors.New("Redis 客户端未初始化")
	}

	lockKey := fmt.Sprintf("lock:%s", key)

	// 延长锁的过期时间
	err := l.redis.Expire(ctx, lockKey, ttl).Err()
	if err != nil {
		return fmt.Errorf("刷新锁失败: %w", err)
	}

	return nil
}
