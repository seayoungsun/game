package handlers

import (
	"encoding/json"

	"github.com/kaifa/game-platform/apps/game-server/utils"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

// extractUserID 从 interface{} 中提取 userID（支持多种数字类型）
func extractUserID(v interface{}) (uint, bool) {
	switch val := v.(type) {
	case float64:
		return uint(val), true
	case int:
		return uint(val), true
	case uint:
		return val, true
	case int64:
		return uint(val), true
	default:
		return 0, false
	}
}

// extractPlayersFromRoomData 从房间数据中提取玩家列表
func extractPlayersFromRoomData(roomData map[string]interface{}, gameStateData map[string]interface{}) []uint {
	var playersToNotify []uint

	// 首先尝试从room数据中获取玩家列表
	if roomData != nil {
		if room, ok := roomData["room"].(map[string]interface{}); ok {
			if playersData, ok := room["players"]; ok {
				// 解析玩家列表（可能是JSON字符串或数组）
				var players []map[string]interface{}

				// 尝试解析为JSON字符串
				if playersStr, ok := playersData.(string); ok {
					var playersArray []map[string]interface{}
					if err := json.Unmarshal([]byte(playersStr), &playersArray); err == nil {
						players = playersArray
					}
				} else if playersArray, ok := playersData.([]interface{}); ok {
					// 已经是数组格式
					for _, p := range playersArray {
						if pMap, ok := p.(map[string]interface{}); ok {
							players = append(players, pMap)
						}
					}
				}

				// 提取所有玩家ID
				for _, player := range players {
					if userID, ok := extractUserID(player["user_id"]); ok {
						playersToNotify = append(playersToNotify, userID)
					}
				}
			}
		}
	}

	// 如果没有从room数据获取到，尝试从game_state中获取
	if len(playersToNotify) == 0 && gameStateData != nil {
		playersToNotify = extractPlayersFromGameState(gameStateData)
	}

	return playersToNotify
}

// extractPlayersFromGameState 从游戏状态中提取玩家列表
func extractPlayersFromGameState(gameStateData map[string]interface{}) []uint {
	var playersToNotify []uint

	if playersData, ok := gameStateData["players"].(map[string]interface{}); ok {
		for _, playerData := range playersData {
			if playerInfo, ok := playerData.(map[string]interface{}); ok {
				if userID, ok := extractUserID(playerInfo["user_id"]); ok {
					playersToNotify = append(playersToNotify, userID)
				}
			}
		}
	}

	return playersToNotify
}

// broadcastFilteredGameState 为每个用户过滤手牌并广播游戏状态
func broadcastFilteredGameState(roomID string, userID uint, gameStateData map[string]interface{}) {
	if playersData, ok := gameStateData["players"].(map[string]interface{}); ok {
		for playerKey, playerData := range playersData {
			if playerInfo, ok := playerData.(map[string]interface{}); ok {
				playerUserID, ok := extractUserID(playerInfo["user_id"])
				if !ok {
					continue
				}

				// 为每个用户过滤手牌
				filteredState := utils.FilterGameStateForUser(gameStateData, playerUserID)

				// 发送给该用户的客户端
				if client := hubInstance.GetUserClient(playerUserID); client != nil {
					client.SendMessage(newMessageFunc("game_state_update", roomID, playerUserID, map[string]interface{}{
						"game_state": filteredState,
					}))
				}
			}

			_ = playerKey // 避免未使用变量
		}
	}
}

// broadcastToRoomPlayers 向房间内的所有玩家广播消息
func broadcastToRoomPlayers(req *RoomNotifyRequest, msg MessageInterface) {
	if req.RoomData == nil {
		return
	}

	playersData, ok := req.RoomData["players"]
	if !ok {
		return
	}

	// 解析玩家列表（支持多种数字类型）
	playersJSON, err := json.Marshal(playersData)
	if err != nil {
		return
	}

	var players []map[string]interface{}
	if err := json.Unmarshal(playersJSON, &players); err != nil {
		return
	}

	// 向房间内的所有用户发送消息（包括没有通过WebSocket加入房间的）
	for _, player := range players {
		userID, ok := extractUserID(player["user_id"])
		if !ok {
			continue
		}

		// 如果有WebSocket连接，发送消息
		if client := hubInstance.GetUserClient(userID); client != nil {
			logger.Logger.Debug("向用户发送房间更新消息",
				zap.Uint("user_id", userID),
				zap.String("room_id", req.RoomID),
				zap.String("action", req.Action),
			)
			client.SendMessage(msg)
		}
	}
}
