package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/internal/bootstrap"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/pkg/utils"
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
	r.GET("/ws", handleWebSocket)

	// 内部API：房间状态更新通知（供API服务调用）
	r.POST("/internal/room/notify", handleRoomNotify)

	return r
}

// RoomNotifyRequest 房间通知请求
type RoomNotifyRequest struct {
	RoomID   string                 `json:"room_id" binding:"required"`
	Action   string                 `json:"action" binding:"required"` // join, leave, ready, cancel_ready, start, game_end, room_created, room_deleted
	UserID   uint                   `json:"user_id"`                   // 用户ID（可选，game_end和room_deleted时可能为0）
	RoomData map[string]interface{} `json:"room_data,omitempty"`       // 房间数据（可选）
}

// handleRoomNotify 处理房间通知（供API服务调用）
func handleRoomNotify(c *gin.Context) {
	var req RoomNotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		logger.Logger.Error("handleRoomNotify: 参数绑定失败", zap.Error(err))
		return
	}

	// 对于某些action（如game_end, room_created, room_deleted），UserID可以为0
	// 但其他action需要UserID
	if req.Action != "game_end" && req.Action != "room_created" && req.Action != "room_deleted" {
		if req.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "参数错误",
				"error":   "user_id is required for action: " + req.Action,
			})
			logger.Logger.Error("handleRoomNotify: user_id is required", zap.String("action", req.Action))
			return
		}
	}

	// 如果action是room_created，广播房间创建消息给所有客户端（大厅）
	if req.Action == "room_created" && req.RoomData != nil {
		// req.RoomData 中应该包含 room_data 字段（从 room_service.go 发送）
		var roomData map[string]interface{}

		// 检查是否有嵌套的 room_data 字段
		if roomDataValue, ok := req.RoomData["room_data"]; ok {
			if roomDataMap, ok := roomDataValue.(map[string]interface{}); ok {
				roomData = roomDataMap
				logger.Logger.Debug("从 room_data 字段提取房间数据",
					zap.String("room_id", req.RoomID),
					zap.Any("room_data", roomData),
				)
			} else {
				// 如果不是 map，尝试直接使用 req.RoomData
				roomData = req.RoomData
				logger.Logger.Debug("room_data 不是 map，直接使用 req.RoomData",
					zap.String("room_id", req.RoomID),
				)
			}
		} else {
			// 如果没有 room_data 字段，直接使用 req.RoomData
			roomData = req.RoomData
			logger.Logger.Debug("没有 room_data 字段，直接使用 req.RoomData",
				zap.String("room_id", req.RoomID),
				zap.Any("req_room_data", req.RoomData),
			)
		}

		logger.Logger.Info("房间创建通知准备广播",
			zap.String("room_id", req.RoomID),
			zap.Uint("creator_id", req.UserID),
			zap.Any("room_data", roomData),
		)

		// 广播给所有客户端（大厅中的所有人）
		// RoomID和UserID都设为0，表示广播给所有客户端（不限制房间或用户）
		hub.broadcast <- &Message{
			Type:   "room_created",
			RoomID: "", // 大厅消息，没有room_id
			UserID: 0,  // 设为0，表示广播给所有客户端
			RawData: map[string]interface{}{
				"message": "新房间已创建",
				"room":    roomData,
			},
		}

		logger.Logger.Info("房间创建通知已广播",
			zap.String("room_id", req.RoomID),
			zap.Uint("creator_id", req.UserID),
		)

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "通知已发送",
		})
		return
	}

	// 如果action是room_deleted，广播房间删除消息给所有客户端（大厅）
	if req.Action == "room_deleted" {
		logger.Logger.Info("房间删除通知准备广播",
			zap.String("room_id", req.RoomID),
			zap.Uint("user_id", req.UserID),
		)

		// 广播给所有客户端（大厅中的所有人）
		// RoomID和UserID都设为0，表示广播给所有客户端（不限制房间或用户）
		hub.broadcast <- &Message{
			Type:   "room_deleted",
			RoomID: "", // 大厅消息，没有room_id
			UserID: 0,  // 设为0，表示广播给所有客户端
			RawData: map[string]interface{}{
				"message": "房间已解散",
				"room_id": req.RoomID,
			},
		}

		logger.Logger.Info("房间删除通知已广播",
			zap.String("room_id", req.RoomID),
		)

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "通知已发送",
		})
		return
	}

	// 构建广播消息
	msg := &Message{
		Type:   "room_updated",
		RoomID: req.RoomID,
		UserID: req.UserID,
		RawData: map[string]interface{}{
			"action":    req.Action,
			"user_id":   req.UserID,
			"room_data": req.RoomData,
		},
	}

	// 如果action是game_state_update，广播游戏状态（为每个用户过滤手牌）
	if req.Action == "game_state_update" && req.RoomData != nil {
		if gameStateData, ok := req.RoomData["game_state"].(map[string]interface{}); ok {
			isRaw, _ := req.RoomData["is_raw"].(bool)

			if isRaw {
				// 需要为每个用户过滤手牌，发送给房间内所有客户端
				// 获取房间内的所有玩家ID
				if playersData, ok := gameStateData["players"].(map[string]interface{}); ok {
					for playerKey, playerData := range playersData {
						var userIDUint uint

						// 从玩家数据中获取user_id
						if playerInfo, ok := playerData.(map[string]interface{}); ok {
							switch v := playerInfo["user_id"].(type) {
							case float64:
								userIDUint = uint(v)
							case int:
								userIDUint = uint(v)
							case uint:
								userIDUint = v
							case int64:
								userIDUint = uint(v)
							default:
								continue
							}

							// 为每个用户过滤手牌
							filteredState := filterGameStateForUser(gameStateData, userIDUint)

							// 发送给该用户的客户端
							if client := hub.GetUserClient(userIDUint); client != nil {
								client.SendMessage(&Message{
									Type:   "game_state_update",
									RoomID: req.RoomID,
									UserID: userIDUint,
									RawData: map[string]interface{}{
										"game_state": filteredState,
									},
								})
							}
						}

						_ = playerKey // 避免未使用变量
					}
				}

				// 也广播给房间内的所有客户端（通用广播）
				hub.broadcast <- &Message{
					Type:   "game_state_update",
					RoomID: req.RoomID,
					UserID: req.UserID,
					RawData: map[string]interface{}{
						"game_state": gameStateData, // 发送原始数据，客户端需要自己过滤
						"note":       "需要客户端过滤手牌",
					},
				}
			} else {
				// 已经是过滤后的状态，直接广播
				hub.broadcast <- &Message{
					Type:   "game_state_update",
					RoomID: req.RoomID,
					UserID: req.UserID,
					RawData: map[string]interface{}{
						"game_state": gameStateData,
					},
				}
			}
		}
	}

	// 如果action是timer_start，广播倒计时开始
	if req.Action == "timer_start" {
		var timeout, startTime float64
		if data, ok := req.RoomData["timeout"]; ok {
			if t, ok := data.(float64); ok {
				timeout = t
			}
		}
		if data, ok := req.RoomData["start_time"]; ok {
			if st, ok := data.(float64); ok {
				startTime = st
			}
		}

		hub.broadcast <- &Message{
			Type:   "timer_start",
			RoomID: req.RoomID,
			UserID: req.UserID,
			RawData: map[string]interface{}{
				"user_id":    req.UserID,
				"timeout":    int(timeout),
				"start_time": int64(startTime),
				"message":    "开始倒计时",
			},
		}
	}

	// 如果action是timer_stop，广播计时器停止
	if req.Action == "timer_stop" {
		hub.broadcast <- &Message{
			Type:   "timer_stop",
			RoomID: req.RoomID,
			UserID: req.UserID,
			RawData: map[string]interface{}{
				"message": "计时器已停止",
			},
		}
	}

	// 如果action是game_started，广播游戏开始（包含游戏状态）
	if req.Action == "game_started" && req.RoomData != nil {
		if gameStateData, ok := req.RoomData["game_state"].(map[string]interface{}); ok {
			// 首先尝试从room数据中获取玩家列表
			var playersToNotify []uint

			// 从room_data中获取玩家列表
			if roomData, ok := req.RoomData["room"].(map[string]interface{}); ok {
				if playersData, ok := roomData["players"]; ok {
					// 解析玩家列表（可能是JSON字符串或数组）
					var players []map[string]interface{}

					// 尝试解析为JSON字符串
					if playersStr, ok := playersData.(string); ok {
						var playersArray []map[string]interface{}
						if err := json.Unmarshal([]byte(playersStr), &playersArray); err == nil {
							players = playersArray
						}
					} else if playersArray, ok := playersData.([]interface{}); ok {
						// 已经是数组格式
						for _, p := range playersArray {
							if pMap, ok := p.(map[string]interface{}); ok {
								players = append(players, pMap)
							}
						}
					}

					// 提取所有玩家ID
					for _, player := range players {
						var userIDUint uint
						switch v := player["user_id"].(type) {
						case float64:
							userIDUint = uint(v)
						case int:
							userIDUint = uint(v)
						case uint:
							userIDUint = v
						case int64:
							userIDUint = uint(v)
						default:
							continue
						}
						playersToNotify = append(playersToNotify, userIDUint)
					}
				}
			}

			// 如果没有从room数据获取到，尝试从game_state中获取
			if len(playersToNotify) == 0 {
				if playersData, ok := gameStateData["players"].(map[string]interface{}); ok {
					for _, playerData := range playersData {
						if playerInfo, ok := playerData.(map[string]interface{}); ok {
							var userIDUint uint
							switch v := playerInfo["user_id"].(type) {
							case float64:
								userIDUint = uint(v)
							case int:
								userIDUint = uint(v)
							case uint:
								userIDUint = v
							case int64:
								userIDUint = uint(v)
							default:
								continue
							}
							playersToNotify = append(playersToNotify, userIDUint)
						}
					}
				}
			}

			// 给所有玩家发送游戏状态（为每个用户过滤手牌）
			for _, userIDUint := range playersToNotify {
				if client := hub.GetUserClient(userIDUint); client != nil {
					filteredState := filterGameStateForUser(gameStateData, userIDUint)

					// 发送过滤后的游戏状态给该客户端
					client.SendMessage(&Message{
						Type:   "game_state_update",
						RoomID: req.RoomID,
						UserID: userIDUint,
						RawData: map[string]interface{}{
							"game_state": filteredState,
							"message":    "游戏已开始",
						},
					})

					logger.Logger.Info("发送游戏开始消息给玩家",
						zap.Uint("user_id", userIDUint),
						zap.String("room_id", req.RoomID),
					)
				} else {
					logger.Logger.Warn("玩家未连接WebSocket",
						zap.Uint("user_id", userIDUint),
						zap.String("room_id", req.RoomID),
					)
				}
			}

			// 同时也给已加入房间的客户端发送（确保不漏掉）
			clients := hub.GetRoomClients(req.RoomID)
			for _, client := range clients {
				if client != nil {
					// 检查是否已经发送过（避免重复）
					alreadySent := false
					for _, userID := range playersToNotify {
						if userID == client.userID {
							alreadySent = true
							break
						}
					}

					if !alreadySent {
						filteredState := filterGameStateForUser(gameStateData, client.userID)

						client.SendMessage(&Message{
							Type:   "game_state_update",
							RoomID: req.RoomID,
							UserID: client.userID,
							RawData: map[string]interface{}{
								"game_state": filteredState,
								"message":    "游戏已开始",
							},
						})
					}
				}
			}
		}
	}

	// 如果action是game_end，广播游戏结束（包含结算结果）
	if req.Action == "game_end" && req.RoomData != nil {
		broadcastData := map[string]interface{}{
			"message": "游戏已结束",
		}

		var gameStateData map[string]interface{}
		if gs, ok := req.RoomData["game_state"].(map[string]interface{}); ok {
			gameStateData = gs
			broadcastData["game_state"] = gameStateData
		}

		// 预先获取结算数据（如果存在）
		var settlementData map[string]interface{}
		hasSettlement := false
		if sd, ok := req.RoomData["settlement"].(map[string]interface{}); ok {
			settlementData = sd
			hasSettlement = true
			broadcastData["settlement"] = settlementData
		}

		// 从game_state中获取所有玩家ID，确保所有玩家都收到消息
		var playersToNotify []uint
		if gameStateData != nil {
			if playersData, ok := gameStateData["players"].(map[string]interface{}); ok {
				for _, playerData := range playersData {
					if playerInfo, ok := playerData.(map[string]interface{}); ok {
						var userIDUint uint
						switch v := playerInfo["user_id"].(type) {
						case float64:
							userIDUint = uint(v)
						case int:
							userIDUint = uint(v)
						case uint:
							userIDUint = v
						case int64:
							userIDUint = uint(v)
						default:
							continue
						}
						playersToNotify = append(playersToNotify, userIDUint)
					}
				}
			}
		}

		// 给所有玩家发送游戏结束消息
		if len(playersToNotify) > 0 {
			logger.Logger.Info("发送游戏结束消息给所有玩家",
				zap.String("room_id", req.RoomID),
				zap.Int("player_count", len(playersToNotify)),
				zap.Any("players", playersToNotify),
			)

			for _, userIDUint := range playersToNotify {
				if client := hub.GetUserClient(userIDUint); client != nil {
					// 为每个玩家构建个性化的消息（包含过滤后的游戏状态）
					personalData := make(map[string]interface{})
					if gameStateData != nil {
						filteredState := filterGameStateForUser(gameStateData, userIDUint)
						personalData["game_state"] = filteredState
					}
					if hasSettlement {
						personalData["settlement"] = settlementData
					}
					personalData["message"] = "游戏已结束，请查看结算结果"

					client.SendMessage(&Message{
						Type:    "game_end",
						RoomID:  req.RoomID,
						UserID:  userIDUint,
						RawData: personalData,
					})
					logger.Logger.Info("已发送游戏结束消息给玩家",
						zap.Uint("user_id", userIDUint),
						zap.String("room_id", req.RoomID),
						zap.Bool("has_settlement", hasSettlement),
					)
				} else {
					logger.Logger.Warn("玩家未连接WebSocket，无法发送游戏结束消息",
						zap.Uint("user_id", userIDUint),
						zap.String("room_id", req.RoomID),
					)
				}
			}
		} else {
			logger.Logger.Warn("游戏结束但没有找到玩家列表",
				zap.String("room_id", req.RoomID),
				zap.Any("game_state_data", gameStateData),
			)
		}

		// 同时也广播给房间内的所有客户端（已通过WebSocket加入房间的）
		clients := hub.GetRoomClients(req.RoomID)
		for _, client := range clients {
			if client != nil {
				// 检查是否已经发送过（避免重复）
				alreadySent := false
				for _, userID := range playersToNotify {
					if userID == client.userID {
						alreadySent = true
						break
					}
				}

				if !alreadySent {
					// 为每个客户端构建个性化的消息（包含过滤后的游戏状态）
					personalData := make(map[string]interface{})
					if gameStateData != nil {
						filteredState := filterGameStateForUser(gameStateData, client.userID)
						personalData["game_state"] = filteredState
					}
					if hasSettlement {
						personalData["settlement"] = settlementData
					}
					personalData["message"] = "游戏已结束，请查看结算结果"

					client.SendMessage(&Message{
						Type:    "game_end",
						RoomID:  req.RoomID,
						UserID:  client.userID,
						RawData: personalData,
					})
					logger.Logger.Info("已发送游戏结束消息给房间内客户端",
						zap.Uint("user_id", client.userID),
						zap.String("room_id", req.RoomID),
						zap.Bool("has_settlement", hasSettlement),
					)
				}
			}
		}

		logger.Logger.Info("游戏结束消息已广播",
			zap.String("room_id", req.RoomID),
			zap.Int("notified_count", len(playersToNotify)),
		)
	}

	// 如果提供了房间数据，从房间数据中获取所有用户ID并广播给这些用户
	// 这样即使客户端没有通过WebSocket加入房间，也能收到消息
	if req.RoomData != nil {
		if playersData, ok := req.RoomData["players"]; ok {
			// 解析玩家列表（支持多种数字类型）
			playersJSON, err := json.Marshal(playersData)
			if err == nil {
				var players []map[string]interface{}
				if err := json.Unmarshal(playersJSON, &players); err == nil {
					// 向房间内的所有用户发送消息（包括没有通过WebSocket加入房间的）
					for _, player := range players {
						var userIDUint uint

						// 尝试不同的数字类型（JSON解析可能是float64）
						switch v := player["user_id"].(type) {
						case float64:
							userIDUint = uint(v)
						case int:
							userIDUint = uint(v)
						case uint:
							userIDUint = v
						case int64:
							userIDUint = uint(v)
						default:
							continue
						}

						// 如果有WebSocket连接，发送消息
						if client := hub.GetUserClient(userIDUint); client != nil {
							logger.Logger.Debug("向用户发送房间更新消息",
								zap.Uint("user_id", userIDUint),
								zap.String("room_id", req.RoomID),
								zap.String("action", req.Action),
							)
							client.SendMessage(msg)
						}
					}
				}
			}
		}
	}

	// 同时广播给房间内的所有客户端（已通过WebSocket加入房间的）
	hub.broadcast <- msg

	logger.Logger.Info("房间状态通知已广播",
		zap.String("room_id", req.RoomID),
		zap.String("action", req.Action),
		zap.Uint("user_id", req.UserID),
	)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "通知已发送",
	})
}

