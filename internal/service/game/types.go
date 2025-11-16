package game

// GameSettlement 游戏结算结果
type GameSettlement struct {
	RoomID   string                     `json:"room_id"`
	RecordID uint                       `json:"record_id"`
	Players  map[uint]*PlayerSettlement `json:"players"` // 玩家结算信息
}

// PlayerSettlement 玩家结算信息
type PlayerSettlement struct {
	UserID       uint    `json:"user_id"`
	Rank         int     `json:"rank"`          // 名次（1,2,3,4）
	Balance      float64 `json:"balance"`       // 本局余额变化
	FinalBalance float64 `json:"final_balance"` // 结算后余额
}
