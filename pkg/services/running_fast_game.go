package services

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/kaifa/game-platform/pkg/models"
)

// RunningFastGame 跑得快游戏引擎
type RunningFastGame struct{}

// NewRunningFastGame 创建跑得快游戏引擎
func NewRunningFastGame() *RunningFastGame {
	return &RunningFastGame{}
}

// GetGameName 获取游戏名称
func (g *RunningFastGame) GetGameName() string {
	return "跑得快"
}

// GetGameType 获取游戏类型
func (g *RunningFastGame) GetGameType() string {
	return "running"
}

// DealCards 发牌
func (g *RunningFastGame) DealCards(playerCount int) (map[uint][]int, error) {
	if playerCount < 2 || playerCount > 4 {
		return nil, errors.New("玩家数量必须在2-4之间")
	}

	// 生成一副牌（不含大小王）
	deck := make([]int, 0, 52)
	for suit := 0; suit < 4; suit++ {
		for value := 3; value <= 15; value++ { // 3到2
			card := suit*100 + value
			deck = append(deck, card)
		}
	}

	// 洗牌
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	// 发牌（每人17张）
	cardsPerPlayer := 17

	hands := make(map[uint][]int)
	currentCard := 0

	// 为每个玩家发牌
	for i := 0; i < playerCount; i++ {
		playerID := uint(i + 1)
		hands[playerID] = make([]int, 0, cardsPerPlayer)

		for j := 0; j < cardsPerPlayer && currentCard < len(deck); j++ {
			hands[playerID] = append(hands[playerID], deck[currentCard])
			currentCard++
		}

		// 排序手牌（方便查看）
		sort.Ints(hands[playerID])
	}

	return hands, nil
}

// ValidateCards 验证出牌是否合法
func (g *RunningFastGame) ValidateCards(cards []int, lastCards []int) (bool, string) {
	if len(cards) == 0 {
		return false, "不能出空牌"
	}

	// 检查牌是否重复
	cardMap := make(map[int]int)
	for _, card := range cards {
		cardMap[card]++
		if cardMap[card] > 1 {
			return false, "不能出重复的牌"
		}
	}

	// 如果这是第一次出牌或者是新一轮（所有人过牌）
	if len(lastCards) == 0 {
		return g.validateFirstPlay(cards)
	}

	// 验证能否压过上家
	return g.canBeatLastCards(cards, lastCards)
}

// validateFirstPlay 验证首次出牌
func (g *RunningFastGame) validateFirstPlay(cards []int) (bool, string) {
	cardCount := len(cards)

	switch cardCount {
	case 1:
		// 单张
		return true, "单张"
	case 2:
		// 对子或王炸
		return g.validatePairOrBomb(cards)
	case 3:
		// 三张（三带零）
		return g.validateThree(cards, 0)
	case 4:
		// 炸弹、三带一、四带二
		return g.validateFour(cards)
	case 5:
		// 顺子或三带二
		return g.validateFive(cards)
	default:
		// 顺子、连对、飞机等
		return g.validateMultiCards(cards)
	}
}

// validatePairOrBomb 验证对子或王炸
func (g *RunningFastGame) validatePairOrBomb(cards []int) (bool, string) {
	val1 := models.GetCardValue(cards[0])
	val2 := models.GetCardValue(cards[1])

	// 王炸
	if (val1 == models.CardJoker && val2 == models.CardKing) ||
		(val1 == models.CardKing && val2 == models.CardJoker) {
		return true, "王炸"
	}

	// 对子
	if val1 == val2 && val1 != models.CardJoker && val1 != models.CardKing {
		return true, "对子"
	}

	return false, "不是有效的对子或王炸"
}

// validateThree 验证三张（三带零）
func (g *RunningFastGame) validateThree(cards []int, withCount int) (bool, string) {
	if len(cards) != 3+withCount {
		return false, "牌数不对"
	}

	values := make(map[int]int)
	for _, card := range cards {
		val := models.GetCardValue(card)
		if val != models.CardJoker && val != models.CardKing {
			values[val]++
		}
	}

	// 检查是否有三张相同的
	for val, count := range values {
		if count == 3 {
			return true, fmt.Sprintf("三张%d", val)
		}
	}

	return false, "不是有效的三张"
}

