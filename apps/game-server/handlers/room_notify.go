package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

	// 根据 action 路由到不同的处理函数
	switch req.Action {
	case "room_created":
		handleRoomCreated(c, &req)
		return
	case "room_deleted":
		handleRoomDeleted(c, &req)
		return
	case "game_state_update":
		handleGameStateUpdate(&req)
		handleGenericBroadcast(c, &req)
	case "timer_start":
		handleTimerStart(&req)
		handleGenericBroadcast(c, &req)
	case "timer_stop":
		handleTimerStop(&req)
		handleGenericBroadcast(c, &req)
	case "game_started":
		handleGameStarted(&req)
		handleGenericBroadcast(c, &req)
	case "game_end":
		handleGameEnd(&req)
		handleGenericBroadcast(c, &req)
	default:
		// 其他通用 action（join, leave, ready等）
		handleGenericAction(c, &req)
		return
	}
}

// handleGenericAction 处理通用 action（join, leave, ready等）
func handleGenericAction(c *gin.Context, req *RoomNotifyRequest) {
	handleGenericBroadcast(c, req)
}

// handleGenericBroadcast 处理通用广播（用于需要额外广播的 action）
func handleGenericBroadcast(c *gin.Context, req *RoomNotifyRequest) {
	// 构建广播消息
	msg := newMessageFunc("room_updated", req.RoomID, req.UserID, map[string]interface{}{
		"action":    req.Action,
		"user_id":   req.UserID,
		"room_data": req.RoomData,
	})

	// 如果提供了房间数据，从房间数据中获取所有用户ID并广播给这些用户
	broadcastToRoomPlayers(req, msg)

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
