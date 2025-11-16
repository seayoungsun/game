package main

import (
	"encoding/json"
	"sync"

	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
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
}

// Message WebSocket消息
type Message struct {
	Type    string          `json:"type"`    // 消息类型
	RoomID  string          `json:"room_id"` // 房间ID（可选）
	UserID  uint            `json:"user_id"` // 用户ID
	Data    json.RawMessage `json:"data"`    // 消息数据
	RawData interface{}     `json:"-"`       // 原始数据（用于内部处理）
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		rooms:                make(map[string]map[*Client]bool),
		clientRooms:          make(map[*Client]string),
		userClients:          make(map[uint]*Client),
		register:             make(chan *Client, 1000), // 增加缓冲，避免高并发时阻塞
		unregister:           make(chan *Client, 1000), // 增加缓冲，避免高并发时阻塞
		broadcast:            make(chan *Message, 256),
		workerCount:          4, // 默认4个worker goroutine处理注册/注销
		broadcastWorkerCount: 2, // 默认2个worker goroutine处理广播（可以并行，因为使用读锁）
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	// 启动多个 worker goroutine 并行处理注册/注销
	for i := 0; i < h.workerCount; i++ {
		go h.runWorker()
	}

	// 启动多个 worker goroutine 并行处理广播
	// 注意：广播可以并行处理，因为：
	// 1. 读取客户端列表时使用读锁（RWMutex.RLock），多个goroutine可以同时读取
	// 2. 发送消息时不需要锁，只是写入channel（线程安全）
	// 3. 多个广播消息可以并行处理，互不影响
	for i := 0; i < h.broadcastWorkerCount; i++ {
		go h.runBroadcastWorker()
	}

	// 主goroutine保持运行（防止程序退出）
	select {}
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

// runBroadcastWorker 处理广播消息的 worker goroutine
func (h *Hub) runBroadcastWorker() {
	// 使用 for range 从 channel 读取消息（channel关闭时自动退出）
	for message := range h.broadcast {
		h.broadcastMessage(message)
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果用户已有连接，先断开旧连接
	if oldClient, exists := h.userClients[client.userID]; exists {
		logger.Logger.Info("用户已有连接，断开旧连接",
			zap.Uint("user_id", client.userID),
		)
		delete(h.userClients, client.userID)
		if oldRoomID, ok := h.clientRooms[oldClient]; ok {
			h.removeClientFromRoom(oldClient, oldRoomID)
		}
		close(oldClient.send)
	}

	// 注册新连接
	h.userClients[client.userID] = client

	logger.Logger.Info("客户端已注册",
		zap.Uint("user_id", client.userID),
		zap.String("ip", client.ip),
	)
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

	close(client.send)

	logger.Logger.Info("客户端已注销",
		zap.Uint("user_id", client.userID),
	)
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

	logger.Logger.Info("客户端加入房间",
		zap.Uint("user_id", client.userID),
		zap.String("room_id", roomID),
	)
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
			logger.Logger.Info("房间已清空", zap.String("room_id", roomID))
		}
	}
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(message *Message) {
	// 第一步：在锁内快速获取目标客户端列表并复制到切片
	var clientList []*Client
	var targetCount int

	h.mu.RLock()
	if message.RoomID != "" {
		// 房间广播
		if room, exists := h.rooms[message.RoomID]; exists {
			targetCount = len(room)
			clientList = make([]*Client, 0, targetCount)
			for client := range room {
				clientList = append(clientList, client)
			}
			logger.Logger.Debug("房间广播消息",
				zap.String("room_id", message.RoomID),
				zap.String("type", message.Type),
				zap.Int("clients", targetCount),
			)
		}
	} else if message.UserID != 0 {
		// 单播给指定用户
		if client, exists := h.userClients[message.UserID]; exists {
			clientList = []*Client{client}
			targetCount = 1
			logger.Logger.Debug("单播消息",
				zap.Uint("user_id", message.UserID),
				zap.String("type", message.Type),
			)
		}
	} else {
		// RoomID为空且UserID为0，广播给所有客户端（大厅消息）
		targetCount = len(h.userClients)
		clientList = make([]*Client, 0, targetCount)
		for _, client := range h.userClients {
			clientList = append(clientList, client)
		}
		logger.Logger.Debug("大厅广播消息",
			zap.String("type", message.Type),
			zap.Int("clients", targetCount),
		)
	}
	h.mu.RUnlock()

	// 如果没有目标客户端，直接返回
	if len(clientList) == 0 {
		return
	}

	// 第二步：在锁外构建和序列化消息（避免长时间持有锁）
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

	// 序列化消息
	data, err := json.Marshal(sendMsg)
	if err != nil {
		logger.Logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	// 第三步：在锁外发送消息（优化：根据规模选择策略）
	if len(clientList) < 100 {
		// 小规模：直接发送（避免 goroutine 开销）
		for _, client := range clientList {
			select {
			case client.send <- data:
			default:
				// 发送缓冲区满了，关闭连接
				close(client.send)
			}
		}
	} else {
		// 大规模：使用 goroutine 并行发送（限制并发数，避免 goroutine 爆炸）
		const maxConcurrent = 50 // 最多50个goroutine同时发送
		sem := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup

		for _, client := range clientList {
			wg.Add(1)
			sem <- struct{}{} // 获取信号量
			go func(c *Client) {
				defer wg.Done()
				defer func() { <-sem }() // 释放信号量

				select {
				case c.send <- data:
				default:
					// 发送缓冲区满了，关闭连接
					close(c.send)
				}
			}(client)
		}
		wg.Wait()
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
