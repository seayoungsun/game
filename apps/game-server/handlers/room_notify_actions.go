package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/apps/game-server/utils"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// handleRoomCreated 处理房间创建通知
func handleRoomCreated(c *gin.Context, req *RoomNotifyRequest) {
	if req.RoomData == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "房间数据不能为空",
		})
		return
	}

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

	// 发布系统消息到 Kafka，通知所有实例订阅该房间的广播频道
	if err := hubInstance.PublishSystemMessage("room_created", req.RoomID, nil); err != nil {
		logger.Logger.Error("发布房间创建系统消息失败",
			zap.String("room_id", req.RoomID),
			zap.Error(err),
		)
	}

	logger.Logger.Info("房间创建通知已广播",
		zap.String("room_id", req.RoomID),
		zap.Uint("creator_id", req.UserID),
	)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "通知已发送",
	})
}

// handleRoomDeleted 处理房间删除通知
func handleRoomDeleted(c *gin.Context, req *RoomNotifyRequest) {
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
}

// handleGameStateUpdate 处理游戏状态更新
func handleGameStateUpdate(req *RoomNotifyRequest) {
	if req.RoomData == nil {
		return
	}

	gameStateData, ok := req.RoomData["game_state"].(map[string]interface{})
	if !ok {
		return
	}

	isRaw, _ := req.RoomData["is_raw"].(bool)

	if isRaw {
		// 需要为每个用户过滤手牌，发送给房间内所有客户端
		broadcastFilteredGameState(req.RoomID, req.UserID, gameStateData)

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

// handleTimerStart 处理计时器开始
func handleTimerStart(req *RoomNotifyRequest) {
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

// handleTimerStop 处理计时器停止
func handleTimerStop(req *RoomNotifyRequest) {
	hubInstance.BroadcastMessage(newMessageFunc("timer_stop", req.RoomID, req.UserID, map[string]interface{}{
		"message": "计时器已停止",
	}))
}

// handleGameStarted 处理游戏开始
func handleGameStarted(req *RoomNotifyRequest) {
	if req.RoomData == nil {
		return
	}

	gameStateData, ok := req.RoomData["game_state"].(map[string]interface{})
	if !ok {
		return
	}

	// 获取玩家列表
	playersToNotify := extractPlayersFromRoomData(req.RoomData, gameStateData)

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

// handleGameEnd 处理游戏结束
func handleGameEnd(req *RoomNotifyRequest) {
	if req.RoomData == nil {
		return
	}

	var gameStateData map[string]interface{}
	if gs, ok := req.RoomData["game_state"].(map[string]interface{}); ok {
		gameStateData = gs
	}

	// 预先获取结算数据（如果存在）
	var settlementData map[string]interface{}
	hasSettlement := false
	if sd, ok := req.RoomData["settlement"].(map[string]interface{}); ok {
		settlementData = sd
		hasSettlement = true
	}

	// 从game_state中获取所有玩家ID，确保所有玩家都收到消息
	var playersToNotify []uint
	if gameStateData != nil {
		playersToNotify = extractPlayersFromGameState(gameStateData)
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