// validateFour 验证四张（炸弹、三带一、四带二）
func (g *RunningFastGame) validateFour(cards []int) (bool, string) {
	if len(cards) != 4 {
		return false, "牌数不对"
	}

	values := make(map[int]int)
	for _, card := range cards {
		val := models.GetCardValue(card)
		if val != models.CardJoker && val != models.CardKing {
			values[val]++
		}
	}

	// 炸弹（四张相同）
	for val, count := range values {
		if count == 4 {
			return true, fmt.Sprintf("炸弹%d", val)
		}
	}

	// 三带一
	for _, count := range values {
		if count == 3 && len(values) == 2 {
			return true, "三带一"
		}
	}

	return false, "不是有效的四张牌型"
}

// validateFive 验证五张（顺子或三带二）
func (g *RunningFastGame) validateFive(cards []int) (bool, string) {
	if len(cards) != 5 {
		return false, "牌数不对"
	}

	values := make(map[int]int)
	for _, card := range cards {
		val := models.GetCardValue(card)
		if val != models.CardJoker && val != models.CardKing && val != models.CardValue2 {
			values[val]++
		}
	}

	// 三带二
	if len(values) == 2 {
		for _, count := range values {
			if count == 3 {
				return true, "三带二"
			}
		}
	}

	// 顺子（五张连续）
	if g.isStraight(values, 5) {
		return true, "顺子"
	}

	return false, "不是有效的五张牌型"
}

// validateMultiCards 验证多张牌型（顺子、连对、飞机等）
func (g *RunningFastGame) validateMultiCards(cards []int) (bool, string) {
	values := make(map[int]int)
	for _, card := range cards {
		val := models.GetCardValue(card)
		// 2、大小王不能参与顺子
		if val != models.CardJoker && val != models.CardKing && val != models.CardValue2 {
			values[val]++
		}
	}

	cardCount := len(cards)

	// 单顺（5张以上连续单张）
	if cardCount >= 5 && len(values) == cardCount {
		if g.isStraight(values, cardCount) {
			return true, fmt.Sprintf("%d张顺子", cardCount)
		}
	}

	// 连对（3对以上连续对子）
	if cardCount >= 6 && cardCount%2 == 0 {
		pairCount := cardCount / 2
		if g.isConsecutivePairs(values, pairCount) {
			return true, fmt.Sprintf("%d对连对", pairCount)
		}
	}

	// 飞机（2组以上连续三张）
	if cardCount >= 6 && cardCount%3 == 0 {
		threeCount := cardCount / 3
		if g.isConsecutiveThrees(values, threeCount) {
			return true, fmt.Sprintf("%d组飞机", threeCount)
		}
	}

	return false, "不是有效的牌型"
}

// canBeatLastCards 判断能否压过上家
func (g *RunningFastGame) canBeatLastCards(cards []int, lastCards []int) (bool, string) {
	// 王炸最大
	if g.isKingBomb(cards) {
		return true, "王炸"
	}

	// 王炸不能被打（除了新的王炸）
	if g.isKingBomb(lastCards) {
		return false, "上家出的是王炸，不能压"
	}

	// 普通炸弹压普通牌型
	if g.isBomb(cards) && !g.isBomb(lastCards) {
		return true, "炸弹"
	}

	// 牌型必须匹配
	lastType := g.getCardType(lastCards)
	currentType := g.getCardType(cards)

	if lastType != currentType {
		return false, fmt.Sprintf("牌型不匹配，需要出%s", lastType)
	}

	// 同类型比较大小
	return g.compareSameType(cards, lastCards)
}

// compareSameType 比较同类型牌的大小
func (g *RunningFastGame) compareSameType(cards []int, lastCards []int) (bool, string) {
	// 获取主牌的值（用于比较）
	cardValue := g.getMainCardValue(cards)
	lastValue := g.getMainCardValue(lastCards)

	if cardValue > lastValue {
		return true, "可以压过"
	}

	return false, "牌值不够大"
}