func handleWebSocket(c *gin.Context) {
	// 获取Token（从query参数或header）
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "缺少认证token"})
		return
	}

	// 验证Token
	claims, err := utils.ParseToken(token)
	if err != nil {
		logger.Logger.Warn("Token验证失败", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的token"})
		return
	}

	// 升级到WebSocket连接
	// 注意：Upgrade会接管ResponseWriter，后续不能再用c.JSON等
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// 如果Upgrade失败，ResponseWriter可能已经被部分写入
		// 检查是否已经写入响应头
		if !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "WebSocket升级失败",
			})
		}
		logger.Logger.Error("WebSocket升级失败",
			zap.Error(err),
			zap.Uint("user_id", claims.UserID),
			zap.String("ip", c.ClientIP()),
			zap.String("remote_addr", c.Request.RemoteAddr),
		)
		return
	}

	logger.Logger.Info("新的WebSocket连接",
		zap.Uint("user_id", claims.UserID),
		zap.String("ip", c.ClientIP()),
	)

	// 创建客户端
	client := NewClient(conn, c.ClientIP(), claims.UserID)

	// 注册到Hub（非阻塞，如果channel满了则记录错误）
	select {
	case hub.register <- client:
		// 成功注册
	default:
		// Hub的register channel满了，说明Hub处理不过来
		logger.Logger.Error("Hub注册channel已满，无法注册客户端",
			zap.Uint("user_id", claims.UserID),
			zap.String("ip", c.ClientIP()),
		)
		conn.Close()
		return
	}

	// 启动读写goroutine
	go client.readPump()
	go client.writePump()

	// 发送连接成功消息
	client.SendMessage(&Message{
		Type:   "connected",
		UserID: claims.UserID,
		RawData: map[string]interface{}{
			"message": "连接成功",
			"user_id": claims.UserID,
		},
	})
}

