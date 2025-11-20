package core

import "encoding/json"

// Message WebSocket消息
type Message struct {
	Type    string          `json:"type"`    // 消息类型
	RoomID  string          `json:"room_id"` // 房间ID（可选）
	UserID  uint            `json:"user_id"` // 用户ID
	Data    json.RawMessage `json:"data"`    // 消息数据
	RawData interface{}     `json:"-"`       // 原始数据（用于内部处理）
}

// GetString 从 map 中安全获取字符串
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// GetUint 从 map 中安全获取 uint
func GetUint(m map[string]interface{}, key string) uint {
	if v, ok := m[key].(float64); ok {
		return uint(v)
	}
	return 0
}
