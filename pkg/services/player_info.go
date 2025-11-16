package services

// PlayerInfo 描述房间内玩家的基本状态，供游戏流程与房间管理共用。
type PlayerInfo struct {
	UserID   uint   `json:"user_id"`
	UID      int64  `json:"uid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Position int    `json:"position"`
	Ready    bool   `json:"ready"`
}
