package metrics

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics 全局监控指标收集器
type Metrics struct {
	// 锁相关指标
	lockAcquireCount   int64 // 锁获取总次数
	lockAcquireSuccess int64 // 锁获取成功次数
	lockAcquireFailed  int64 // 锁获取失败次数
	lockWaitTimeTotal  int64 // 锁等待总时间（纳秒）
	lockHoldTimeTotal  int64 // 锁持有总时间（纳秒）

	// 按key统计的锁信息
	lockStatsByKey sync.Map // key: lockKey, value: *LockStats

	// Worker Pool 指标（已在 worker.Pool 中实现）

	// 系统指标
	mu        sync.RWMutex
	startTime time.Time
}

// LockStats 单个锁的统计信息
type LockStats struct {
	Key           string
	AcquireCount  int64
	SuccessCount  int64
	FailedCount   int64
	TotalWaitTime int64 // 纳秒
	TotalHoldTime int64 // 纳秒
	LastAcquireAt int64 // Unix 时间戳
	LastReleaseAt int64 // Unix 时间戳
}

var globalMetrics = &Metrics{
	startTime: time.Now(),
}

// GetGlobalMetrics 获取全局监控指标
func GetGlobalMetrics() *Metrics {
	return globalMetrics
}

// RecordLockAcquire 记录锁获取
func (m *Metrics) RecordLockAcquire(key string, success bool, waitTime time.Duration) {
	atomic.AddInt64(&m.lockAcquireCount, 1)

	if success {
		atomic.AddInt64(&m.lockAcquireSuccess, 1)
	} else {
		atomic.AddInt64(&m.lockAcquireFailed, 1)
	}

	atomic.AddInt64(&m.lockWaitTimeTotal, int64(waitTime))

	// 更新按key的统计
	stats := m.getOrCreateLockStats(key)
	atomic.AddInt64(&stats.AcquireCount, 1)
	if success {
		atomic.AddInt64(&stats.SuccessCount, 1)
		atomic.StoreInt64(&stats.LastAcquireAt, time.Now().Unix())
	} else {
		atomic.AddInt64(&stats.FailedCount, 1)
	}
	atomic.AddInt64(&stats.TotalWaitTime, int64(waitTime))
}

// RecordLockRelease 记录锁释放
func (m *Metrics) RecordLockRelease(key string, holdTime time.Duration) {
	atomic.AddInt64(&m.lockHoldTimeTotal, int64(holdTime))

	// 更新按key的统计
	stats := m.getOrCreateLockStats(key)
	atomic.AddInt64(&stats.TotalHoldTime, int64(holdTime))
	atomic.StoreInt64(&stats.LastReleaseAt, time.Now().Unix())
}

// getOrCreateLockStats 获取或创建锁统计
func (m *Metrics) getOrCreateLockStats(key string) *LockStats {
	if stats, ok := m.lockStatsByKey.Load(key); ok {
		return stats.(*LockStats)
	}

	stats := &LockStats{
		Key: key,
	}
	m.lockStatsByKey.Store(key, stats)
	return stats
}

// GetLockStats 获取所有锁的统计信息
func (m *Metrics) GetLockStats() []*LockStats {
	stats := make([]*LockStats, 0)
	m.lockStatsByKey.Range(func(key, value interface{}) bool {
		stats = append(stats, value.(*LockStats))
		return true
	})
	return stats
}

// GetLockSummary 获取锁的汇总信息
func (m *Metrics) GetLockSummary() map[string]interface{} {
	totalCount := atomic.LoadInt64(&m.lockAcquireCount)
	successCount := atomic.LoadInt64(&m.lockAcquireSuccess)
	failedCount := atomic.LoadInt64(&m.lockAcquireFailed)
	totalWaitTime := atomic.LoadInt64(&m.lockWaitTimeTotal)
	totalHoldTime := atomic.LoadInt64(&m.lockHoldTimeTotal)

	avgWaitTime := int64(0)
	avgHoldTime := int64(0)
	successRate := float64(0)

	if totalCount > 0 {
		avgWaitTime = totalWaitTime / totalCount
		avgHoldTime = totalHoldTime / successCount
		successRate = float64(successCount) / float64(totalCount) * 100
	}

	return map[string]interface{}{
		"total_acquire_count": totalCount,
		"success_count":       successCount,
		"failed_count":        failedCount,
		"success_rate":        successRate,
		"avg_wait_time_ms":    float64(avgWaitTime) / 1e6,
		"avg_hold_time_ms":    float64(avgHoldTime) / 1e6,
		"total_wait_time_ms":  float64(totalWaitTime) / 1e6,
		"total_hold_time_ms":  float64(totalHoldTime) / 1e6,
	}
}

// GetGoroutineStats 获取 goroutine 统计
func (m *Metrics) GetGoroutineStats() map[string]interface{} {
	return map[string]interface{}{
		"current_count": runtime.NumGoroutine(),
		"cpu_count":     runtime.NumCPU(),
	}
}

// GetRuntimeStats 获取运行时统计
func (m *Metrics) GetRuntimeStats() map[string]interface{} {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return map[string]interface{}{
		"uptime_seconds":  time.Since(m.startTime).Seconds(),
		"goroutine_count": runtime.NumGoroutine(),
		"cpu_count":       runtime.NumCPU(),
		"memory_alloc_mb": float64(mem.Alloc) / 1024 / 1024,
		"memory_total_mb": float64(mem.TotalAlloc) / 1024 / 1024,
		"memory_sys_mb":   float64(mem.Sys) / 1024 / 1024,
		"gc_count":        mem.NumGC,
		"last_gc_time":    time.Unix(0, int64(mem.LastGC)).Format("2006-01-02 15:04:05"),
	}
}

// GetAllMetrics 获取所有监控指标
func (m *Metrics) GetAllMetrics() map[string]interface{} {
	return map[string]interface{}{
		"lock_summary": m.GetLockSummary(),
		"lock_details": m.GetLockStats(),
		"goroutine":    m.GetGoroutineStats(),
		"runtime":      m.GetRuntimeStats(),
	}
}

// Reset 重置所有指标（用于测试）
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.lockAcquireCount, 0)
	atomic.StoreInt64(&m.lockAcquireSuccess, 0)
	atomic.StoreInt64(&m.lockAcquireFailed, 0)
	atomic.StoreInt64(&m.lockWaitTimeTotal, 0)
	atomic.StoreInt64(&m.lockHoldTimeTotal, 0)
	m.lockStatsByKey = sync.Map{}
}
