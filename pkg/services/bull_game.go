package services

import (
	"errors"
	"math/rand"
	"sort"
	"time"
)

// BullGame 牛牛游戏引擎
type BullGame struct{}

// NewBullGame 创建牛牛游戏引擎
func NewBullGame() *BullGame {
	return &BullGame{}
}

// GetGameName 获取游戏名称
func (g *BullGame) GetGameName() string {
	return "牛牛"
}

// GetGameType 获取游戏类型
func (g *BullGame) GetGameType() string {
	return "bull"
}

// DealCards 发牌（牛牛：每人5张牌）
func (g *BullGame) DealCards(playerCount int) (map[uint][]int, error) {
	if playerCount < 2 || playerCount > 5 {
		return nil, errors.New("玩家数量必须在2-5之间")
	}

	// 生成一副牌（52张，不含大小王）
	deck := make([]int, 0, 52)
	for suit := 0; suit < 4; suit++ {
		for rank := 1; rank <= 13; rank++ { // A(1), 2, 3, ..., J(11), Q(12), K(13)
			card := suit*100 + rank
			deck = append(deck, card)
		}
	}

	// 洗牌
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	// 发牌（每人5张）
	hands := make(map[uint][]int)
	for i := 0; i < playerCount; i++ {
		playerID := uint(i + 1)
		hands[playerID] = make([]int, 0, 5)

		// 每人发5张
		for j := 0; j < 5; j++ {
			if len(deck) == 0 {
				return nil, errors.New("牌不够发")
			}
			hands[playerID] = append(hands[playerID], deck[0])
			deck = deck[1:]
		}

		// 排序手牌（方便查看）
		sort.Ints(hands[playerID])
	}

	return hands, nil
}

// ValidateCards 验证出牌（牛牛游戏不需要出牌，这里是占位）
func (g *BullGame) ValidateCards(cards []int, lastCards []int) (bool, string) {
	// 牛牛游戏不需要出牌验证，直接返回true
	return true, ""
}

// CalculateBull 计算牛牛牌型
// 返回: (牛牛类型, 牛数, 最大牌的点数)
// 牛牛类型: 0=无牛, 1-9=有牛(1牛到9牛), 10=牛牛, 11=四花, 12=五花, 13=炸弹, 14=五小牛
func (g *BullGame) CalculateBull(cards []int) (bullType int, bullNum int, maxCard int) {
	if len(cards) != 5 {
		return 0, 0, 0
	}

	// 转换牌为点数（A=1, 2-10=2-10, J/Q/K=10）
	points := make([]int, 5)
	values := make([]int, 5)
	for i, card := range cards {
		rank := card % 100
		values[i] = rank

		if rank >= 11 { // J, Q, K 都是10点
			points[i] = 10
		} else if rank == 1 { // A 是1点
			points[i] = 1
		} else {
			points[i] = rank
		}
	}

	// 计算所有可能的3张牌组合
	combinations := [][]int{
		{0, 1, 2}, {0, 1, 3}, {0, 1, 4},
		{0, 2, 3}, {0, 2, 4}, {0, 3, 4},
		{1, 2, 3}, {1, 2, 4}, {1, 3, 4},
		{2, 3, 4},
	}

	// 找出3张牌的和是10的倍数的组合
	var validCombos [][]int
	for _, combo := range combinations {
		sum := points[combo[0]] + points[combo[1]] + points[combo[2]]
		if sum%10 == 0 {
			validCombos = append(validCombos, combo)
		}
	}

	if len(validCombos) == 0 {
		// 无牛，返回最大牌
		maxCard = g.getMaxCard(cards)
		return 0, 0, maxCard
	}

	// 取第一个有效组合
	combo := validCombos[0]

	// 找出剩余2张牌的索引
	remaining := make([]int, 0, 2)
	for i := 0; i < 5; i++ {
		isInCombo := false
		for _, idx := range combo {
			if i == idx {
				isInCombo = true
				break
			}
		}
		if !isInCombo {
			remaining = append(remaining, i)
		}
	}

	// 计算剩余2张牌的和
	remainingSum := points[remaining[0]] + points[remaining[1]]
	bullNum = remainingSum % 10

	// 找出最大牌
	maxCard = g.getMaxCard(cards)

	// 特殊牌型判断
	if bullNum == 0 {
		// 检查是否是炸弹（4张同点数）
		if g.isBomb(cards) {
			return 13, 0, maxCard // 炸弹
		}
		return 10, 0, maxCard // 牛牛
	}

	// 检查特殊牌型
	if g.isFiveFlower(cards) {
		return 12, 0, maxCard // 五花
	}
	if g.isFourFlower(cards) {
		return 11, 0, maxCard // 四花
	}
	if g.isFiveSmall(cards) {
		return 14, 0, maxCard // 五小牛
	}

	return bullNum, bullNum, maxCard
}

// getMaxCard 获取最大牌的点数
func (g *BullGame) getMaxCard(cards []int) int {
	max := 0
	for _, card := range cards {
		rank := card % 100
		if rank == 1 { // A
			rank = 14 // A最大
		}
		if rank > max {
			max = rank
		}
	}
	return max
}

// isBomb 判断是否是炸弹（4张同点数）
func (g *BullGame) isBomb(cards []int) bool {
	rankCount := make(map[int]int)
	for _, card := range cards {
		rank := card % 100
		rankCount[rank]++
		if rankCount[rank] >= 4 {
			return true
		}
	}
	return false
}

// isFiveFlower 判断是否是五花（5张都是J/Q/K）
func (g *BullGame) isFiveFlower(cards []int) bool {
	for _, card := range cards {
		rank := card % 100
		if rank < 11 { // 不是J/Q/K
			return false
		}
	}
	return true
}

// isFourFlower 判断是否是四花（4张是J/Q/K）
func (g *BullGame) isFourFlower(cards []int) bool {
	count := 0
	for _, card := range cards {
		rank := card % 100
		if rank >= 11 { // J/Q/K
			count++
		}
	}
	return count == 4
}

// isFiveSmall 判断是否是五小牛（5张牌都小于5且和小于等于10）
func (g *BullGame) isFiveSmall(cards []int) bool {
	sum := 0
	for _, card := range cards {
		rank := card % 100
		if rank >= 5 { // 大于等于5
			return false
		}
		if rank == 1 { // A
			sum += 1
		} else {
			sum += rank
		}
	}
	return sum <= 10
}

// CompareBull 比较两个牛牛牌型
// 返回: >0表示card1大于card2, <0表示card1小于card2, 0表示相等
func (g *BullGame) CompareBull(cards1, cards2 []int) int {
	bullType1, bullNum1, maxCard1 := g.CalculateBull(cards1)
	bullType2, bullNum2, maxCard2 := g.CalculateBull(cards2)

	// 先比较牛牛类型
	if bullType1 != bullType2 {
		return bullType1 - bullType2
	}

	// 如果类型相同，比较牛数（对于牛牛类型，比较最大牌）
	if bullType1 == 10 || bullType1 == 0 {
		// 牛牛或无牛，比较最大牌
		return maxCard1 - maxCard2
	}

	// 有牛（1-9），比较牛数
	if bullNum1 != bullNum2 {
		return bullNum1 - bullNum2
	}

	// 牛数相同，比较最大牌
	return maxCard1 - maxCard2
}
