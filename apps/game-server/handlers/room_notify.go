package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/apps/game-server/utils"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// RoomNotifyRequest 房间通知请求
type RoomNotifyRequest struct {
	RoomID   string                 `json:"room_id" binding:"required"`
	Action   string                 `json:"action" binding:"required"` // join, leave, ready, cancel_ready, start, game_end, room_created, room_deleted
	UserID   uint                   `json:"user_id"`                   // 用户ID（可选，game_end和room_deleted时可能为0）
	RoomData map[string]interface{} `json:"room_data,omitempty"`       // 房间数据（可选）
}

// HandleRoomNotify 处理房间通知（供API服务调用）
func HandleRoomNotify(c *gin.Context) {
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
		hubInstance.BroadcastMessage(newMessageFunc("room_created", "", 0, map[string]interface{}{
			"message": "新房间已创建",
			"room":    roomData,
		}))

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
		hubInstance.BroadcastMessage(newMessageFunc("room_deleted", "", 0, map[string]interface{}{
			"message": "房间已解散",
			"room_id": req.RoomID,
		}))

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
	msg := newMessageFunc("room_updated", req.RoomID, req.UserID, map[string]interface{}{
		"action":    req.Action,
		"user_id":   req.UserID,
		"room_data": req.RoomData,
	})

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
							filteredState := utils.FilterGameStateForUser(gameStateData, userIDUint)

							// 发送给该用户的客户端
							if client := hubInstance.GetUserClient(userIDUint); client != nil {
								client.SendMessage(newMessageFunc("game_state_update", req.RoomID, userIDUint, map[string]interface{}{
									"game_state": filteredState,
								}))
							}
						}

						_ = playerKey // 避免未使用变量
					}
				}

				// 也广播给房间内的所有客户端（通用广播）
				hubInstance.BroadcastMessage(newMessageFunc("game_state_update", req.RoomID, req.UserID, map[string]interface{}{
					"game_state": gameStateData, // 发送原始数据，客户端需要自己过滤
					"note":       "需要客户端过滤手牌",
				}))
			} else {
				// 已经是过滤后的状态，直接广播
				hubInstance.BroadcastMessage(newMessageFunc("game_state_update", req.RoomID, req.UserID, map[string]interface{}{
					"game_state": gameStateData,
				}))
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

		hubInstance.BroadcastMessage(newMessageFunc("timer_start", req.RoomID, req.UserID, map[string]interface{}{
			"user_id":    req.UserID,
			"timeout":    int(timeout),
			"start_time": int64(startTime),
			"message":    "开始倒计时",
		}))
	}

	// 如果action是timer_stop，广播计时器停止
	if req.Action == "timer_stop" {
		hubInstance.BroadcastMessage(newMessageFunc("timer_stop", req.RoomID, req.UserID, map[string]interface{}{
			"message": "计时器已停止",
		}))
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
				if client := hubInstance.GetUserClient(userIDUint); client != nil {
					filteredState := utils.FilterGameStateForUser(gameStateData, userIDUint)

					// 发送过滤后的游戏状态给该客户端
					client.SendMessage(newMessageFunc("game_state_update", req.RoomID, userIDUint, map[string]interface{}{
						"game_state": filteredState,
						"message":    "游戏已开始",
					}))

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
			clients := hubInstance.GetRoomClients(req.RoomID)
			for _, client := range clients {
				if client != nil {
					// 检查是否已经发送过（避免重复）
					alreadySent := false
					for _, userID := range playersToNotify {
						if userID == client.GetUserID() {
							alreadySent = true
							break
						}
					}

					if !alreadySent {
						filteredState := utils.FilterGameStateForUser(gameStateData, client.GetUserID())

						client.SendMessage(newMessageFunc("game_state_update", req.RoomID, client.GetUserID(), map[string]interface{}{
							"game_state": filteredState,
							"message":    "游戏已开始",
						}))
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
				if client := hubInstance.GetUserClient(userIDUint); client != nil {
					// 为每个玩家构建个性化的消息（包含过滤后的游戏状态）
					personalData := make(map[string]interface{})
					if gameStateData != nil {
						filteredState := utils.FilterGameStateForUser(gameStateData, userIDUint)
						personalData["game_state"] = filteredState
					}
					if hasSettlement {
						personalData["settlement"] = settlementData
					}
					personalData["message"] = "游戏已结束，请查看结算结果"

					client.SendMessage(newMessageFunc("game_end", req.RoomID, userIDUint, personalData))
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
		clients := hubInstance.GetRoomClients(req.RoomID)
		for _, client := range clients {
			if client != nil {
				// 检查是否已经发送过（避免重复）
				alreadySent := false
				for _, userID := range playersToNotify {
					if userID == client.GetUserID() {
						alreadySent = true
						break
					}
				}

				if !alreadySent {
					// 为每个客户端构建个性化的消息（包含过滤后的游戏状态）
					personalData := make(map[string]interface{})
					if gameStateData != nil {
						filteredState := utils.FilterGameStateForUser(gameStateData, client.GetUserID())
						personalData["game_state"] = filteredState
					}
					if hasSettlement {
						personalData["settlement"] = settlementData
					}
					personalData["message"] = "游戏已结束，请查看结算结果"

					client.SendMessage(newMessageFunc("game_end", req.RoomID, client.GetUserID(), personalData))
					logger.Logger.Info("已发送游戏结束消息给房间内客户端",
						zap.Uint("user_id", client.GetUserID()),
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
						if client := hubInstance.GetUserClient(userIDUint); client != nil {
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
	hubInstance.BroadcastMessage(msg)

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
