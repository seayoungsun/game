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
	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/apps/game-server/handlers"
	"github.com/kaifa/game-platform/internal/bootstrap"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 生产环境需要验证来源
			return true
		},
		// 增加读写缓冲区大小，提高性能
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		// 允许所有来源（开发环境）
		EnableCompression: false, // 禁用压缩，减少CPU开销
	}

	// 全局Hub实例
	hub *Hub
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
	}

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化Hub
	hub = NewHub()
	go hub.Run()

	// 初始化 handlers 依赖
	initHandlers()

	// 创建路由
	r := setupRouter()

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.GamePort),
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		IdleTimeout:    120 * time.Second,
		// 不限制连接数（Go的http.Server默认无限制）
		// 但需要确保系统资源充足
	}

	// 启动服务器（goroutine）
	go func() {
		logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Logger.Info("🎮 游戏服务器启动",
			zap.String("address", srv.Addr),
			zap.String("mode", cfg.Server.Mode),
		)
		logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("游戏服务器启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("正在关闭游戏服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal("游戏服务器强制关闭", zap.Error(err))
	}

	logger.Logger.Info("游戏服务器已关闭")
}

func setupRouter() *gin.Engine {
	r := gin.New()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"type":   "game-server",
			"port":   8081,
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 连接统计（用于测试和监控）
	r.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"connections": hub.GetConnectionCount(),
			"rooms":       hub.GetRoomCount(),
			"time":        time.Now().Format(time.RFC3339),
		})
	})

	// WebSocket连接
	r.GET("/ws", handlers.HandleWebSocket)

	// 内部API：房间状态更新通知（供API服务调用）
	r.POST("/internal/room/notify", handlers.HandleRoomNotify)

	return r
}
