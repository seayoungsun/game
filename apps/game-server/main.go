package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/apps/game-server/adapters"
	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/apps/game-server/handlers"
	gameMessaging "github.com/kaifa/game-platform/apps/game-server/messaging"
	"github.com/kaifa/game-platform/apps/game-server/services"
	"github.com/kaifa/game-platform/internal/bootstrap"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/discovery"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/internal/messaging"
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

	// 全局实例
	hubInstance          *core.Hub
	broadcasterInstance  *gameMessaging.Broadcaster
	kafkaHandlerInstance *gameMessaging.KafkaHandler
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

	// 获取本机IP
	localIP := getLocalIP()
	instanceID := fmt.Sprintf("gs-%d-%d", cfg.Server.MachineID, os.Getpid())

	// 确定健康检查地址
	healthCheckAddr := cfg.ServiceDiscovery.HealthCheckAddress
	if healthCheckAddr == "" {
		// 如果未配置，使用自动检测的 IP
		healthCheckAddr = localIP
	}
	healthCheckURL := fmt.Sprintf("http://%s:%d/health", healthCheckAddr, cfg.Server.GamePort)

	// 初始化服务发现
	var registry discovery.Registry
	var stopKeepAlive func()
	if cfg.ServiceDiscovery.Enabled {
		registryDeps := discovery.RegistryDeps{
			Type:                cfg.ServiceDiscovery.Type,
			ConsulAddr:          cfg.ServiceDiscovery.ConsulAddr,
			Redis:               infra.Redis,
			ServiceName:         "game-server",
			InstanceID:          instanceID,
			InstanceAddress:     localIP,
			InstancePort:        cfg.Server.GamePort,
			HealthCheckURL:      healthCheckURL,
			HealthCheckInterval: time.Duration(cfg.ServiceDiscovery.HealthCheckInterval) * time.Second,
			HealthCheckTimeout:  time.Duration(cfg.ServiceDiscovery.HealthCheckTimeout) * time.Second,
			DeregisterAfter:     time.Duration(cfg.ServiceDiscovery.DeregisterAfter) * time.Second,
			InstanceTTL:         time.Duration(cfg.ServiceDiscovery.InstanceTTL) * time.Second,
			HeartbeatInterval:   time.Duration(cfg.ServiceDiscovery.HeartbeatInterval) * time.Second,
		}

		var err error
		registry, err = discovery.NewRegistry(registryDeps)
		if err != nil {
			logger.Logger.Fatal("创建服务注册器失败", zap.Error(err))
		}

		if registry != nil {
			// 注册服务
			instance := discovery.ServiceInstance{
				ServiceName: "game-server",
				InstanceID:  instanceID,
				Address:     localIP,
				Port:        cfg.Server.GamePort,
				Meta: map[string]string{
					"machine_id": fmt.Sprintf("%d", cfg.Server.MachineID),
					"version":    "1.0.0",
				},
			}

			if err := registry.Register(context.Background(), instance); err != nil {
				logger.Logger.Fatal("服务注册失败", zap.Error(err))
			}

			// 启动心跳
			stopKeepAlive, err = registry.KeepAlive(context.Background(), instanceID)
			if err != nil {
				logger.Logger.Fatal("启动心跳失败", zap.Error(err))
			}

			logger.Logger.Info("服务发现已启用",
				zap.String("type", cfg.ServiceDiscovery.Type),
				zap.String("instance_id", instanceID),
				zap.String("address", localIP),
				zap.Int("port", cfg.Server.GamePort),
			)
		}
	} else {
		logger.Logger.Warn("服务发现未启用，游戏服务器将以单实例模式运行")
	}

	// 初始化消息总线
	var messageBus messaging.MessageBus
	if cfg.Kafka.Enabled {
		busDeps := messaging.BusDeps{
			Type:                   "kafka",
			Brokers:                cfg.Kafka.Brokers,
			TopicPrefix:            cfg.Kafka.TopicPrefix,
			ConsumerGroup:          cfg.Kafka.ConsumerGroup,
			InstanceID:             instanceID,
			ProducerAcks:           cfg.Kafka.ProducerAcks,
			ProducerRetries:        cfg.Kafka.ProducerRetries,
			BatchSize:              cfg.Kafka.BatchSize,
			LingerMs:               cfg.Kafka.LingerMs,
			CompressionType:        cfg.Kafka.CompressionType,
			ConsumerAutoCommit:     cfg.Kafka.ConsumerAutoCommit,
			ConsumerMaxPollRecords: cfg.Kafka.ConsumerMaxPollRecords,
			FetchMinBytes:          cfg.Kafka.FetchMinBytes,
			FetchMaxWaitMs:         cfg.Kafka.FetchMaxWaitMs,
		}

		var err error
		messageBus, err = messaging.NewMessageBus(busDeps)
		if err != nil {
			logger.Logger.Fatal("创建消息总线失败", zap.Error(err))
		}

		if messageBus != nil {
			logger.Logger.Info("消息总线已启用",
				zap.String("type", "kafka"),
				zap.Strings("brokers", cfg.Kafka.Brokers),
				zap.String("consumer_group", cfg.Kafka.ConsumerGroup),
			)
		}
	} else {
		logger.Logger.Warn("消息总线未启用，跨实例消息功能不可用")
	}

	// 初始化 Hub
	hubInstance = core.NewHub(messageBus, instanceID)

	// 初始化 Broadcaster
	broadcasterInstance = gameMessaging.NewBroadcaster(hubInstance, messageBus, instanceID)

	// 初始化 KafkaHandler
	kafkaHandlerInstance = gameMessaging.NewKafkaHandler(hubInstance, broadcasterInstance, messageBus, instanceID)

	// 启动 Hub workers
	hubInstance.StartWorkers()

	// 启动广播 worker
	go runBroadcastWorker(hubInstance, broadcasterInstance)

	// 如果启用了消息总线，订阅全局广播频道
	if messageBus != nil {
		broadcastTopic := "broadcast-all"
		if err := messageBus.Subscribe(context.Background(), broadcastTopic, kafkaHandlerInstance.HandleCrossInstanceBroadcast); err != nil {
			logger.Logger.Error("订阅全局广播频道失败", zap.Error(err))
		} else {
			logger.Logger.Info("已订阅全局广播频道",
				zap.String("topic", broadcastTopic),
				zap.String("instance_id", instanceID),
			)
		}
	}

	// 初始化 handlers 依赖
	initHandlers(broadcasterInstance)

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

	// 停止心跳
	if stopKeepAlive != nil {
		stopKeepAlive()
	}

	// 注销服务
	if registry != nil {
		if err := registry.Deregister(context.Background(), instanceID); err != nil {
			logger.Logger.Error("服务注销失败", zap.Error(err))
		} else {
			logger.Logger.Info("服务已注销", zap.String("instance_id", instanceID))
		}
	}

	// 关闭消息总线
	if messageBus != nil {
		if err := messageBus.Close(); err != nil {
			logger.Logger.Error("关闭消息总线失败", zap.Error(err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal("游戏服务器强制关闭", zap.Error(err))
	}

	logger.Logger.Info("游戏服务器已关闭")
}

// initHandlers 初始化 handlers 包的依赖
func initHandlers(broadcaster *gameMessaging.Broadcaster) {
	// 创建 Hub 适配器
	hubAdapter := adapters.NewHubAdapter(hubInstance, broadcaster, kafkaHandlerInstance)

	// 创建 Client 适配器工厂函数
	clientAdapterFunc := func(conn *websocket.Conn, ip string, userID uint) handlers.ClientInterface {
		// 创建 core.Client
		client := core.NewClient(conn, ip, userID, hubInstance)

		// 创建 MessageHandler
		messageHandler := services.NewMessageHandler(client, hubInstance, broadcaster)

		// 创建 ClientAdapter
		return adapters.NewClientAdapter(client, messageHandler)
	}

	// 创建 Message 适配器工厂函数
	messageAdapterFunc := func(msgType, roomID string, userID uint, rawData interface{}) handlers.MessageInterface {
		return adapters.NewMessageAdapter(&core.Message{
			Type:    msgType,
			RoomID:  roomID,
			UserID:  userID,
			RawData: rawData,
		})
	}

	// 注入依赖
	handlers.SetUpgrader(&upgrader)
	handlers.SetHub(hubAdapter)
	handlers.SetNewClientFunc(clientAdapterFunc)
	handlers.SetNewMessageFunc(messageAdapterFunc)
}

// runBroadcastWorker 处理广播消息的 worker goroutine
func runBroadcastWorker(hub *core.Hub, broadcaster *gameMessaging.Broadcaster) {
	// 使用 for range 从 channel 读取消息（channel关闭时自动退出）
	for message := range hub.GetBroadcastChannel() {
		// 使用 broadcaster 广播消息
		broadcaster.BroadcastMessage(message)
	}
}

// getLocalIP 获取本机IP地址
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
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
			"connections": hubInstance.GetConnectionCount(),
			"rooms":       hubInstance.GetRoomCount(),
			"time":        time.Now().Format(time.RFC3339),
		})
	})

	// WebSocket连接
	r.GET("/ws", handlers.HandleWebSocket)

	// 内部API：房间状态更新通知（供API服务调用）
	r.POST("/internal/room/notify", handlers.HandleRoomNotify)

	return r
}
