package models

import "encoding/json"

// GameState 游戏状态
type GameState struct {
	RoomID        string                   `json:"room_id"`        // 房间ID
	GameType      string                   `json:"game_type"`      // 游戏类型
	Status        int                      `json:"status"`         // 游戏状态: 0等待, 1进行中, 2结算中, 3已结束
	Round         int                      `json:"round"`          // 当前回合数
	CurrentPlayer uint                     `json:"current_player"` // 当前出牌玩家ID
	LastCards     []int                    `json:"last_cards"`     // 上次出的牌
	LastPlayer    uint                     `json:"last_player"`    // 上次出牌的玩家ID
	PassCount     int                      `json:"pass_count"`     // 连续过牌次数
	Players       map[uint]*PlayerGameInfo `json:"players"`        // 玩家游戏信息
	Deck          []int                    `json:"deck,omitempty"` // 牌堆（仅用于调试）
	StartTime     int64                    `json:"start_time"`     // 游戏开始时间
}

// PlayerGameInfo 玩家游戏信息
type PlayerGameInfo struct {
	UserID     uint  `json:"user_id"`     // 用户ID
	Position   int   `json:"position"`    // 位置
	Cards      []int `json:"cards"`       // 手牌
	CardCount  int   `json:"card_count"`  // 手牌数量
	IsPassed   bool  `json:"is_passed"`   // 本回合是否已过
	IsFinished bool  `json:"is_finished"` // 是否已出完牌
	Rank       int   `json:"rank"`        // 名次（1,2,3,4）

	// 牛牛游戏专用字段
	PlayedCards []int `json:"played_cards,omitempty"` // 玩家出的牌（牛牛游戏：5张牌）
	BullType    int   `json:"bull_type,omitempty"`    // 牛牛类型：0=无牛, 1-9=有牛, 10=牛牛, 11=四花, 12=五花, 13=炸弹, 14=五小牛
	BullNum     int   `json:"bull_num,omitempty"`     // 牛数（当有牛时）
	MaxCard     int   `json:"max_card,omitempty"`     // 最大牌点数
}

// ToJSON 转换为JSON
func (gs *GameState) ToJSON() (json.RawMessage, error) {
	return json.Marshal(gs)
}

// FromJSON 从JSON解析
func (gs *GameState) FromJSON(data json.RawMessage) error {
	return json.Unmarshal(data, gs)
}

// FilterForUser 为指定用户过滤游戏状态（隐藏其他玩家手牌）
func (gs *GameState) FilterForUser(userID uint) *GameState {
	// 创建新的游戏状态副本
	filtered := &GameState{
		RoomID:        gs.RoomID,
		GameType:      gs.GameType,
		Status:        gs.Status,
		Round:         gs.Round,
		CurrentPlayer: gs.CurrentPlayer,
		LastCards:     gs.LastCards, // 已出的牌可以显示
		LastPlayer:    gs.LastPlayer,
		PassCount:     gs.PassCount,
		Players:       make(map[uint]*PlayerGameInfo),
		StartTime:     gs.StartTime,
		// Deck 不返回（调试用）
	}

	// 复制玩家信息，但隐藏其他玩家的手牌
	for uid, playerInfo := range gs.Players {
		filteredPlayer := &PlayerGameInfo{
			UserID:     playerInfo.UserID,
			Position:   playerInfo.Position,
			CardCount:  playerInfo.CardCount,
			IsPassed:   playerInfo.IsPassed,
			IsFinished: playerInfo.IsFinished,
			Rank:       playerInfo.Rank,
		}

		// 只返回当前用户的完整手牌，其他玩家的手牌隐藏
		if uid == userID {
			// 自己的手牌完整返回
			filteredPlayer.Cards = make([]int, len(playerInfo.Cards))
			copy(filteredPlayer.Cards, playerInfo.Cards)
		} else if userID == 0 {
			// userID为0表示隐藏所有手牌（未登录用户）
			filteredPlayer.Cards = []int{}
		} else {
			// 其他玩家的手牌隐藏，返回空数组
			filteredPlayer.Cards = []int{}
		}

		filtered.Players[uid] = filteredPlayer
	}

	return filtered
}

// Card 扑克牌定义
// 牌值：3=3, 4=4, ..., K=13, A=14, 2=15, 小王=16, 大王=17
// 花色：红桃=0, 方块=1, 黑桃=2, 梅花=3
// 完整牌值：花色*100 + 点数
// 例如：红桃3 = 0*100 + 3 = 3，方块K = 1*100 + 13 = 113

const (
	CardValue3  = 3
	CardValue4  = 4
	CardValue5  = 5
	CardValue6  = 6
	CardValue7  = 7
	CardValue8  = 8
	CardValue9  = 9
	CardValue10 = 10
	CardValueJ  = 11
	CardValueQ  = 12
	CardValueK  = 13
	CardValueA  = 14
	CardValue2  = 15
	CardJoker   = 16 // 小王
	CardKing    = 17 // 大王

	SuitHeart   = 0 // 红桃
	SuitDiamond = 1 // 方块
	SuitSpade   = 2 // 黑桃
	SuitClub    = 3 // 梅花
)

// GetCardValue 获取牌的点数
func GetCardValue(card int) int {
	if card == CardJoker || card == CardKing {
		return card
	}
	return card % 100
}

// GetCardSuit 获取牌的花色
func GetCardSuit(card int) int {
	if card == CardJoker || card == CardKing {
		return -1 // 大小王没有花色
	}
	return card / 100
}
