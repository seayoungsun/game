package services

import (
	"encoding/json"
	"time"

	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/apps/game-server/messaging"
	"github.com/kaifa/game-platform/apps/game-server/services/game"
	"github.com/kaifa/game-platform/apps/game-server/services/room"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// MessageHandler 消息处理器
type MessageHandler struct {
	client      *core.Client
	hub         *core.Hub
	broadcaster *messaging.Broadcaster
	roomService *room.Service
	gameService *game.Service
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(client *core.Client, hub *core.Hub, broadcaster *messaging.Broadcaster) *MessageHandler {
	return &MessageHandler{
		client:      client,
		hub:         hub,
		broadcaster: broadcaster,
		roomService: room.NewService(client, hub, broadcaster),
		gameService: game.NewService(client, hub, broadcaster),
	}
}

// HandleMessage 处理消息
func (h *MessageHandler) HandleMessage(msg *core.Message) {
	logger.Logger.Debug("处理消息",
		zap.Uint("user_id", h.client.GetUserID()),
		zap.String("type", msg.Type),
		zap.String("room_id", msg.RoomID),
	)

	switch msg.Type {
	case "join_room":
		h.roomService.HandleJoinRoom(msg)

	case "leave_room":
		h.roomService.HandleLeaveRoom(msg)

	case "ping":
		// 心跳响应
		h.sendMessage(&core.Message{
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
				h.roomService.SendGameStateRecovery(roomID)
			}
		}

	case "play_cards":
		// 出牌
		h.gameService.HandlePlayCards(msg)

	case "pass":
		// 过牌
		h.gameService.HandlePass(msg)

	case "get_game_state":
		// 获取游戏状态
		h.gameService.HandleGetGameState(msg)

	case "test_message", "room_message":
		// 测试消息/房间消息（用于跨实例消息传播测试）
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			roomID := msg.RoomID
			if roomID == "" {
				if rid, ok := data["room_id"].(string); ok {
					roomID = rid
				}
			}
			// 转发到广播通道（会触发跨实例消息传播）
			h.broadcaster.BroadcastMessage(&core.Message{
				Type:    msg.Type,
				RoomID:  roomID,
				UserID:  h.client.GetUserID(),
				RawData: data,
			})
			h.sendMessage(&core.Message{
				Type: "message_sent",
				RawData: map[string]interface{}{
					"message": "消息已发送",
					"room_id": roomID,
				},
			})
		}

	default:
		logger.Logger.Warn("未知消息类型",
			zap.String("type", msg.Type),
			zap.Uint("user_id", h.client.GetUserID()),
		)
		h.sendMessage(&core.Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "未知的消息类型: " + msg.Type,
			},
		})
	}
}

// sendMessage 发送消息给客户端
func (h *MessageHandler) sendMessage(msg *core.Message) {
	// 构建要发送的消息对象
	sendMsg := map[string]interface{}{
		"type":    msg.Type,
		"room_id": msg.RoomID,
		"user_id": msg.UserID,
	}

	// 如果有RawData，将其添加到消息中
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
	case h.client.GetSendChannel() <- data:
	default:
		logger.Logger.Warn("发送缓冲区满", zap.Uint("user_id", h.client.GetUserID()))
	}
}
