package core

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// MessageHandlerInterface 消息处理器接口（避免循环依赖）
type MessageHandlerInterface interface {
	HandleMessage(msg *Message)
}

// Client WebSocket客户端
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	ip        string
	userID    uint
	hub       *Hub
	closeOnce sync.Once // 确保 send channel 只被关闭一次
}

// NewClient 创建新的客户端
func NewClient(conn *websocket.Conn, ip string, userID uint, hub *Hub) *Client {
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		ip:     ip,
		userID: userID,
		hub:    hub,
	}
}

// CloseSend 安全地关闭 send channel（确保只关闭一次）
func (c *Client) CloseSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

// closeSend 内部方法（保持向后兼容）
func (c *Client) closeSend() {
	c.CloseSend()
}

// GetConn 获取 WebSocket 连接
func (c *Client) GetConn() *websocket.Conn {
	return c.conn
}

// GetSendChannel 获取发送通道
func (c *Client) GetSendChannel() chan<- []byte {
	return c.send
}

// GetIP 获取客户端IP
func (c *Client) GetIP() string {
	return c.ip
}

// GetUserID 获取用户ID
func (c *Client) GetUserID() uint {
	return c.userID
}

// GetHub 获取 Hub
func (c *Client) GetHub() *Hub {
	return c.hub
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg *Message) {
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
	case c.send <- data:
	default:
		logger.Logger.Warn("发送缓冲区满", zap.Uint("user_id", c.userID))
	}
}

// ReadPump 读取消息
func (c *Client) ReadPump(messageHandler MessageHandlerInterface) {
	defer func() {
		c.hub.GetUnregisterChannel() <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Logger.Error("WebSocket读取错误",
					zap.Uint("user_id", c.userID),
					zap.Error(err),
				)
			}
			break
		}

		// 解析消息
		var msg Message
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			logger.Logger.Warn("解析消息失败",
				zap.Uint("user_id", c.userID),
				zap.Error(err),
				zap.String("raw", string(rawMessage)),
			)
			c.SendMessage(&Message{
				Type: "error",
				RawData: map[string]interface{}{
					"message": "消息格式错误",
				},
			})
			continue
		}

		// 设置用户ID
		msg.UserID = c.userID

		// 处理消息
		messageHandler.HandleMessage(&msg)
	}
}

// WritePump 发送消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
