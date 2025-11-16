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
	"github.com/kaifa/game-platform/apps/admin/router"
	"github.com/kaifa/game-platform/internal/cache"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/internal/elasticsearch"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 初始化日志
	if err := logger.InitLogger(cfg.Log); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}
	defer logger.Sync()

	// 初始化数据库
	_, err = database.InitMySQL(cfg)
	if err != nil {
		logger.Logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer database.Close()

	// 初始化Redis（可选）
	if _, err := cache.InitRedis(cfg); err != nil {
		logger.Logger.Warn("Redis连接失败，将使用降级方案", zap.Error(err))
	} else {
		logger.Logger.Info("Redis连接成功")
	}
	defer cache.Close()

	// 初始化Elasticsearch（必需）
	if err := elasticsearch.Init(cfg); err != nil {
		logger.Logger.Fatal("Elasticsearch连接失败，服务无法启动", zap.Error(err))
	}
	logger.Logger.Info("Elasticsearch连接成功")

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := router.Setup(cfg)

	// 管理后台使用独立端口（8082）
	adminPort := 8082
	if cfg.Server.AdminPort > 0 {
		adminPort = cfg.Server.AdminPort
	}

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", adminPort),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务器（goroutine）
	go func() {
		logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Logger.Info("🔐 管理后台服务启动",
			zap.String("address", srv.Addr),
			zap.String("mode", cfg.Server.Mode),
		)
		logger.Logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("管理后台服务启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("正在关闭管理后台服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal("管理后台服务强制关闭", zap.Error(err))
	}

	logger.Logger.Info("管理后台服务已关闭")
}
