package messaging

import (
	"context"
	"encoding/json"

	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/internal/messaging"
	"go.uber.org/zap"
)

// KafkaHandler 处理 Kafka 消息
type KafkaHandler struct {
	hub         *core.Hub
	broadcaster *Broadcaster
	messageBus  messaging.MessageBus
	instanceID  string
}

// NewKafkaHandler 创建 Kafka 处理器
func NewKafkaHandler(hub *core.Hub, broadcaster *Broadcaster, messageBus messaging.MessageBus, instanceID string) *KafkaHandler {
	return &KafkaHandler{
		hub:         hub,
		broadcaster: broadcaster,
		messageBus:  messageBus,
		instanceID:  instanceID,
	}
}

// HandleCrossInstanceBroadcast 处理跨实例广播消息（来自 Kafka）
func (h *KafkaHandler) HandleCrossInstanceBroadcast(topic string, message []byte) error {
	var wrapper map[string]interface{}
	if err := json.Unmarshal(message, &wrapper); err != nil {
		return err
	}

	// Kafka 消息被包装在 data 字段中：{"source_instance": "...", "timestamp": ..., "data": {...}}
	var msg map[string]interface{}
	if data, ok := wrapper["data"].(map[string]interface{}); ok {
		msg = data
	} else {
		// 如果没有 data 字段，直接使用 wrapper（兼容旧格式）
		msg = wrapper
	}

	// 检查是否是自己的消息（避免重复处理）
	sourceInstance, _ := wrapper["source_instance"].(string)
	if sourceInstance == h.instanceID {
		logger.Logger.Debug("忽略自己发布的消息",
			zap.String("topic", topic),
			zap.String("instance_id", h.instanceID),
		)
		return nil
	}

	// 获取消息类型和房间ID
	msgType, _ := msg["type"].(string)
	roomID, _ := msg["room_id"].(string)

	// 构建内部 Message 格式
	userID := core.GetUint(msg, "user_id")
	internalMsg := &core.Message{
		Type:   core.GetString(msg, "type"),
		RoomID: roomID,
		UserID: userID,
	}

	if rawData, ok := msg["raw_data"].(map[string]interface{}); ok {
		internalMsg.RawData = rawData
	}

	logger.Logger.Info("收到跨实例广播消息",
		zap.String("type", msgType),
		zap.String("room_id", roomID),
		zap.Uint("user_id", userID),
		zap.String("source_instance", sourceInstance),
		zap.String("instance_id", h.instanceID),
		zap.Any("raw_data", internalMsg.RawData),
	)

	// 如果有房间ID，只广播给该房间的客户端
	if roomID != "" {
		roomClients := h.hub.GetRoomClients(roomID)
		if len(roomClients) > 0 {
			logger.Logger.Info("广播消息给房间客户端",
				zap.String("room_id", roomID),
				zap.Int("client_count", len(roomClients)),
			)
			// 通过 broadcaster 广播消息（不再次发布到 Kafka，避免循环）
			h.broadcaster.BroadcastMessageLocal(internalMsg)
		} else {
			logger.Logger.Debug("没有本地客户端在房间中，忽略消息",
				zap.String("room_id", roomID),
			)
		}
	} else {
		// room_id 为空
		// test_message 和 room_message 类型应该广播给所有客户端（用于跨实例消息传播测试）
		if msgType == "test_message" || msgType == "room_message" {
			// 广播给所有客户端（大厅广播）
			totalClients := h.hub.GetConnectionCount()
			if totalClients > 0 {
				logger.Logger.Info("广播消息给所有客户端（大厅广播）",
					zap.String("type", msgType),
					zap.Int("total_clients", totalClients),
				)
				// 确保 UserID 为 0，以便 broadcastMessage 走大厅广播逻辑
				internalMsg.UserID = 0
				// 通过 broadcaster 广播消息（不再次发布到 Kafka，避免循环）
				h.broadcaster.BroadcastMessageLocal(internalMsg)
			} else {
				logger.Logger.Debug("没有本地客户端连接，忽略大厅广播消息",
					zap.String("type", msgType),
				)
			}
		} else {
			logger.Logger.Debug("room_id 为空且不是测试消息类型，忽略消息",
				zap.String("type", msgType),
			)
		}
	}

	return nil
}

// HandleRoomBroadcast 处理房间广播消息（来自 Kafka）
func (h *KafkaHandler) HandleRoomBroadcast(topic string, message []byte) error {
	var wrapper map[string]interface{}
	if err := json.Unmarshal(message, &wrapper); err != nil {
		return err
	}

	// Kafka 消息被包装在 data 字段中：{"source_instance": "...", "timestamp": ..., "data": {...}}
	var msg map[string]interface{}
	if data, ok := wrapper["data"].(map[string]interface{}); ok {
		msg = data
	} else {
		// 如果没有 data 字段，直接使用 wrapper（兼容旧格式）
		msg = wrapper
	}

	// 检查是否是自己的消息（避免重复处理）
	sourceInstance, _ := wrapper["source_instance"].(string)
	if sourceInstance == h.instanceID {
		logger.Logger.Debug("忽略自己发布的消息",
			zap.String("topic", topic),
			zap.String("instance_id", h.instanceID),
		)
		return nil
	}

	// 检查是否有本地客户端在这个房间
	roomID, _ := msg["room_id"].(string)
	if roomID == "" {
		return nil
	}

	roomClients := h.hub.GetRoomClients(roomID)
	if len(roomClients) == 0 {
		logger.Logger.Debug("没有本地客户端在房间中，忽略消息",
			zap.String("room_id", roomID),
			zap.String("topic", topic),
		)
		return nil
	}

	// 构建内部 Message 格式
	internalMsg := &core.Message{
		Type:   core.GetString(msg, "type"),
		RoomID: roomID,
		UserID: core.GetUint(msg, "user_id"),
	}

	if rawData, ok := msg["raw_data"].(map[string]interface{}); ok {
		internalMsg.RawData = rawData
	}

	logger.Logger.Info("收到跨实例房间广播消息",
		zap.String("room_id", roomID),
		zap.String("type", internalMsg.Type),
		zap.String("source_instance", sourceInstance),
		zap.Int("local_clients", len(roomClients)),
	)

	// 通过 broadcaster 广播消息（不再次发布到 Kafka，避免循环）
	h.broadcaster.BroadcastMessageLocal(internalMsg)

	return nil
}

// PublishSystemMessage 发布系统消息到 Kafka（用于跨实例通知）
func (h *KafkaHandler) PublishSystemMessage(msgType, roomID string, data map[string]interface{}) error {
	if h.messageBus == nil {
		return nil // 消息总线未启用，忽略
	}

	systemMsg := map[string]interface{}{
		"type":    msgType,
		"room_id": roomID,
	}
	for k, v := range data {
		systemMsg[k] = v
	}

	topic := "game-system-notify"
	if err := h.messageBus.Publish(context.Background(), topic, systemMsg); err != nil {
		logger.Logger.Error("发布系统消息失败",
			zap.String("topic", topic),
			zap.String("type", msgType),
			zap.String("room_id", roomID),
			zap.Error(err),
		)
		return err
	}

	logger.Logger.Info("发布系统消息成功",
		zap.String("topic", topic),
		zap.String("type", msgType),
		zap.String("room_id", roomID),
	)

	return nil
}
