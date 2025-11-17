package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// Client WebSocket客户端
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	ip     string
	userID uint
	hub    *Hub
}

func NewClient(conn *websocket.Conn, ip string, userID uint) *Client {
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		ip:     ip,
		userID: userID,
		hub:    hub,
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg *Message) {
	// 构建要发送的消息对象
	sendMsg := map[string]interface{}{
		"type":    msg.Type,
		"room_id": msg.RoomID,
		"user_id": msg.UserID,
	}

	// 如果有RawData，将其添加到消息中（使用raw_data作为key）
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

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
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
		c.handleMessage(&msg)
	}
}

// sendGameStateRecovery 发送游戏状态恢复（断线重连）
func (c *Client) sendGameStateRecovery(roomID string) {
	// 调用API服务获取游戏状态
	cfg := config.Get()
	if cfg == nil {
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/game-state", cfg.Server.Port, roomID)
	resp, err := http.Get(apiURL)
	if err != nil {
		logger.Logger.Warn("获取游戏状态失败",
			zap.Uint("user_id", c.userID),
			zap.String("room_id", roomID),
			zap.Error(err),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	// 发送游戏状态恢复消息
	if data, ok := result["data"].(map[string]interface{}); ok {
		c.SendMessage(&Message{
			Type:   "game_state_recovery",
			RoomID: roomID,
			UserID: c.userID,
			RawData: map[string]interface{}{
				"game_state": data,
				"message":    "游戏状态已恢复",
			},
		})
	}
}

// handleMessage 处理消息
func (c *Client) handleMessage(msg *Message) {
	logger.Logger.Debug("处理消息",
		zap.Uint("user_id", c.userID),
		zap.String("type", msg.Type),
		zap.String("room_id", msg.RoomID),
	)

	switch msg.Type {
	case "join_room":
		// 加入房间
		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			if roomID, ok := data["room_id"].(string); ok {
				c.hub.JoinRoom(c, roomID)
				c.SendMessage(&Message{
					Type:   "room_joined",
					RoomID: roomID,
					RawData: map[string]interface{}{
						"message": "加入房间成功",
						"room_id": roomID,
					},
				})

				// 如果房间有游戏状态，发送恢复消息（断线重连）
				c.sendGameStateRecovery(roomID)

				// 广播房间状态更新
				c.hub.broadcast <- &Message{
					Type:   "room_updated",
					RoomID: roomID,
					RawData: map[string]interface{}{
						"user_id": c.userID,
						"action":  "join",
					},
				}
			}
		}

	case "leave_room":
		// 离开房间
		// 获取当前房间ID（如果存在）
		var currentRoomID string
		c.hub.mu.RLock()
		if roomID, ok := c.hub.clientRooms[c]; ok {
			currentRoomID = roomID
		}
		c.hub.mu.RUnlock()

		c.hub.LeaveRoom(c)
		c.SendMessage(&Message{
			Type: "room_left",
			RawData: map[string]interface{}{
				"message": "离开房间成功",
			},
		})

		// 如果有房间ID，广播房间状态更新给房间内其他客户端
		if currentRoomID != "" {
			c.hub.broadcast <- &Message{
				Type:   "room_updated",
				RoomID: currentRoomID,
				RawData: map[string]interface{}{
					"user_id": c.userID,
					"action":  "leave",
				},
			}
		}

	case "ping":
		// 心跳响应
		c.SendMessage(&Message{
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
				c.sendGameStateRecovery(roomID)
			}
		}

	case "play_cards":
		// 出牌
		c.handlePlayCards(msg)

	case "pass":
		// 过牌
		c.handlePass(msg)

	case "get_game_state":
		// 获取游戏状态
		c.handleGetGameState(msg)

	default:
		logger.Logger.Warn("未知消息类型",
			zap.String("type", msg.Type),
			zap.Uint("user_id", c.userID),
		)
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "未知的消息类型: " + msg.Type,
			},
		})
	}
}

// handlePlayCards 处理出牌
func (c *Client) handlePlayCards(msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析出牌数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		// 尝试从消息的RoomID字段获取
		roomID = msg.RoomID
	}

	cardsData, ok := data["cards"].([]interface{})
	if !ok {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "无效的牌数据",
			},
		})
		return
	}

	// 转换牌数据
	cards := make([]int, 0, len(cardsData))
	for _, card := range cardsData {
		if cardNum, ok := card.(float64); ok {
			cards = append(cards, int(cardNum))
		}
	}

	// 通过HTTP调用API服务的出牌接口
	cfg := config.Get()
	if cfg == nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	// 构建请求数据
	reqData := map[string]interface{}{
		"cards": cards,
	}

	// 通知客户端通过API调用
	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/play", cfg.Server.Port, roomID)
	logger.Logger.Debug("提示客户端通过API调用",
		zap.String("url", apiURL),
		zap.Uint("user_id", c.userID),
	)

	// 发送消息通知客户端通过API调用
	c.SendMessage(&Message{
		Type:   "play_cards_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API调用出牌接口",
			"url":     apiURL,
			"method":  "POST",
			"data":    reqData,
		},
	})

	// 广播给房间内其他客户端（告知有人出牌）
	hub.broadcast <- &Message{
		Type:   "player_playing",
		RoomID: roomID,
		UserID: c.userID,
		RawData: map[string]interface{}{
			"user_id": c.userID,
			"action":  "playing",
		},
	}
}

// handlePass 处理过牌
func (c *Client) handlePass(msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		roomID = msg.RoomID
	}

	// 通知客户端通过API调用
	cfg := config.Get()
	if cfg == nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/pass", cfg.Server.Port, roomID)
	c.SendMessage(&Message{
		Type:   "pass_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API调用过牌接口",
			"url":     apiURL,
			"method":  "POST",
		},
	})

	// 广播给房间内其他客户端
	hub.broadcast <- &Message{
		Type:   "player_passed",
		RoomID: roomID,
		UserID: c.userID,
		RawData: map[string]interface{}{
			"user_id": c.userID,
			"action":  "passed",
		},
	}
}

// handleGetGameState 处理获取游戏状态
func (c *Client) handleGetGameState(msg *Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		roomID = msg.RoomID
	}

	// 通知客户端通过API调用
	cfg := config.Get()
	if cfg == nil {
		c.SendMessage(&Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/game-state", cfg.Server.Port, roomID)
	c.SendMessage(&Message{
		Type:   "get_game_state_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API获取游戏状态",
			"url":     apiURL,
			"method":  "GET",
		},
	})
}

func (c *Client) writePump() {
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
