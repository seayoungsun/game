package lock

import (
	"sync"
)

// LocalRWLock 本地读写锁实现
// 适用于单实例部署，基于 sync.RWMutex 实现
// 性能优于分布式锁，但不支持多实例
type LocalRWLock struct {
	locks sync.Map // key: string, value: *sync.RWMutex
}

// NewLocalRWLock 创建本地读写锁实例
func NewLocalRWLock() RWLock {
	return &LocalRWLock{}
}

// getLock 获取或创建指定 key 的锁
func (l *LocalRWLock) getLock(key string) *sync.RWMutex {
	lock, _ := l.locks.LoadOrStore(key, &sync.RWMutex{})
	return lock.(*sync.RWMutex)
}

// RLock 获取读锁
func (l *LocalRWLock) RLock(key string) {
	mu := l.getLock(key)
	mu.RLock()
}

// RUnlock 释放读锁
func (l *LocalRWLock) RUnlock(key string) {
	mu := l.getLock(key)
	mu.RUnlock()
}

// Lock 获取写锁
func (l *LocalRWLock) Lock(key string) {
	mu := l.getLock(key)
	mu.Lock()
}

// Unlock 释放写锁
func (l *LocalRWLock) Unlock(key string) {
	mu := l.getLock(key)
	mu.Unlock()
}

// WithRLock 在读锁保护下执行函数
func (l *LocalRWLock) WithRLock(key string, fn func() error) error {
	l.RLock(key)
	defer l.RUnlock(key)
	return fn()
}

// WithLock 在写锁保护下执行函数
func (l *LocalRWLock) WithLock(key string, fn func() error) error {
	l.Lock(key)
	defer l.Unlock(key)
	return fn()
}

// CleanupUnusedLocks 清理不再使用的锁（可选，用于节省内存）
func (l *LocalRWLock) CleanupUnusedLocks() {
	// 实际使用中，可以定期清理长时间未使用的锁
	// 这里提供一个简单的实现框架
	l.locks.Range(func(key, value interface{}) bool {
		// 可以添加逻辑判断是否需要删除
		// 例如：记录最后使用时间，超过1小时未使用则删除
		return true
	})
}