// Client WebSocket客户端
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	ip     string
	userID uint
	hub    *Hub
}

func NewClient(conn *websocket.Conn, ip string, userID uint) *Client {
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		ip:     ip,
		userID: userID,
		hub:    hub,
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg *Message) {
	// 构建要发送的消息对象
	sendMsg := map[string]interface{}{
		"type":    msg.Type,
		"room_id": msg.RoomID,
		"user_id": msg.UserID,
	}

	// 如果有RawData，将其添加到消息中（使用raw_data作为key）
	if msg.RawData != nil {
		sendMsg["raw_data"] = msg.RawData
	}

	// 如果有Data（RawMessage），解析后添加
	if len(msg.Data) > 0 {
		var dataMap map[string]interface{}
		if err := json.Unmarshal(msg.Data, &dataMap); err == nil {
			for k, v := range dataMap {
				sendMsg[k] = v
			}
		}
	}

	data, err := json.Marshal(sendMsg)
	if err != nil {
		logger.Logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	select {
	case c.send <- data:
	default:
		logger.Logger.Warn("发送缓冲区满", zap.Uint("user_id", c.userID))
	}
}

// filterGameStateForUser 为指定用户过滤游戏状态（隐藏其他玩家手牌）
func filterGameStateForUser(gameStateData map[string]interface{}, userID uint) map[string]interface{} {
	// 创建新的游戏状态副本
	filtered := make(map[string]interface{})

	// 复制所有字段
	for key, value := range gameStateData {
		if key == "players" {
			// 处理玩家信息
			if players, ok := value.(map[string]interface{}); ok {
				filteredPlayers := make(map[string]interface{})
				for playerKey, playerData := range players {
					if playerInfo, ok := playerData.(map[string]interface{}); ok {
						filteredPlayer := make(map[string]interface{})

						// 复制所有玩家信息
						for k, v := range playerInfo {
							filteredPlayer[k] = v
						}

						// 获取玩家user_id
						var playerUserID uint
						switch v := playerInfo["user_id"].(type) {
						case float64:
							playerUserID = uint(v)
						case int:
							playerUserID = uint(v)
						case uint:
							playerUserID = v
						case int64:
							playerUserID = uint(v)
						}

						// 只返回当前用户的完整手牌，其他玩家的手牌隐藏
						if playerUserID == userID {
							// 自己的手牌完整返回
							// cards 字段保持不变
						} else {
							// 其他玩家的手牌隐藏，返回空数组
							filteredPlayer["cards"] = []interface{}{}
						}

						filteredPlayers[playerKey] = filteredPlayer
					}
				}
				filtered[key] = filteredPlayers
			} else {
				filtered[key] = value
			}
		} else {
			filtered[key] = value
		}
	}

	return filtered
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Logger.Error("WebSocket读取错误",
					zap.Uint("user_id", c.userID),
					zap.Error(err),
				)
			}
			break
		}

		// 解析消息
		var msg Message
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			logger.Logger.Warn("解析消息失败",
				zap.Uint("user_id", c.userID),
				zap.Error(err),
				zap.String("raw", string(rawMessage)),
			)
			c.SendMessage(&Message{
				Type: "error",
				RawData: map[string]interface{}{
					"message": "消息格式错误",
				},
			})
			continue
		}

		// 设置用户ID
		msg.UserID = c.userID

		// 处理消息
		c.handleMessage(&msg)
	}
}