// 辅助函数
func (g *RunningFastGame) isKingBomb(cards []int) bool {
	if len(cards) != 2 {
		return false
	}
	val1 := models.GetCardValue(cards[0])
	val2 := models.GetCardValue(cards[1])
	return (val1 == models.CardJoker && val2 == models.CardKing) ||
		(val1 == models.CardKing && val2 == models.CardJoker)
}

func (g *RunningFastGame) isBomb(cards []int) bool {
	if len(cards) != 4 {
		return false
	}
	values := make(map[int]int)
	for _, card := range cards {
		val := models.GetCardValue(card)
		if val != models.CardJoker && val != models.CardKing {
			values[val]++
		}
	}
	for _, count := range values {
		if count == 4 {
			return true
		}
	}
	return false
}

func (g *RunningFastGame) isStraight(values map[int]int, length int) bool {
	if len(values) != length {
		return false
	}
	vals := make([]int, 0, length)
	for val := range values {
		vals = append(vals, val)
	}
	sort.Ints(vals)

	// 检查是否连续
	for i := 1; i < len(vals); i++ {
		if vals[i] != vals[i-1]+1 {
			return false
		}
	}

	return true
}

func (g *RunningFastGame) isConsecutivePairs(values map[int]int, pairCount int) bool {
	if len(values) != pairCount {
		return false
	}

	// 检查每个值都是2张
	for _, count := range values {
		if count != 2 {
			return false
		}
	}

	vals := make([]int, 0, pairCount)
	for val := range values {
		vals = append(vals, val)
	}
	sort.Ints(vals)

	// 检查是否连续
	for i := 1; i < len(vals); i++ {
		if vals[i] != vals[i-1]+1 {
			return false
		}
	}

	return true
}

func (g *RunningFastGame) isConsecutiveThrees(values map[int]int, threeCount int) bool {
	if len(values) != threeCount {
		return false
	}

	// 检查每个值都是3张
	for _, count := range values {
		if count != 3 {
			return false
		}
	}

	vals := make([]int, 0, threeCount)
	for val := range values {
		vals = append(vals, val)
	}
	sort.Ints(vals)

	// 检查是否连续
	for i := 1; i < len(vals); i++ {
		if vals[i] != vals[i-1]+1 {
			return false
		}
	}

	return true
}

func (g *RunningFastGame) getCardType(cards []int) string {
	switch len(cards) {
	case 1:
		return "单张"
	case 2:
		if g.isKingBomb(cards) {
			return "王炸"
		}
		return "对子"
	case 3:
		return "三张"
	case 4:
		if g.isBomb(cards) {
			return "炸弹"
		}
		return "三带一"
	case 5:
		values := make(map[int]int)
		for _, card := range cards {
			val := models.GetCardValue(card)
			if val != models.CardJoker && val != models.CardKing && val != models.CardValue2 {
				values[val]++
			}
		}
		if len(values) == 2 {
			return "三带二"
		}
		return "顺子"
	default:
		// 顺子、连对、飞机等
		values := make(map[int]int)
		for _, card := range cards {
			val := models.GetCardValue(card)
			if val != models.CardJoker && val != models.CardKing && val != models.CardValue2 {
				values[val]++
			}
		}
		if len(values) == len(cards) {
			return "顺子"
		}
		if len(values) == len(cards)/2 {
			return "连对"
		}
		if len(values) == len(cards)/3 {
			return "飞机"
		}
		return "未知"
	}
}

func (g *RunningFastGame) getMainCardValue(cards []int) int {
	if len(cards) == 0 {
		return 0
	}

	// 对于单张、对子等，返回最大牌值
	values := make([]int, 0, len(cards))
	for _, card := range cards {
		val := models.GetCardValue(card)
		values = append(values, val)
	}
	sort.Ints(values)

	// 返回最小值的牌（因为要比较是否大于上家）
	// 对于单张、对子、三张等，比较最小牌即可
	return values[0]
}
