package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/apps/api/handlers"
	"github.com/kaifa/game-platform/apps/api/router"
	"github.com/kaifa/game-platform/internal/bootstrap"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
	mysqlrepo "github.com/kaifa/game-platform/internal/repository/mysql"
	gamesvc "github.com/kaifa/game-platform/internal/service/game"
	gamerecordsrv "github.com/kaifa/game-platform/internal/service/gamerecord"
	leaderboardsrv "github.com/kaifa/game-platform/internal/service/leaderboard"
	messagesvc "github.com/kaifa/game-platform/internal/service/message"
	paymentsvc "github.com/kaifa/game-platform/internal/service/payment"
	roomsrv "github.com/kaifa/game-platform/internal/service/room"
	usersvc "github.com/kaifa/game-platform/internal/service/user"
	userstatssvc "github.com/kaifa/game-platform/internal/service/userstats"
	"github.com/kaifa/game-platform/internal/storage"
	"github.com/kaifa/game-platform/pkg/services"
	"go.uber.org/zap"
)

func main() {
	// 加载配置（优先使用config.local.yaml，然后config.yaml，最后默认值）
	cfg, err := config.Load("")
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 初始化日志
	if err := logger.InitLogger(cfg.Log); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}
	defer logger.Sync()

	infra, err := bootstrap.InitInfrastructure(cfg)
	if err != nil {
		logger.Logger.Fatal("初始化基础设施失败", zap.Error(err))
	}
	defer infra.Close()

	if infra.RedisErr != nil {
		logger.Logger.Warn("Redis连接失败，将使用降级方案", zap.Error(infra.RedisErr))
	} else {
		logger.Logger.Info("Redis连接成功")
	}

	// ============================================
	// 初始化 Repository 层（9个）
	// ============================================
	roomRepo := mysqlrepo.NewRoomRepository(infra.DB)
	userRepo := mysqlrepo.NewUserRepository(infra.DB)
	gameRecordRepo := mysqlrepo.NewGameRecordRepository(infra.DB)
	messageRepo := mysqlrepo.NewMessageRepository(infra.DB)
	gamePlayerRepo := mysqlrepo.NewGamePlayerRepository(infra.DB)

	// 支付相关 Repository
	rechargeOrderRepo := mysqlrepo.NewRechargeOrderRepository(infra.DB)
	withdrawOrderRepo := mysqlrepo.NewWithdrawOrderRepository(infra.DB)
	transactionRepo := mysqlrepo.NewTransactionRepository(infra.DB)
	depositAddrRepo := mysqlrepo.NewDepositAddressRepository(infra.DB)

	// ============================================
	// 初始化 Service 层并注入到 handlers
	// 注意：有依赖关系的服务需要按顺序初始化
	// ============================================

	// 1. 游戏记录服务（无外部依赖）
	gameRecordService := gamerecordsrv.New(gameRecordRepo)
	handlers.SetGameRecordService(gameRecordService)
	logger.Logger.Info("✓ 游戏记录服务初始化成功")

	// 2. 排行榜服务（依赖 UserRepo）
	leaderboardService := leaderboardsrv.New(infra.Redis, userRepo)
	handlers.SetLeaderboardService(leaderboardService)
	logger.Logger.Info("✓ 排行榜服务初始化成功")

	// 3. 游戏状态存储
	gameStateStorage := storage.NewRedisGameStateStorage(infra.Redis)

	// 4. 游戏管理器（依赖 Storage + Repositories + LeaderboardService + 并发控制）
	gameManager := gamesvc.NewManager(
		gameStateStorage,   // 游戏状态存储
		roomRepo,           // 房间Repository
		userRepo,           // 用户Repository
		gameRecordRepo,     // 游戏记录Repository
		leaderboardService, // 排行榜服务
		infra.DistLock,     // ✅ 分布式锁
		infra.LocalLock,    // ✅ 本地读写锁
	)
	handlers.SetGameManager(gameManager)
	logger.Logger.Info("✓ 游戏管理器初始化成功（已启用并发控制）")

	// 5. 房间服务（依赖 GameManager + 并发控制组件）
	notifyURL := fmt.Sprintf("http://localhost:%d/internal/room/notify", cfg.Server.GamePort)
	roomService := roomsrv.New(
		roomRepo,         // Repository
		userRepo,         // Repository
		gameManager,      // Service（依赖前面创建的）
		infra.Redis,      // 基础设施
		notifyURL,        // 配置
		infra.DistLock,   // ✅ 分布式锁
		infra.LocalLock,  // ✅ 本地锁
		infra.NotifyPool, // ✅ 通知池
	)
	handlers.SetRoomService(roomService)
	logger.Logger.Info("✓ 房间服务初始化成功（已启用并发控制）")

	// 6. 用户服务（无外部依赖）
	userService := usersvc.New(userRepo)
	handlers.SetUserService(userService)
	logger.Logger.Info("✓ 用户服务初始化成功")

	// 7. 用户统计服务（依赖 GamePlayerRepo）
	userStatsService := userstatssvc.New(gamePlayerRepo)
	handlers.SetUserStatsService(userStatsService)
	logger.Logger.Info("✓ 用户统计服务初始化成功")

	// 8. 消息服务（无外部依赖）
	messageService := messagesvc.New(messageRepo)
	handlers.SetMessageService(messageService)
	logger.Logger.Info("✓ 消息服务初始化成功")

	// 9. 支付服务（依赖多个 Repository + 区块链服务）
	// 初始化 HD 钱包和转账服务
	var hdWallet *services.HDWallet
	var transferService *services.USDTTransferService

	if cfg.Payment.MasterMnemonic != "" {
		var err error
		hdWallet, err = services.NewHDWallet(cfg.Payment.MasterMnemonic)
		if err != nil {
			logger.Logger.Fatal("初始化HD钱包失败",
				zap.Error(err),
				zap.String("error_message", "请检查助记词格式是否正确"),
			)
		}
		logger.Logger.Info("✓ HD钱包初始化成功")

		transferService = services.NewUSDTTransferService(hdWallet)
		logger.Logger.Info("✓ USDT转账服务初始化成功")
	} else {
		logger.Logger.Warn("未配置主钱包助记词，支付功能将受限")
	}

	paymentService := paymentsvc.New(
		rechargeOrderRepo,
		withdrawOrderRepo,
		transactionRepo,
		depositAddrRepo,
		userRepo,
		hdWallet,
		transferService,
		cfg.Payment.EtherscanAPIKey,
	)
	handlers.SetPaymentService(paymentService)

	// 启动交易监控
	paymentService.StartTransactionMonitor()
	logger.Logger.Info("✓ 支付服务初始化成功，交易监控已启动")

	logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Logger.Info("✅ 所有服务初始化完成")
	logger.Logger.Info("   - 9个Repository（数据访问层）")
	logger.Logger.Info("   - 9个Service（业务逻辑层）")
	logger.Logger.Info("   - 1个Storage（状态存储层）")
	logger.Logger.Info("   - 并发控制（Lock + Worker Pool）")
	logger.Logger.Info("   - 监控系统（Metrics）")
	logger.Logger.Info("   - 全部使用依赖注入和接口隔离")
	logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// ✅ 设置基础设施引用（用于监控端点）
	handlers.SetInfrastructure(infra)

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := router.Setup(cfg)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务器（goroutine）
	go func() {
		logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Logger.Info("🚀 API服务启动",
			zap.String("address", srv.Addr),
			zap.String("mode", cfg.Server.Mode),
		)
		logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("API服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("正在关闭API服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal("API服务器强制关闭", zap.Error(err))
	}

	logger.Logger.Info("API服务器已关闭")
}

// 健康检查和通用中间件保留在此文件，业务路由在 router 包

// ginLogger 日志中间件
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		logger.Logger.Info("HTTP请求",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		)
	}
}

// ginRecovery 恢复中间件
func ginRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Logger.Error("Panic恢复",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "服务器内部错误",
				})
			}
		}()
		c.Next()
	}
}

// 业务处理器已移动到 apps/api/handlers 包