// sendGameStateRecovery 发送游戏状态恢复（断线重连）
func (c *Client) sendGameStateRecovery(roomID string) {
	// 调用API服务获取游戏状态
	cfg := config.Get()
	if cfg == nil {
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/game-state", cfg.Server.Port, roomID)
	resp, err := http.Get(apiURL)
	if err != nil {
		logger.Logger.Warn("获取游戏状态失败",
			zap.Uint("user_id", c.userID),
			zap.String("room_id", roomID),
			zap.Error(err),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	// 发送游戏状态恢复消息
	if data, ok := result["data"].(map[string]interface{}); ok {
		c.SendMessage(&Message{
			Type:   "game_state_recovery",
			RoomID: roomID,
			UserID: c.userID,
			RawData: map[string]interface{}{
				"game_state": data,
				"message":    "游戏状态已恢复",
			},
		})
	}
}

// handleMessage 处理消息
func (c *Client) handleMessage(msg *Message) {
	logger.Logger.Debug("处理消息",
		zap.Uint("user_id", c.userID),
		zap.String("type", msg.Type),
		zap.String("room_id", msg.RoomID),
	)

	switch msg.Type {
	case "join_room":
		// 加入房间
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			if roomID, ok := data["room_id"].(string); ok {
				c.hub.JoinRoom(c, roomID)
				c.SendMessage(&Message{
					Type:   "room_joined",
					RoomID: roomID,
					RawData: map[string]interface{}{
						"message": "加入房间成功",
						"room_id": roomID,
					},
				})

				// 如果房间有游戏状态，发送恢复消息（断线重连）
				c.sendGameStateRecovery(roomID)

				// 广播房间状态更新
				c.hub.broadcast <- &Message{
					Type:   "room_updated",
					RoomID: roomID,
					RawData: map[string]interface{}{
						"user_id": c.userID,
						"action":  "join",
					},
				}
			}
		}

	case "leave_room":
		// 离开房间
		// 获取当前房间ID（如果存在）
		var currentRoomID string
		c.hub.mu.RLock()
		if roomID, ok := c.hub.clientRooms[c]; ok {
			currentRoomID = roomID
		}
		c.hub.mu.RUnlock()

		c.hub.LeaveRoom(c)
		c.SendMessage(&Message{
			Type: "room_left",
			RawData: map[string]interface{}{
				"message": "离开房间成功",
			},
		})

		// 如果有房间ID，广播房间状态更新给房间内其他客户端
		if currentRoomID != "" {
			c.hub.broadcast <- &Message{
				Type:   "room_updated",
				RoomID: currentRoomID,
				RawData: map[string]interface{}{
					"user_id": c.userID,
					"action":  "leave",
				},
			}
		}

	case "ping":
		// 心跳响应
		c.SendMessage(&Message{
			Type: "pong",
			RawData: map[string]interface{}{
				"timestamp": time.Now().Unix(),
			},
		})

	case "reconnect":
		// 断线重连请求
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			if roomID, ok := data["room_id"].(string); ok {
				// 发送游戏状态恢复
				c.sendGameStateRecovery(roomID)
			}
		}

	case "play_cards":
		// 出牌
		c.handlePlayCards(msg)

	case "pass":
		// 过牌
		c.handlePass(msg)

	case "get_game_state":
		// 获取游戏状态
		c.handleGetGameState(msg)

	default:
		logger.Logger.Warn("未知消息类型",
			zap.String("type", msg.Type),
			zap.Uint("user_id", c.userID),
		)
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "未知的消息类型: " + msg.Type,
			},
		})
	}
}

