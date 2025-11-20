package adapters

import (
	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/apps/game-server/handlers"
	"github.com/kaifa/game-platform/apps/game-server/messaging"
	"github.com/kaifa/game-platform/apps/game-server/services"
)

// HubAdapter Hub 适配器，实现 handlers.HubInterface
type HubAdapter struct {
	hub          *core.Hub
	broadcaster  *messaging.Broadcaster
	kafkaHandler *messaging.KafkaHandler
}

// NewHubAdapter 创建 Hub 适配器
func NewHubAdapter(hub *core.Hub, broadcaster *messaging.Broadcaster, kafkaHandler *messaging.KafkaHandler) *HubAdapter {
	return &HubAdapter{
		hub:          hub,
		broadcaster:  broadcaster,
		kafkaHandler: kafkaHandler,
	}
}

// RegisterClient 注册客户端
func (a *HubAdapter) RegisterClient(client handlers.ClientInterface) bool {
	// 从适配器中获取原始 Client
	ca, ok := client.(*ClientAdapter)
	if !ok {
		return false
	}
	select {
	case a.hub.GetRegisterChannel() <- ca.client:
		return true
	default:
		return false
	}
}

// GetUserClient 获取用户客户端
func (a *HubAdapter) GetUserClient(userID uint) handlers.ClientInterface {
	client := a.hub.GetUserClient(userID)
	if client == nil {
		return nil
	}
	return &ClientAdapter{client: client}
}

// GetRoomClients 获取房间客户端列表
func (a *HubAdapter) GetRoomClients(roomID string) []handlers.ClientInterface {
	clients := a.hub.GetRoomClients(roomID)
	result := make([]handlers.ClientInterface, len(clients))
	for i, client := range clients {
		result[i] = &ClientAdapter{client: client}
	}
	return result
}

// BroadcastMessage 广播消息
func (a *HubAdapter) BroadcastMessage(msg handlers.MessageInterface) {
	// 从适配器中获取原始 Message
	ma, ok := msg.(*MessageAdapter)
	if !ok {
		return
	}
	// 使用 broadcaster 广播消息
	a.broadcaster.BroadcastMessage(ma.msg)
}

// PublishSystemMessage 发布系统消息
func (a *HubAdapter) PublishSystemMessage(msgType, roomID string, data map[string]interface{}) error {
	if a.kafkaHandler != nil {
		return a.kafkaHandler.PublishSystemMessage(msgType, roomID, data)
	}
	return nil
}

// ClientAdapter Client 适配器，实现 handlers.ClientInterface
type ClientAdapter struct {
	client         *core.Client
	messageHandler *services.MessageHandler
}

// NewClientAdapter 创建 Client 适配器
func NewClientAdapter(client *core.Client, messageHandler *services.MessageHandler) *ClientAdapter {
	return &ClientAdapter{
		client:         client,
		messageHandler: messageHandler,
	}
}

// Start 启动客户端（启动读写 goroutine）
func (a *ClientAdapter) Start() {
	go a.client.ReadPump(a.messageHandler)
	go a.client.WritePump()
}

// SendMessage 发送消息
func (a *ClientAdapter) SendMessage(msg handlers.MessageInterface) {
	// 从适配器中获取原始 Message
	ma, ok := msg.(*MessageAdapter)
	if !ok {
		return
	}
	a.client.SendMessage(ma.msg)
}

// GetUserID 获取用户ID
func (a *ClientAdapter) GetUserID() uint {
	return a.client.GetUserID()
}

// MessageAdapter Message 适配器，实现 handlers.MessageInterface
type MessageAdapter struct {
	msg *core.Message
}

// NewMessageAdapter 创建 Message 适配器
func NewMessageAdapter(msg *core.Message) *MessageAdapter {
	return &MessageAdapter{msg: msg}
}

// GetType 获取消息类型
func (a *MessageAdapter) GetType() string {
	return a.msg.Type
}

// GetRoomID 获取房间ID
func (a *MessageAdapter) GetRoomID() string {
	return a.msg.RoomID
}

// GetUserID 获取用户ID
func (a *MessageAdapter) GetUserID() uint {
	return a.msg.UserID
}

// GetRawData 获取原始数据
func (a *MessageAdapter) GetRawData() interface{} {
	return a.msg.RawData
}
