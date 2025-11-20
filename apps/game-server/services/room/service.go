package room

import (
	"encoding/json"

	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/apps/game-server/messaging"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// Service 房间服务
type Service struct {
	client      *core.Client
	hub         *core.Hub
	broadcaster *messaging.Broadcaster
}

// NewService 创建房间服务
func NewService(client *core.Client, hub *core.Hub, broadcaster *messaging.Broadcaster) *Service {
	return &Service{
		client:      client,
		hub:         hub,
		broadcaster: broadcaster,
	}
}

// HandleJoinRoom 处理加入房间
func (s *Service) HandleJoinRoom(msg *core.Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err == nil {
		if roomID, ok := data["room_id"].(string); ok {
			s.hub.JoinRoom(s.client, roomID)
			s.sendMessage(&core.Message{
				Type:   "room_joined",
				RoomID: roomID,
				RawData: map[string]interface{}{
					"message": "加入房间成功",
					"room_id": roomID,
				},
			})

			// 如果房间有游戏状态，发送恢复消息（断线重连）
			s.SendGameStateRecovery(roomID)

			// 广播房间状态更新
			s.broadcaster.BroadcastMessage(&core.Message{
				Type:   "room_updated",
				RoomID: roomID,
				RawData: map[string]interface{}{
					"user_id": s.client.GetUserID(),
					"action":  "join",
				},
			})
		}
	}
}

// HandleLeaveRoom 处理离开房间
func (s *Service) HandleLeaveRoom(msg *core.Message) {
	// 获取当前房间ID（如果存在）
	var currentRoomID string
	// 通过遍历查找客户端所在的房间
	for roomID, clients := range s.hub.GetRooms() {
		if _, exists := clients[s.client]; exists {
			currentRoomID = roomID
			break
		}
	}

	s.hub.LeaveRoom(s.client)
	s.sendMessage(&core.Message{
		Type: "room_left",
		RawData: map[string]interface{}{
			"message": "离开房间成功",
		},
	})

	// 如果有房间ID，广播房间状态更新给房间内其他客户端
	if currentRoomID != "" {
		s.broadcaster.BroadcastMessage(&core.Message{
			Type:   "room_updated",
			RoomID: currentRoomID,
			RawData: map[string]interface{}{
				"user_id": s.client.GetUserID(),
				"action":  "leave",
			},
		})
	}
}

// SendGameStateRecovery 发送游戏状态恢复（断线重连）
func (s *Service) SendGameStateRecovery(roomID string) {
	// TODO: 从 API Server 或 Redis 获取游戏状态
	// 目前暂时不实现，等待后续集成
	// gameState := utils.GetGameState(roomID)
	// if gameState != nil {
	// 	s.sendMessage(&core.Message{
	// 		Type:   "game_state_recovery",
	// 		RoomID: roomID,
	// 		RawData: map[string]interface{}{
	// 			"game_state": gameState,
	// 		},
	// 	})
	// }
}

// sendMessage 发送消息给客户端
func (s *Service) sendMessage(msg *core.Message) {
	// 构建要发送的消息对象
	sendMsg := map[string]interface{}{
		"type":    msg.Type,
		"room_id": msg.RoomID,
		"user_id": msg.UserID,
	}

	if msg.RawData != nil {
		sendMsg["raw_data"] = msg.RawData
	}

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
	case s.client.GetSendChannel() <- data:
	default:
		logger.Logger.Warn("发送缓冲区满", zap.Uint("user_id", s.client.GetUserID()))
	}
}
