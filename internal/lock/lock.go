package lock

import (
	"context"
	"time"
)

// Lock 定义分布式锁接口
// 支持 Redis 分布式锁、本地内存锁等多种实现
type Lock interface {
	// TryLock 尝试获取锁（非阻塞）
	// key: 锁的唯一标识
	// ttl: 锁的过期时间（防止死锁）
	// 返回：是否成功获取锁
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Lock 获取锁（阻塞，带重试）
	// maxRetries: 最大重试次数
	// retryInterval: 重试间隔
	Lock(ctx context.Context, key string, ttl time.Duration, maxRetries int, retryInterval time.Duration) error

	// Unlock 释放锁
	Unlock(ctx context.Context, key string) error

	// WithLock 在锁保护下执行函数（推荐使用）
	// 自动获取锁、执行函数、释放锁
	// 如果获取锁失败，返回错误
	WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error

	// Refresh 刷新锁的过期时间（用于长时间操作）
	Refresh(ctx context.Context, key string, ttl time.Duration) error
}

// RWLock 定义读写锁接口（用于本地锁）
type RWLock interface {
	// RLock 获取读锁（允许多个并发读）
	RLock(key string)

	// RUnlock 释放读锁
	RUnlock(key string)

	// Lock 获取写锁（独占）
	Lock(key string)

	// Unlock 释放写锁
	Unlock(key string)

	// WithRLock 在读锁保护下执行函数
	WithRLock(key string, fn func() error) error

	// WithLock 在写锁保护下执行函数
	WithLock(key string, fn func() error) error
}
