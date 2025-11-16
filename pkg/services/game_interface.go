package services

// GameEngine 游戏引擎接口
type GameEngine interface {
	// DealCards 发牌
	// playerCount: 玩家数量
	// 返回: map[玩家索引]手牌列表
	DealCards(playerCount int) (map[uint][]int, error)

	// ValidateCards 验证出牌是否合法
	// cards: 要出的牌
	// lastCards: 上家出的牌（为空表示首出）
	// 返回: (是否合法, 错误信息)
	ValidateCards(cards []int, lastCards []int) (bool, string)

	// GetGameName 获取游戏名称
	GetGameName() string

	// GetGameType 获取游戏类型
	GetGameType() string
}
