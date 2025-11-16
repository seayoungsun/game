package bootstrap

import (
	"log"
	"time"

	"github.com/kaifa/game-platform/internal/cache"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/internal/lock"
	"github.com/kaifa/game-platform/internal/worker"
	"github.com/kaifa/game-platform/pkg/utils"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Infrastructure 聚合应用运行时依赖的基础设施组件。
// 包括数据库、缓存、锁管理器、Worker Pool 等
type Infrastructure struct {
	DB       *gorm.DB
	Redis    *redis.Client
	RedisErr error

	// 并发控制组件
	DistLock   lock.Lock    // 分布式锁（Redis）
	LocalLock  lock.RWLock  // 本地读写锁
	NotifyPool *worker.Pool // 通知 Worker Pool
	TaskPool   *worker.Pool // 通用任务 Worker Pool

	closers []func() error
}

// InitInfrastructure 初始化数据库、缓存、并发控制等基础依赖。
// - MySQL 初始化失败将返回错误，需要调用方中止启动流程。
// - Redis 初始化失败会记录在 RedisErr 中，调用方可按需降级处理。
func InitInfrastructure(cfg *config.Config) (*Infrastructure, error) {
	infra := &Infrastructure{}

	// 0. 初始化雪花算法ID生成器
	machineID := int64(0) // 默认机器ID为0
	if cfg.Server.MachineID > 0 {
		machineID = int64(cfg.Server.MachineID)
	}

	if err := utils.InitSnowflake(machineID); err != nil {
		log.Printf("Warning: 雪花算法初始化失败，将使用随机算法: %v", err)
	} else {
		log.Printf("✓ 雪花算法初始化成功（机器ID: %d）", machineID)
	}

	// 1. 初始化 MySQL
	db, err := database.InitMySQL(cfg)
	if err != nil {
		return nil, err
	}
	infra.DB = db
	infra.closers = append(infra.closers, database.Close)

	// 2. 初始化 Redis
	if rdb, err := cache.InitRedis(cfg); err != nil {
		infra.RedisErr = err
		log.Printf("Warning: Redis 初始化失败，将使用降级方案: %v", err)
	} else {
		infra.Redis = rdb
		infra.closers = append(infra.closers, cache.Close)

		// ✅ 初始化分布式锁（依赖 Redis）
		infra.DistLock = lock.NewRedisLock(rdb)
		log.Printf("✓ Redis 分布式锁初始化成功")
	}

	// 3. 初始化本地读写锁（不依赖外部服务）
	infra.LocalLock = lock.NewLocalRWLock()
	log.Printf("✓ 本地读写锁初始化成功")

	// 4. 初始化 Worker Pool
	// NotifyPool: 用于发送通知（5个worker，队列100）
	infra.NotifyPool = worker.NewPool(5, 100)
	infra.closers = append(infra.closers, func() error {
		infra.NotifyPool.Shutdown(5 * time.Second)
		return nil
	})

	// TaskPool: 用于通用任务（10个worker，队列1000）
	infra.TaskPool = worker.NewPool(10, 1000)
	infra.closers = append(infra.closers, func() error {
		infra.TaskPool.Shutdown(10 * time.Second)
		return nil
	})

	log.Printf("✓ Worker Pool 初始化成功（NotifyPool: 5 workers, TaskPool: 10 workers）")

	return infra, nil
}

// Close 依照逆序调用已注册的释放函数，确保资源按初始化顺序倒序释放。
func (infra *Infrastructure) Close() {
	for i := len(infra.closers) - 1; i >= 0; i-- {
		if err := infra.closers[i](); err != nil {
			log.Printf("关闭资源失败: %v", err)
		}
	}
}
