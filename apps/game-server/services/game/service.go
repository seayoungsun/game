package game

import (
	"encoding/json"
	"fmt"

	"github.com/kaifa/game-platform/apps/game-server/core"
	"github.com/kaifa/game-platform/apps/game-server/messaging"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// Service 游戏服务
type Service struct {
	client      *core.Client
	hub         *core.Hub
	broadcaster *messaging.Broadcaster
}

// NewService 创建游戏服务
func NewService(client *core.Client, hub *core.Hub, broadcaster *messaging.Broadcaster) *Service {
	return &Service{
		client:      client,
		hub:         hub,
		broadcaster: broadcaster,
	}
}

// HandlePlayCards 处理出牌
func (s *Service) HandlePlayCards(msg *core.Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		s.sendMessage(&core.Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "解析出牌数据失败",
			},
		})
		return
	}

	roomID, _ := data["room_id"].(string)
	if roomID == "" {
		roomID = msg.RoomID
	}

	cardsData, ok := data["cards"].([]interface{})
	if !ok {
		s.sendMessage(&core.Message{
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
		s.sendMessage(&core.Message{
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
		zap.Uint("user_id", s.client.GetUserID()),
	)

	// 发送消息通知客户端通过API调用
	s.sendMessage(&core.Message{
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
	s.broadcaster.BroadcastMessage(&core.Message{
		Type:   "player_playing",
		RoomID: roomID,
		UserID: s.client.GetUserID(),
		RawData: map[string]interface{}{
			"user_id": s.client.GetUserID(),
			"action":  "playing",
		},
	})
}

// HandlePass 处理过牌
func (s *Service) HandlePass(msg *core.Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		s.sendMessage(&core.Message{
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
		s.sendMessage(&core.Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/pass", cfg.Server.Port, roomID)
	s.sendMessage(&core.Message{
		Type:   "pass_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API调用过牌接口",
			"url":     apiURL,
			"method":  "POST",
		},
	})

	// 广播给房间内其他客户端
	s.broadcaster.BroadcastMessage(&core.Message{
		Type:   "player_passed",
		RoomID: roomID,
		UserID: s.client.GetUserID(),
		RawData: map[string]interface{}{
			"user_id": s.client.GetUserID(),
			"action":  "passed",
		},
	})
}

// HandleGetGameState 处理获取游戏状态
func (s *Service) HandleGetGameState(msg *core.Message) {
	var data map[string]interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		s.sendMessage(&core.Message{
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
		s.sendMessage(&core.Message{
			Type: "error",
			RawData: map[string]interface{}{
				"message": "配置加载失败",
			},
		})
		return
	}

	apiURL := fmt.Sprintf("http://localhost:%d/api/v1/games/rooms/%s/game-state", cfg.Server.Port, roomID)
	s.sendMessage(&core.Message{
		Type:   "get_game_state_redirect",
		RoomID: roomID,
		RawData: map[string]interface{}{
			"message": "请通过HTTP API获取游戏状态",
			"url":     apiURL,
			"method":  "GET",
		},
	})
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
