package messaging

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/internal/messaging"
	"go.uber.org/zap"
)

// Broadcaster 消息广播器
type Broadcaster struct {
	hub        *core.Hub
	messageBus messaging.MessageBus
	instanceID string
}

// NewBroadcaster 创建消息广播器
func NewBroadcaster(hub *core.Hub, messageBus messaging.MessageBus, instanceID string) *Broadcaster {
	return &Broadcaster{
		hub:        hub,
		messageBus: messageBus,
		instanceID: instanceID,
	}
}

// BroadcastMessage 广播消息（会发布到 Kafka）
func (b *Broadcaster) BroadcastMessage(message *core.Message) {
	// 第一步：获取目标客户端列表
	clientList := b.getTargetClients(message)

	// 如果没有目标客户端，直接返回
	if len(clientList) == 0 {
		logger.Logger.Debug("没有目标客户端，跳过广播",
			zap.String("type", message.Type),
			zap.String("room_id", message.RoomID),
		)
		return
	}

	logger.Logger.Info("开始广播消息给客户端",
		zap.String("type", message.Type),
		zap.String("room_id", message.RoomID),
		zap.Int("target_count", len(clientList)),
	)

	// 第二步：序列化消息
	data, err := b.serializeMessage(message)
	if err != nil {
		logger.Logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	// 第三步：发送消息给客户端
	b.sendToClients(clientList, data, message.Type)

	// 第四步：如果启用了消息总线，发布到 Kafka（跨实例通信）
	// 只对特定类型的消息进行跨实例广播（如 room_message, test_message）
	if b.messageBus != nil && (message.Type == "room_message" || message.Type == "test_message") {
		b.publishToKafka(message)
	}
}

// BroadcastMessageLocal 仅本地广播（不发布到 Kafka）
func (b *Broadcaster) BroadcastMessageLocal(message *core.Message) {
	// 获取目标客户端列表
	clientList := b.getTargetClients(message)

	// 如果没有目标客户端，直接返回
	if len(clientList) == 0 {
		return
	}

	// 序列化消息
	data, err := b.serializeMessage(message)
	if err != nil {
		logger.Logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	// 发送消息给客户端
	b.sendToClients(clientList, data, message.Type)
}

// getTargetClients 获取目标客户端列表
func (b *Broadcaster) getTargetClients(message *core.Message) []*core.Client {
	var clientList []*core.Client

	if message.RoomID != "" {
		// 房间广播
		clientList = b.hub.GetRoomClients(message.RoomID)
		if len(clientList) > 0 {
			logger.Logger.Debug("房间广播消息",
				zap.String("room_id", message.RoomID),
				zap.String("type", message.Type),
				zap.Int("clients", len(clientList)),
			)
		}
	} else if message.UserID != 0 {
		// 单播给指定用户
		if client := b.hub.GetUserClient(message.UserID); client != nil {
			clientList = []*core.Client{client}
			logger.Logger.Debug("单播消息",
				zap.Uint("user_id", message.UserID),
				zap.String("type", message.Type),
			)
		}
	} else {
		// RoomID为空且UserID为0，广播给所有客户端（大厅消息）
		userClients := b.hub.GetUserClients()
		clientList = make([]*core.Client, 0, len(userClients))
		for _, client := range userClients {
			clientList = append(clientList, client)
		}
		logger.Logger.Info("准备大厅广播消息",
			zap.String("type", message.Type),
			zap.Int("clients", len(clientList)),
		)
	}

	return clientList
}

// serializeMessage 序列化消息
func (b *Broadcaster) serializeMessage(message *core.Message) ([]byte, error) {
	sendMsg := map[string]interface{}{
		"type":    message.Type,
		"room_id": message.RoomID,
		"user_id": message.UserID,
	}

	// 如果有RawData，添加到raw_data字段
	if message.RawData != nil {
		sendMsg["raw_data"] = message.RawData
	}

	// 如果没有RawData，使用Data字段
	if message.RawData == nil && len(message.Data) > 0 {
		var data interface{}
		if err := json.Unmarshal(message.Data, &data); err == nil {
			sendMsg["raw_data"] = data
		}
	}

	return json.Marshal(sendMsg)
}

// sendToClients 发送消息给客户端
func (b *Broadcaster) sendToClients(clientList []*core.Client, data []byte, msgType string) {
	if len(clientList) < 100 {
		// 小规模：直接发送（避免 goroutine 开销）
		successCount := 0
		for _, client := range clientList {
			select {
			case client.GetSendChannel() <- data:
				successCount++
			default:
				// 发送缓冲区满了，关闭连接
				logger.Logger.Warn("客户端发送缓冲区满，关闭连接",
					zap.Uint("user_id", client.GetUserID()),
				)
				client.CloseSend()
			}
		}
		logger.Logger.Info("消息已发送给客户端",
			zap.String("type", msgType),
			zap.Int("total", len(clientList)),
			zap.Int("success", successCount),
		)
	} else {
		// 大规模：使用 goroutine 并行发送（限制并发数，避免 goroutine 爆炸）
		const maxConcurrent = 50
		sem := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup

		for _, client := range clientList {
			wg.Add(1)
			sem <- struct{}{} // 获取信号量
			go func(c *core.Client) {
				defer wg.Done()
				defer func() { <-sem }() // 释放信号量

				select {
				case c.GetSendChannel() <- data:
					// 发送成功
				default:
					// 发送缓冲区满了，关闭连接
					logger.Logger.Warn("客户端发送缓冲区满，关闭连接",
						zap.Uint("user_id", c.GetUserID()),
					)
					c.CloseSend()
				}
			}(client)
		}
		wg.Wait()
		logger.Logger.Info("消息已发送给客户端（大规模）",
			zap.String("type", msgType),
			zap.Int("total", len(clientList)),
		)
	}
}

// publishToKafka 发布消息到 Kafka
func (b *Broadcaster) publishToKafka(message *core.Message) {
	crossInstanceMsg := map[string]interface{}{
		"type":    message.Type,
		"room_id": message.RoomID,
		"user_id": message.UserID,
	}
	if message.RawData != nil {
		crossInstanceMsg["raw_data"] = message.RawData
	} else if len(message.Data) > 0 {
		var data interface{}
		if err := json.Unmarshal(message.Data, &data); err == nil {
			crossInstanceMsg["raw_data"] = data
		}
	}

	// 异步发布到全局广播 topic（所有实例都能收到）
	go func() {
		broadcastTopic := "broadcast-all"
		if err := b.messageBus.Publish(context.Background(), broadcastTopic, crossInstanceMsg); err != nil {
			logger.Logger.Error("发布跨实例消息失败",
				zap.String("topic", broadcastTopic),
				zap.String("room_id", message.RoomID),
				zap.String("type", message.Type),
				zap.String("instance_id", b.instanceID),
				zap.Error(err),
			)
		} else {
			logger.Logger.Info("发布跨实例消息成功",
				zap.String("topic", broadcastTopic),
				zap.String("room_id", message.RoomID),
				zap.String("type", message.Type),
				zap.String("instance_id", b.instanceID),
			)
		}
	}()
}
