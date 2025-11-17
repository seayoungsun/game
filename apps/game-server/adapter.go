package main

import (
	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/apps/game-server/handlers"
)

// initHandlers 初始化 handlers 包的依赖
func initHandlers() {
	// 创建适配器
	hubAdapter := &hubAdapter{h: hub}
	clientAdapterFunc := func(conn *websocket.Conn, ip string, userID uint) handlers.ClientInterface {
		return &clientAdapter{client: NewClient(conn, ip, userID)}
	}
	messageAdapterFunc := func(msgType, roomID string, userID uint, rawData interface{}) handlers.MessageInterface {
		return &messageAdapter{msg: &Message{
			Type:    msgType,
			RoomID:  roomID,
			UserID:  userID,
			RawData: rawData,
		}}
	}

	// 注入依赖
	handlers.SetUpgrader(&upgrader)
	handlers.SetHub(hubAdapter)
	handlers.SetNewClientFunc(clientAdapterFunc)
	handlers.SetNewMessageFunc(messageAdapterFunc)
}

// hubAdapter Hub 适配器，实现 handlers.HubInterface
type hubAdapter struct {
	h *Hub
}

func (a *hubAdapter) RegisterClient(client handlers.ClientInterface) bool {
	// 从适配器中获取原始 Client
	ca, ok := client.(*clientAdapter)
	if !ok {
		return false
	}
	select {
	case a.h.register <- ca.client:
		return true
	default:
		return false
	}
}

func (a *hubAdapter) GetUserClient(userID uint) handlers.ClientInterface {
	client := a.h.GetUserClient(userID)
	if client == nil {
		return nil
	}
	return &clientAdapter{client: client}
}

func (a *hubAdapter) GetRoomClients(roomID string) []handlers.ClientInterface {
	clients := a.h.GetRoomClients(roomID)
	result := make([]handlers.ClientInterface, len(clients))
	for i, client := range clients {
		result[i] = &clientAdapter{client: client}
	}
	return result
}

func (a *hubAdapter) BroadcastMessage(msg handlers.MessageInterface) {
	// 从适配器中获取原始 Message
	ma, ok := msg.(*messageAdapter)
	if !ok {
		return
	}
	a.h.broadcast <- ma.msg
}

// clientAdapter Client 适配器，实现 handlers.ClientInterface
type clientAdapter struct {
	client *Client
}

func (a *clientAdapter) Start() {
	go a.client.readPump()
	go a.client.writePump()
}

func (a *clientAdapter) SendMessage(msg handlers.MessageInterface) {
	// 从适配器中获取原始 Message
	ma, ok := msg.(*messageAdapter)
	if !ok {
		return
	}
	a.client.SendMessage(ma.msg)
}

func (a *clientAdapter) GetUserID() uint {
	return a.client.userID
}

// messageAdapter Message 适配器，实现 handlers.MessageInterface
type messageAdapter struct {
	msg *Message
}

func (a *messageAdapter) GetType() string {
	return a.msg.Type
}

func (a *messageAdapter) GetRoomID() string {
	return a.msg.RoomID
}

func (a *messageAdapter) GetUserID() uint {
	return a.msg.UserID
}

func (a *messageAdapter) GetRawData() interface{} {
	return a.msg.RawData
}
