package core

import (
	"sync"

	"github.com/kaifa/game-platform/internal/messaging"
)

// Hub 管理所有WebSocket连接和房间
type Hub struct {
	// 房间映射：roomID -> []*Client
	rooms map[string]map[*Client]bool

	// 客户端到房间的映射：client -> roomID
	clientRooms map[*Client]string

	// 用户到客户端的映射：userID -> *Client
	userClients map[uint]*Client

	// 注册通道
	register chan *Client

	// 注销通道
	unregister chan *Client

	// 广播消息通道
	broadcast chan *Message

	// 互斥锁
	mu sync.RWMutex

	// Worker 数量（用于并行处理注册/注销）
	workerCount int

	// 广播 Worker 数量（用于并行处理广播消息）
	broadcastWorkerCount int

	// 消息总线（用于跨实例通信）
	messageBus messaging.MessageBus

	// 实例ID（用于消息去重）
	instanceID string
}

// NewHub 创建新的Hub
func NewHub(messageBus messaging.MessageBus, instanceID string) *Hub {
	return &Hub{
		rooms:                make(map[string]map[*Client]bool),
		clientRooms:          make(map[*Client]string),
		userClients:          make(map[uint]*Client),
		register:             make(chan *Client, 1000),
		unregister:           make(chan *Client, 1000),
		broadcast:            make(chan *Message, 256),
		workerCount:          4,
		broadcastWorkerCount: 2,
		messageBus:           messageBus,
		instanceID:           instanceID,
	}
}

// GetBroadcastChannel 获取广播通道（供外部使用，返回双向channel以便读取）
func (h *Hub) GetBroadcastChannel() chan *Message {
	return h.broadcast
}

// GetRegisterChannel 获取注册通道（供外部使用）
func (h *Hub) GetRegisterChannel() chan<- *Client {
	return h.register
}

// GetUnregisterChannel 获取注销通道（供外部使用）
func (h *Hub) GetUnregisterChannel() chan<- *Client {
	return h.unregister
}

// GetMessageBus 获取消息总线
func (h *Hub) GetMessageBus() messaging.MessageBus {
	return h.messageBus
}

// GetInstanceID 获取实例ID
func (h *Hub) GetInstanceID() string {
	return h.instanceID
}

// runWorker 处理注册/注销的 worker goroutine
func (h *Hub) runWorker() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)
		}
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果用户已有连接，先断开旧连接
	if oldClient, exists := h.userClients[client.userID]; exists {
		delete(h.userClients, client.userID)
		if oldRoomID, ok := h.clientRooms[oldClient]; ok {
			h.removeClientFromRoom(oldClient, oldRoomID)
		}
		oldClient.CloseSend()
	}

	// 注册新连接
	h.userClients[client.userID] = client
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从用户映射中移除
	delete(h.userClients, client.userID)

	// 从房间中移除
	if roomID, ok := h.clientRooms[client]; ok {
		h.removeClientFromRoom(client, roomID)
		delete(h.clientRooms, client)
	}

	// 安全地关闭 send channel
	client.CloseSend()
}

// JoinRoom 加入房间
func (h *Hub) JoinRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果客户端已在其他房间，先离开
	if oldRoomID, ok := h.clientRooms[client]; ok && oldRoomID != roomID {
		h.removeClientFromRoom(client, oldRoomID)
	}

	// 加入新房间
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	h.clientRooms[client] = roomID
}

// LeaveRoom 离开房间
func (h *Hub) LeaveRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if roomID, ok := h.clientRooms[client]; ok {
		h.removeClientFromRoom(client, roomID)
		delete(h.clientRooms, client)
	}
}

// removeClientFromRoom 从房间移除客户端（需要在锁内调用）
func (h *Hub) removeClientFromRoom(client *Client, roomID string) {
	if room, exists := h.rooms[roomID]; exists {
		delete(room, client)
		if len(room) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

// GetRoomClients 获取房间内的所有客户端
func (h *Hub) GetRoomClients(roomID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		return nil
	}

	clients := make([]*Client, 0, len(room))
	for client := range room {
		clients = append(clients, client)
	}
	return clients
}

// GetUserClient 根据用户ID获取客户端
func (h *Hub) GetUserClient(userID uint) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, exists := h.userClients[userID]; exists {
		return client
	}
	return nil
}

// GetConnectionCount 获取当前连接数
func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients)
}

// GetRoomCount 获取房间数量
func (h *Hub) GetRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// GetRooms 获取所有房间（用于调试）
func (h *Hub) GetRooms() map[string]map[*Client]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms
}

// GetUserClients 获取所有用户客户端（用于调试）
func (h *Hub) GetUserClients() map[uint]*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.userClients
}

// StartWorkers 启动 worker goroutines（处理注册/注销）
func (h *Hub) StartWorkers() {
	// 启动多个 worker goroutine 并行处理注册/注销
	for i := 0; i < h.workerCount; i++ {
		go h.runWorker()
	}
}
