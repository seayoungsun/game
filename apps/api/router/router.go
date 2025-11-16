package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/apps/api/handlers"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/middleware"
)

// Setup 装配所有路由
func Setup(cfg *config.Config) *gin.Engine {
	r := gin.New()

	// 通用中间件（保留与 main.go 一致的行为）
	r.Use(ginLogger())
	r.Use(ginRecovery())

	// 静态文件服务（前端页面）
	// 注意：API服务从apps/api目录运行，需要使用相对路径
	r.Static("/static", "../../web/static")
	r.StaticFile("/", "../../web/index.html")
	r.StaticFile("/login", "../../web/index.html")
	r.StaticFile("/register", "../../web/index.html")
	r.StaticFile("/lobby", "../../web/index.html")
	r.StaticFile("/room", "../../web/index.html")
	r.StaticFile("/game", "../../web/index.html")
	r.StaticFile("/leaderboard", "../../web/index.html")
	r.StaticFile("/records", "../../web/index.html")
	r.StaticFile("/recharge", "../../web/index.html")
	r.StaticFile("/withdraw", "../../web/index.html")

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"type":   "api-server",
			"port":   8080,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// ✅ 监控端点（调试用，生产环境可以添加认证）
	debug := r.Group("/debug")
	{
		debug.GET("/metrics", handlers.GetMetrics)                       // 所有监控指标
		debug.GET("/metrics/lock", handlers.GetLockMetrics)              // 锁监控
		debug.GET("/metrics/worker-pool", handlers.GetWorkerPoolMetrics) // Worker Pool 监控
		debug.GET("/metrics/goroutine", handlers.GetGoroutineMetrics)    // goroutine 监控
		debug.GET("/metrics/runtime", handlers.GetRuntimeMetrics)        // 运行时监控
	}

	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("/register", handlers.Register)
			users.POST("/login", handlers.Login)
			users.GET("/profile", middleware.AuthMiddleware(), handlers.Profile)
			users.GET("/stats", middleware.AuthMiddleware(), handlers.GetUserStats)

			// 用户消息相关
			users.GET("/messages", middleware.AuthMiddleware(), handlers.GetUserMessages)
			users.GET("/messages/unread-count", middleware.AuthMiddleware(), handlers.GetUnreadMessageCount)
			users.PUT("/messages/:id/read", middleware.AuthMiddleware(), handlers.ReadMessage)
			users.POST("/messages/batch-read", middleware.AuthMiddleware(), handlers.BatchReadMessages)
			users.DELETE("/messages/:id", middleware.AuthMiddleware(), handlers.DeleteUserMessage)
		}

		// 公告相关（公开接口）
		v1.GET("/announcements", handlers.GetAnnouncements)

		games := v1.Group("/games")
		{
			games.GET("/list", handlers.GameList)
			games.POST("/rooms", middleware.AuthMiddleware(), handlers.CreateRoom)
			games.GET("/rooms", handlers.RoomList)
			games.POST("/rooms/:roomId/join", middleware.AuthMiddleware(), handlers.JoinRoom)
			games.POST("/rooms/:roomId/leave", middleware.AuthMiddleware(), handlers.LeaveRoom)
			games.POST("/rooms/:roomId/ready", middleware.AuthMiddleware(), handlers.Ready)
			games.POST("/rooms/:roomId/cancel-ready", middleware.AuthMiddleware(), handlers.CancelReady)
			games.POST("/rooms/:roomId/start", middleware.AuthMiddleware(), handlers.StartGame)
			games.POST("/rooms/:roomId/play", middleware.AuthMiddleware(), handlers.PlayCards)
			games.POST("/rooms/:roomId/pass", middleware.AuthMiddleware(), handlers.Pass)
			games.GET("/rooms/:roomId/game-state", handlers.GetGameState)
			games.GET("/rooms/:roomId/records", middleware.AuthMiddleware(), handlers.GetRoomRecords)
			games.GET("/rooms/:roomId", handlers.GetRoom)

			// 游戏记录相关
			games.GET("/records", middleware.AuthMiddleware(), handlers.GetUserRecords)
			games.GET("/records/:id", middleware.AuthMiddleware(), handlers.GetRecordDetail)

			// 排行榜相关
			games.GET("/leaderboard", handlers.GetLeaderboard)
			games.GET("/leaderboard/my-rank", middleware.AuthMiddleware(), handlers.GetUserRank)
		}

		// 支付相关
		payments := v1.Group("/payments")
		{
			// 获取支付配置（公开接口，不需要认证）
			payments.GET("/config", handlers.GetPaymentConfig)

			// 需要认证的支付接口
			paymentsAuth := payments.Group("")
			paymentsAuth.Use(middleware.AuthMiddleware())
			{
				// 充值相关
				paymentsAuth.POST("/recharge", handlers.CreateRechargeOrder)
				paymentsAuth.GET("/recharge/:orderId", handlers.GetRechargeOrder)
				paymentsAuth.GET("/recharge", handlers.GetUserRechargeOrders)
				paymentsAuth.POST("/recharge/:orderId/check", handlers.CheckRechargeTransaction)

				// 提现相关
				paymentsAuth.POST("/withdraw", handlers.CreateWithdrawOrder)
				paymentsAuth.GET("/withdraw/:orderId", handlers.GetWithdrawOrder)
				paymentsAuth.GET("/withdraw", handlers.GetUserWithdrawOrders)
				paymentsAuth.POST("/withdraw/:orderId/audit", handlers.AuditWithdrawOrder)
			}
		}

	}

	return r
}

// 轻量日志与恢复（与 main.go 同实现，避免循环依赖）
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		_ = time.Since(start)
		_ = path
	}
}

func ginRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() { _ = recover() }()
		c.Next()
	}
}
