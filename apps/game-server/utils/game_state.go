package utils

// FilterGameStateForUser 为指定用户过滤游戏状态（隐藏其他玩家手牌）
func FilterGameStateForUser(gameStateData map[string]interface{}, userID uint) map[string]interface{} {
	// 创建新的游戏状态副本
	filtered := make(map[string]interface{})

	// 复制所有字段
	for key, value := range gameStateData {
		if key == "players" {
			// 处理玩家信息
			if players, ok := value.(map[string]interface{}); ok {
				filteredPlayers := make(map[string]interface{})
				for playerKey, playerData := range players {
					if playerInfo, ok := playerData.(map[string]interface{}); ok {
						filteredPlayer := make(map[string]interface{})

						// 复制所有玩家信息
						for k, v := range playerInfo {
							filteredPlayer[k] = v
						}

						// 获取玩家user_id
						var playerUserID uint
						switch v := playerInfo["user_id"].(type) {
						case float64:
							playerUserID = uint(v)
						case int:
							playerUserID = uint(v)
						case uint:
							playerUserID = v
						case int64:
							playerUserID = uint(v)
						}

						// 只返回当前用户的完整手牌，其他玩家的手牌隐藏
						if playerUserID == userID {
							// 自己的手牌完整返回
							// cards 字段保持不变
						} else {
							// 其他玩家的手牌隐藏，返回空数组
							filteredPlayer["cards"] = []interface{}{}
						}

						filteredPlayers[playerKey] = filteredPlayer
					}
				}
				filtered[key] = filteredPlayers
			} else {
				filtered[key] = value
			}
		} else {
			filtered[key] = value
		}
	}

	return filtered
}
