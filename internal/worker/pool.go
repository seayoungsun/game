package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// Task 定义任务函数类型
type Task func(ctx context.Context) error

// Pool Worker Pool 工作池
// 用于限制并发 goroutine 数量，防止资源耗尽
type Pool struct {
	ctx       context.Context
	cancel    context.CancelFunc
	taskQueue chan Task
	workerNum int
	wg        sync.WaitGroup

	// 统计信息
	totalTasks   int64
	successTasks int64
	failedTasks  int64
	mu           sync.Mutex
}

// NewPool 创建 Worker Pool
// workerNum: worker 数量（并发执行任务的 goroutine 数）
// queueSize: 任务队列大小（缓冲区）
func NewPool(workerNum, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		ctx:       ctx,
		cancel:    cancel,
		taskQueue: make(chan Task, queueSize),
		workerNum: workerNum,
	}

	// 启动 workers
	for i := 0; i < workerNum; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	logger.Logger.Info("Worker Pool 启动",
		zap.Int("worker_num", workerNum),
		zap.Int("queue_size", queueSize),
	)

	return pool
}

// worker 工作协程
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	logger.Logger.Debug("Worker 启动",
		zap.Int("worker_id", id),
	)

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				// 队列已关闭
				logger.Logger.Debug("Worker 退出（队列关闭）",
					zap.Int("worker_id", id),
				)
				return
			}

			// 执行任务（带超时）
			p.executeTask(id, task)

		case <-p.ctx.Done():
			// 收到关闭信号
			logger.Logger.Debug("Worker 退出（收到关闭信号）",
				zap.Int("worker_id", id),
			)
			return
		}
	}
}

// executeTask 执行任务
func (p *Pool) executeTask(workerID int, task Task) {
	// 更新统计
	p.mu.Lock()
	p.totalTasks++
	p.mu.Unlock()

	// 创建带超时的 context（默认30秒）
	taskCtx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
	defer cancel()

	// 执行任务
	startTime := time.Now()
	err := task(taskCtx)
	duration := time.Since(startTime)

	// 更新统计
	p.mu.Lock()
	if err != nil {
		p.failedTasks++
	} else {
		p.successTasks++
	}
	p.mu.Unlock()

	// 记录日志
	if err != nil {
		logger.Logger.Warn("任务执行失败",
			zap.Int("worker_id", workerID),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		logger.Logger.Debug("任务执行成功",
			zap.Int("worker_id", workerID),
			zap.Duration("duration", duration),
		)
	}
}

// Submit 提交任务到队列
// 非阻塞，如果队列满则返回错误
func (p *Pool) Submit(task Task) error {
	select {
	case p.taskQueue <- task:
		return nil
	case <-time.After(100 * time.Millisecond):
		return errors.New("任务队列已满，请稍后重试")
	case <-p.ctx.Done():
		return errors.New("Worker Pool 已关闭")
	}
}

// SubmitWithTimeout 提交任务（带超时）
// 阻塞直到任务被接受或超时
func (p *Pool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	select {
	case p.taskQueue <- task:
		return nil
	case <-time.After(timeout):
		return errors.New("提交任务超时")
	case <-p.ctx.Done():
		return errors.New("Worker Pool 已关闭")
	}
}

// Shutdown 关闭 Worker Pool（优雅关闭）
func (p *Pool) Shutdown(timeout time.Duration) error {
	logger.Logger.Info("开始关闭 Worker Pool",
		zap.Int("worker_num", p.workerNum),
	)

	// 1. 停止接收新任务
	p.cancel()

	// 2. 关闭任务队列（等待现有任务完成）
	close(p.taskQueue)

	// 3. 等待所有 worker 完成（带超时）
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Logger.Info("Worker Pool 已关闭",
			zap.Int64("total_tasks", p.totalTasks),
			zap.Int64("success_tasks", p.successTasks),
			zap.Int64("failed_tasks", p.failedTasks),
		)
		return nil
	case <-time.After(timeout):
		logger.Logger.Warn("Worker Pool 关闭超时",
			zap.Duration("timeout", timeout),
		)
		return errors.New("关闭超时")
	}
}

// Stats 获取统计信息
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"worker_num":    p.workerNum,
		"queue_size":    len(p.taskQueue),
		"queue_cap":     cap(p.taskQueue),
		"total_tasks":   p.totalTasks,
		"success_tasks": p.successTasks,
		"failed_tasks":  p.failedTasks,
	}
}

// QueueLength 获取当前队列长度
func (p *Pool) QueueLength() int {
	return len(p.taskQueue)
}

// IsFull 判断队列是否已满
func (p *Pool) IsFull() bool {
	return len(p.taskQueue) >= cap(p.taskQueue)
}