// handlePlayCards 处理出牌
func (c *Client) handlePlayCards(msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析出牌数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		// 尝试从消息的RoomID字段获取
		roomID = msg.RoomID
	}

	cardsData, ok := data["cards"].([]interface{})
	if !ok {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "无效的牌数据",
			},
		})
		return
	}

	// 转换牌数据
	cards := make([]int, 0, len(cardsData))
	for _, card := range cardsData {
		if cardNum, ok := card.(float64); ok {
			cards = append(cards, int(cardNum))
		}
	}

	// 通过HTTP调用API服务的出牌接口
	cfg := config.Get()
	if cfg == nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	// 构建请求数据
	reqData := map[string]interface{}{
		"cards": cards,
	}

	// 通知客户端通过API调用
	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/play", cfg.Server.Port, roomID)
	logger.Logger.Debug("提示客户端通过API调用",
		zap.String("url", apiURL),
		zap.Uint("user_id", c.userID),
	)

	// 发送消息通知客户端通过API调用
	c.SendMessage(&Message{
		Type:   "play_cards_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API调用出牌接口",
			"url":     apiURL,
			"method":  "POST",
			"data":    reqData,
		},
	})

	// 广播给房间内其他客户端（告知有人出牌）
	hub.broadcast <- &Message{
		Type:   "player_playing",
		RoomID: roomID,
		UserID: c.userID,
		RawData: map[string]interface{}{
			"user_id": c.userID,
			"action":  "playing",
		},
	}
}

// handlePass 处理过牌
func (c *Client) handlePass(msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		roomID = msg.RoomID
	}

	// 通知客户端通过API调用
	cfg := config.Get()
	if cfg == nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/pass", cfg.Server.Port, roomID)
	c.SendMessage(&Message{
		Type:   "pass_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API调用过牌接口",
			"url":     apiURL,
			"method":  "POST",
		},
	})

	// 广播给房间内其他客户端
	hub.broadcast <- &Message{
		Type:   "player_passed",
		RoomID: roomID,
		UserID: c.userID,
		RawData: map[string]interface{}{
			"user_id": c.userID,
			"action":  "passed",
		},
	}
}

// handleGetGameState 处理获取游戏状态
func (c *Client) handleGetGameState(msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		roomID = msg.RoomID
	}

	// 通知客户端通过API调用
	cfg := config.Get()
	if cfg == nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/game-state", cfg.Server.Port, roomID)
	c.SendMessage(&Message{
		Type:   "get_game_state_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API获取游戏状态",
			"url":     apiURL,
			"method":  "GET",
		},
	})
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
