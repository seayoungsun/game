package handlers

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/bootstrap"
	"github.com/kaifa/game-platform/internal/metrics"
)

var infrastructure *bootstrap.Infrastructure

// SetInfrastructure 设置基础设施引用（用于监控）
func SetInfrastructure(infra *bootstrap.Infrastructure) {
	infrastructure = infra
}

// GetMetrics 获取所有监控指标
func GetMetrics(c *gin.Context) {
	m := metrics.GetGlobalMetrics()

	data := map[string]interface{}{
		"lock":      m.GetLockSummary(),
		"goroutine": m.GetGoroutineStats(),
		"runtime":   m.GetRuntimeStats(),
	}

	// 添加 Worker Pool 统计
	if infrastructure != nil {
		if infrastructure.NotifyPool != nil {
			data["notify_pool"] = infrastructure.NotifyPool.Stats()
		}
		if infrastructure.TaskPool != nil {
			data["task_pool"] = infrastructure.TaskPool.Stats()
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": data,
	})
}

// GetLockMetrics 获取锁的详细监控
func GetLockMetrics(c *gin.Context) {
	m := metrics.GetGlobalMetrics()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"summary": m.GetLockSummary(),
			"details": m.GetLockStats(),
		},
	})
}

// GetWorkerPoolMetrics 获取 Worker Pool 监控
func GetWorkerPoolMetrics(c *gin.Context) {
	if infrastructure == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "基础设施未初始化",
		})
		return
	}

	data := gin.H{}

	if infrastructure.NotifyPool != nil {
		data["notify_pool"] = infrastructure.NotifyPool.Stats()
	}

	if infrastructure.TaskPool != nil {
		data["task_pool"] = infrastructure.TaskPool.Stats()
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": data,
	})
}

// GetGoroutineMetrics 获取 goroutine 监控
func GetGoroutineMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"current_count": runtime.NumGoroutine(),
			"cpu_count":     runtime.NumCPU(),
		},
	})
}

// GetRuntimeMetrics 获取运行时监控
func GetRuntimeMetrics(c *gin.Context) {
	m := metrics.GetGlobalMetrics()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": m.GetRuntimeStats(),
	})
}
