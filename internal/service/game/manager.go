package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/kaifa/game-platform/internal/lock"
	gamerecordrepo "github.com/kaifa/game-platform/internal/repository/gamerecord"
	roomrepo "github.com/kaifa/game-platform/internal/repository/room"
	userrepo "github.com/kaifa/game-platform/internal/repository/user"
	leaderboardsvc "github.com/kaifa/game-platform/internal/service/leaderboard"
	"github.com/kaifa/game-platform/internal/storage"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/kaifa/game-platform/pkg/services"
)

// Manager 游戏管理器（重构版本 - 使用依赖注入）
// 职责：管理游戏流程逻辑，不直接操作数据库和缓存
type Manager struct {
	// Repository 和 Service 依赖
	stateStorage   storage.GameStateStorage  // 游戏状态存储
	roomRepo       roomrepo.Repository       // 房间数据访问
	userRepo       userrepo.Repository       // 用户数据访问
	gameRecordRepo gamerecordrepo.Repository // 游戏记录数据访问
	leaderboardSvc leaderboardsvc.Service    // 排行榜服务

	// 并发控制组件
	distLock  lock.Lock   // ✅ 分布式锁（用于关键游戏操作）
	localLock lock.RWLock // ✅ 本地读写锁（用于快速读取）

	// 游戏引擎
	engines map[string]services.GameEngine // 游戏引擎映射
}

// NewManager 创建游戏管理器实例
func NewManager(
	stateStorage storage.GameStateStorage,
	roomRepo roomrepo.Repository,
	userRepo userrepo.Repository,
	gameRecordRepo gamerecordrepo.Repository,
	leaderboardSvc leaderboardsvc.Service,
	distLock lock.Lock, // ✅ 注入分布式锁
	localLock lock.RWLock, // ✅ 注入本地锁
) *Manager {
	engines := make(map[string]services.GameEngine)
	// 注册游戏引擎
	engines["running"] = services.NewRunningFastGame()
	engines["bull"] = services.NewBullGame()

	return &Manager{
		stateStorage:   stateStorage,
		roomRepo:       roomRepo,
		userRepo:       userRepo,
		gameRecordRepo: gameRecordRepo,
		leaderboardSvc: leaderboardSvc,
		distLock:       distLock,
		localLock:      localLock,
		engines:        engines,
	}
}

// StartGame 开始游戏（重构版本）
func (m *Manager) StartGame(ctx context.Context, roomID string) (*models.GameState, error) {
	// ✅ 通过 Repository 获取房间信息
	room, err := m.roomRepo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, errors.New("房间不存在")
	}

	// 检查房间状态
	if room.Status != 1 {
		return nil, errors.New("房间状态不正确")
	}

	// 解析玩家列表
	var players []services.PlayerInfo
	if err := json.Unmarshal(room.Players, &players); err != nil {
		return nil, fmt.Errorf("解析玩家列表失败: %w", err)
	}

	if len(players) < 2 {
		return nil, errors.New("至少需要2人才能开始游戏")
	}

	// 检查所有人是否都准备好了
	for _, p := range players {
		if !p.Ready {
			return nil, errors.New("还有玩家未准备")
		}
	}

	// 获取游戏引擎
	engine, err := m.getEngine(room.GameType)
	if err != nil {
		return nil, err
	}

	// ✅ 业务逻辑：创建游戏状态
	var gameState *models.GameState
	switch room.GameType {
	case "running":
		gameState, err = m.startRunningFastGame(roomID, players)
	case "bull":
		gameState, err = m.startBullGame(roomID, players, engine.(*services.BullGame))
	default:
		return nil, fmt.Errorf("未知的游戏类型: %s", room.GameType)
	}

	if err != nil {
		return nil, err
	}

	// ✅ 通过 Storage 保存游戏状态
	if err := m.stateStorage.Save(ctx, gameState, 2*time.Hour); err != nil {
		return nil, fmt.Errorf("保存游戏状态失败: %w", err)
	}

	// ✅ 通过 Repository 更新房间状态
	room.Status = 2 // 游戏中
	if err := m.roomRepo.Update(ctx, room); err != nil {
		return nil, fmt.Errorf("更新房间状态失败: %w", err)
	}

	return gameState, nil
}

// GetGameState 获取游戏状态（重构版本）
func (m *Manager) GetGameState(ctx context.Context, roomID string) (*models.GameState, error) {
	// ✅ 通过 Storage 获取游戏状态
	return m.stateStorage.Get(ctx, roomID)
}

// GetGameStateForUser 获取游戏状态（为指定用户过滤手牌）
func (m *Manager) GetGameStateForUser(ctx context.Context, roomID string, userID uint) (*models.GameState, error) {
	gameState, err := m.stateStorage.Get(ctx, roomID)
	if err != nil {
		return nil, err
	}
	// 过滤手牌，只返回该用户的手牌
	return gameState.FilterForUser(userID), nil
}

// PlayCards 出牌（重构版本）
func (m *Manager) PlayCards(ctx context.Context, roomID string, userID uint, cards []int) (*models.GameState, error) {
	// ✅ 使用分布式锁保护出牌操作（防止并发出牌导致状态错乱）
	lockKey := fmt.Sprintf("game:%s:play", roomID)

	var finalState *models.GameState
	var finalErr error

	err := m.distLock.WithLock(ctx, lockKey, 5*time.Second, func() error {
		// ✅ 在锁保护下获取游戏状态
		gameState, err := m.stateStorage.Get(ctx, roomID)
		if err != nil {
			finalErr = err
			return finalErr
		}

		// 检查是否轮到自己
		if gameState.CurrentPlayer != userID {
			finalErr = errors.New("还没轮到你出牌")
			return finalErr
		}

		// 检查玩家是否已经完成
		playerInfo, ok := gameState.Players[userID]
		if !ok {
			finalErr = errors.New("玩家不在游戏中")
			return finalErr
		}

		if playerInfo.IsFinished {
			finalErr = errors.New("你已经出完牌了")
			return finalErr
		}

		// 验证出的牌是否在手牌中
		if !m.hasCards(playerInfo.Cards, cards) {
			finalErr = errors.New("你手中没有这些牌")
			return finalErr
		}

		// 验证出牌是否合法
		var lastCardsForValidation []int
		if gameState.PassCount > 0 {
			// 有人过牌，可以自由出牌
			lastCardsForValidation = nil
		} else {
			// 没人过牌，需要压过上家
			lastCardsForValidation = gameState.LastCards
		}

		// 获取游戏引擎
		engine, err := m.getEngine(gameState.GameType)
		if err != nil {
			finalErr = err
			return finalErr
		}

		if valid, msg := engine.ValidateCards(cards, lastCardsForValidation); !valid {
			finalErr = errors.New(msg)
			return finalErr
		}

		// 移除手牌
		playerInfo.Cards = m.removeCards(playerInfo.Cards, cards)
		playerInfo.CardCount = len(playerInfo.Cards)
		playerInfo.IsPassed = false

		// 检查是否出完牌
		if len(playerInfo.Cards) == 0 {
			playerInfo.IsFinished = true
			// 计算名次
			rank := m.calculateRank(gameState)
			playerInfo.Rank = rank
		}

		// 更新游戏状态
		gameState.LastCards = cards
		gameState.LastPlayer = userID
		gameState.PassCount = 0
		gameState.Round++

		// 设置下一个出牌玩家
		gameState.CurrentPlayer = m.getNextPlayer(gameState, userID)

		// ✅ 通过 Storage 保存游戏状态
		if err := m.stateStorage.Save(ctx, gameState, 2*time.Hour); err != nil {
			finalErr = fmt.Errorf("保存游戏状态失败: %w", err)
			return finalErr
		}

		// 检查游戏是否结束（只剩一人未完成）
		isEnded, endedGameState := m.checkGameEnd(ctx, roomID, gameState)
		if isEnded {
			// 游戏结束，进行结算
			_, err := m.SettleGame(ctx, roomID, endedGameState)
			if err != nil {
				// 结算失败，记录日志但返回游戏状态
				finalState = endedGameState
				return nil
			}
			finalState = endedGameState
			return nil
		}

		finalState = gameState
		return nil
	})

	if err != nil {
		return nil, finalErr
	}

	return finalState, nil
}

// PlayBullGame 牛牛游戏出牌（重构版本）
func (m *Manager) PlayBullGame(ctx context.Context, roomID string, userID uint, selectedCards []int) (*models.GameState, error) {
	// ✅ 使用分布式锁保护牛牛出牌操作
	lockKey := fmt.Sprintf("game:%s:play", roomID)

	var finalState *models.GameState
	var finalErr error

	err := m.distLock.WithLock(ctx, lockKey, 5*time.Second, func() error {
		// ✅ 在锁保护下获取游戏状态
		gameState, err := m.stateStorage.Get(ctx, roomID)
		if err != nil {
			finalErr = err
			return finalErr
		}

		// 检查游戏类型
		if gameState.GameType != "bull" {
			finalErr = fmt.Errorf("当前房间不是牛牛游戏")
			return finalErr
		}

		// 检查是否轮到自己
		if gameState.CurrentPlayer != userID {
			finalErr = fmt.Errorf("还没轮到你")
			return finalErr
		}

		// 检查玩家信息
		playerInfo, ok := gameState.Players[userID]
		if !ok {
			finalErr = fmt.Errorf("玩家不在游戏中")
			return finalErr
		}

		if playerInfo.IsFinished {
			finalErr = fmt.Errorf("你已经完成")
			return finalErr
		}

		// 验证选择的牌（必须是5张）
		if len(selectedCards) != 5 {
			finalErr = fmt.Errorf("必须选择5张牌")
			return finalErr
		}

		// 验证牌是否在手牌中
		if !m.hasCards(playerInfo.Cards, selectedCards) {
			finalErr = fmt.Errorf("你手中没有这些牌")
			return finalErr
		}

		// 获取牛牛游戏引擎
		engine, err := m.getEngine("bull")
		if err != nil {
			finalErr = err
			return finalErr
		}
		bullGame := engine.(*services.BullGame)

		// 计算牛牛牌型
		bullType, bullNum, maxCard := bullGame.CalculateBull(selectedCards)

		// 存储玩家出的牌和牛牛结果
		playerInfo.PlayedCards = selectedCards
		playerInfo.BullType = bullType
		playerInfo.BullNum = bullNum
		playerInfo.MaxCard = maxCard

		// 标记玩家已完成
		playerInfo.IsFinished = true
		playerInfo.CardCount = 0
		playerInfo.Cards = nil

		// 更新游戏状态
		gameState.Round++
		gameState.CurrentPlayer = m.getNextPlayer(gameState, userID)

		// ✅ 通过 Storage 保存游戏状态
		if err := m.stateStorage.Save(ctx, gameState, 2*time.Hour); err != nil {
			finalErr = fmt.Errorf("保存游戏状态失败: %w", err)
			return finalErr
		}

		// 检查游戏是否结束（所有人都出完牌）
		isEnded, endedGameState := m.checkGameEnd(ctx, roomID, gameState)
		if isEnded {
			// 游戏结束，进行牛牛结算
			settlement, err := m.settleBullGame(ctx, roomID, endedGameState, bullGame)
			if err != nil {
				finalState = endedGameState
				return nil
			}
			_ = settlement
			finalState = endedGameState
			return nil
		}

		finalState = gameState
		return nil
	})

	if err != nil {
		return nil, finalErr
	}

	return finalState, nil
}

// Pass 过牌（重构版本）
func (m *Manager) Pass(ctx context.Context, roomID string, userID uint) (*models.GameState, error) {
	// ✅ 通过 Storage 获取游戏状态
	gameState, err := m.stateStorage.Get(ctx, roomID)
	if err != nil {
		return nil, err
	}

	// 检查是否轮到自己
	if gameState.CurrentPlayer != userID {
		return nil, errors.New("还没轮到你出牌")
	}

	// 检查玩家信息
	playerInfo, ok := gameState.Players[userID]
	if !ok {
		return nil, errors.New("玩家不在游戏中")
	}

	if playerInfo.IsFinished {
		return nil, errors.New("你已经出完牌了")
	}

	// 检查是否可以过（必须有人出过牌）
	if len(gameState.LastCards) == 0 {
		return nil, errors.New("第一手牌不能过")
	}

	// 标记已过
	playerInfo.IsPassed = true
	gameState.PassCount++

	// 设置下一个出牌玩家
	gameState.CurrentPlayer = m.getNextPlayer(gameState, userID)

	// 检查是否所有人都过了（新一轮）
	if gameState.PassCount >= m.getActivePlayerCount(gameState) {
		gameState.LastCards = nil
		gameState.LastPlayer = 0
		gameState.PassCount = 0
	}

	// ✅ 通过 Storage 保存游戏状态
	if err := m.stateStorage.Save(ctx, gameState, 2*time.Hour); err != nil {
		return nil, fmt.Errorf("保存游戏状态失败: %w", err)
	}

	return gameState, nil
}

// CheckGameEnd 检查游戏是否结束（重构版本）
func (m *Manager) CheckGameEnd(ctx context.Context, roomID string) (bool, *models.GameState) {
	gameState, err := m.stateStorage.Get(ctx, roomID)
	if err != nil {
		return false, nil
	}

	return m.checkGameEnd(ctx, roomID, gameState)
}

// SettleGame 结算游戏（重构版本）
func (m *Manager) SettleGame(ctx context.Context, roomID string, gameState *models.GameState) (*GameSettlement, error) {
	// ✅ 通过 Repository 获取房间信息
	room, err := m.gameRecordRepo.GetRoomByRoomID(ctx, roomID)
	if err != nil {
		return nil, errors.New("房间不存在")
	}

	// ✅ 业务逻辑：计算结算结果
	settlement := m.calculateSettlement(gameState, room.BaseBet)

	// 准备批量更新余额的数据
	balanceUpdates := make(map[uint]float64)
	for userID, playerSettlement := range settlement.Players {
		// ✅ 通过 Repository 获取当前余额
		user, err := m.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("用户不存在: %d", userID)
		}

		// 计算新余额
		newBalance := user.Balance + playerSettlement.Balance
		if newBalance < 0 {
			newBalance = 0
		}

		balanceUpdates[userID] = newBalance
		playerSettlement.FinalBalance = newBalance
	}

	// ✅ 通过 Repository 批量更新余额（使用事务）
	if err := m.userRepo.BatchUpdateBalances(ctx, balanceUpdates); err != nil {
		return nil, fmt.Errorf("更新用户余额失败: %w", err)
	}

	// ✅ 保存游戏记录
	now := time.Now().Unix()
	startTime := gameState.StartTime
	if startTime == 0 {
		startTime = now - 300
	}

	gameRecord, err := m.saveGameRecord(ctx, roomID, room.GameType, gameState, settlement, startTime, now)
	if err != nil {
		return nil, fmt.Errorf("保存游戏记录失败: %w", err)
	}

	// ✅ 保存玩家对局记录
	if err := m.saveGamePlayers(ctx, roomID, gameState, settlement); err != nil {
		return nil, fmt.Errorf("保存玩家记录失败: %w", err)
	}

	// ✅ 通过 Repository 更新房间状态为已结束
	room.Status = 3
	if err := m.roomRepo.Update(ctx, room); err != nil {
		return nil, fmt.Errorf("更新房间状态失败: %w", err)
	}

	// ✅ 通过 Service 更新排行榜
	scores := make(map[uint]float64, len(settlement.Players))
	for userID, info := range settlement.Players {
		scores[userID] = info.Balance
	}
	_ = m.leaderboardSvc.UpdateLeaderboard(ctx, room.GameType, scores)

	settlement.RecordID = gameRecord.ID
	return settlement, nil
}

// ==================== 私有辅助方法 ====================

func (m *Manager) getEngine(gameType string) (services.GameEngine, error) {
	engine, ok := m.engines[gameType]
	if !ok {
		return nil, fmt.Errorf("未知的游戏类型: %s", gameType)
	}
	return engine, nil
}

func (m *Manager) startRunningFastGame(roomID string, players []services.PlayerInfo) (*models.GameState, error) {
	playerCount := len(players)

	engine, err := m.getEngine("running")
	if err != nil {
		return nil, err
	}

	// 发牌
	hands, err := engine.DealCards(playerCount)
	if err != nil {
		return nil, err
	}

	// 创建游戏状态
	now := time.Now().Unix()
	gameState := &models.GameState{
		RoomID:        roomID,
		GameType:      "running",
		Status:        1,
		Round:         1,
		CurrentPlayer: 0,
		Players:       make(map[uint]*models.PlayerGameInfo),
		StartTime:     now,
	}

	// 初始化玩家游戏信息
	firstPlayer := uint(0)
	minCard := 999

	for i, player := range players {
		playerID := player.UserID
		cards := hands[uint(i+1)]

		// 查找手牌中最小的牌（确定首出玩家）
		for _, card := range cards {
			val := models.GetCardValue(card)
			if val < minCard && val != models.CardJoker && val != models.CardKing {
				minCard = val
				firstPlayer = playerID
			}
		}

		gameState.Players[playerID] = &models.PlayerGameInfo{
			UserID:     playerID,
			Position:   player.Position,
			Cards:      cards,
			CardCount:  len(cards),
			IsPassed:   false,
			IsFinished: false,
			Rank:       0,
		}
	}

	if firstPlayer == 0 && len(players) > 0 {
		firstPlayer = players[0].UserID
	}
	gameState.CurrentPlayer = firstPlayer

	return gameState, nil
}

func (m *Manager) startBullGame(roomID string, players []services.PlayerInfo, bullGame *services.BullGame) (*models.GameState, error) {
	playerCount := len(players)

	// 发牌（每人5张）
	hands, err := bullGame.DealCards(playerCount)
	if err != nil {
		return nil, err
	}

	// 创建游戏状态
	now := time.Now().Unix()
	gameState := &models.GameState{
		RoomID:        roomID,
		GameType:      "bull",
		Status:        1,
		Round:         1,
		CurrentPlayer: 0,
		Players:       make(map[uint]*models.PlayerGameInfo),
		StartTime:     now,
	}

	// 初始化玩家游戏信息
	for i, player := range players {
		playerID := player.UserID
		cards := hands[uint(i+1)]

		playerInfo := &models.PlayerGameInfo{
			UserID:     playerID,
			Position:   player.Position,
			Cards:      cards,
			CardCount:  len(cards),
			IsFinished: false,
			IsPassed:   false,
			Rank:       0,
		}

		gameState.Players[playerID] = playerInfo

		// 找出牛最大的玩家作为庄家
		if gameState.CurrentPlayer == 0 {
			gameState.CurrentPlayer = playerID
		} else {
			currentCards := gameState.Players[gameState.CurrentPlayer].Cards
			if bullGame.CompareBull(cards, currentCards) > 0 {
				gameState.CurrentPlayer = playerID
			}
		}
	}

	return gameState, nil
}

func (m *Manager) calculateSettlement(gameState *models.GameState, baseBet float64) *GameSettlement {
	settlement := &GameSettlement{
		RoomID:  gameState.RoomID,
		Players: make(map[uint]*PlayerSettlement),
	}

	// 获取所有玩家的名次
	rankedPlayers := make([]*models.PlayerGameInfo, 0, len(gameState.Players))
	for _, playerInfo := range gameState.Players {
		rankedPlayers = append(rankedPlayers, playerInfo)
	}

	// 按名次排序
	sort.Slice(rankedPlayers, func(i, j int) bool {
		return rankedPlayers[i].Rank < rankedPlayers[j].Rank
	})

	// 计算每个玩家的输赢
	playerCount := len(rankedPlayers)
	for i, playerInfo := range rankedPlayers {
		rank := i + 1
		var balance float64

		if rank == 1 {
			balance = float64(playerCount-1) * baseBet
		} else {
			balance = -float64(rank-1) * baseBet
		}

		settlement.Players[playerInfo.UserID] = &PlayerSettlement{
			UserID:  playerInfo.UserID,
			Rank:    rank,
			Balance: balance,
		}
	}

	return settlement
}

func (m *Manager) saveGameRecord(ctx context.Context, roomID, gameType string, gameState *models.GameState, settlement *GameSettlement, startTime, endTime int64) (*models.GameRecord, error) {
	// 构建玩家列表
	playersData := make([]map[string]interface{}, 0, len(gameState.Players))
	for userID, playerInfo := range gameState.Players {
		playersData = append(playersData, map[string]interface{}{
			"user_id":    userID,
			"position":   playerInfo.Position,
			"rank":       playerInfo.Rank,
			"card_count": playerInfo.CardCount,
		})
	}
	playersJSON, _ := json.Marshal(playersData)

	// 构建结算结果
	resultData := make(map[string]interface{})
	for userID, playerSettlement := range settlement.Players {
		resultData[fmt.Sprintf("%d", userID)] = map[string]interface{}{
			"user_id":       playerSettlement.UserID,
			"rank":          playerSettlement.Rank,
			"balance":       playerSettlement.Balance,
			"final_balance": playerSettlement.FinalBalance,
		}
	}
	resultJSON, _ := json.Marshal(resultData)

	// 创建游戏记录
	gameRecord := &models.GameRecord{
		RoomID:    roomID,
		GameType:  gameType,
		Players:   models.JSON(playersJSON),
		Result:    models.JSON(resultJSON),
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  int(endTime - startTime),
	}

	// ✅ 通过 Repository 保存
	if err := m.gameRecordRepo.CreateGameRecord(ctx, gameRecord); err != nil {
		return nil, err
	}

	return gameRecord, nil
}

func (m *Manager) saveGamePlayers(ctx context.Context, roomID string, gameState *models.GameState, settlement *GameSettlement) error {
	players := make([]*models.GamePlayer, 0, len(gameState.Players))

	for userID, playerInfo := range gameState.Players {
		playerSettlement, ok := settlement.Players[userID]
		if !ok {
			continue
		}

		players = append(players, &models.GamePlayer{
			RoomID:   roomID,
			UserID:   userID,
			Position: playerInfo.Position,
			Balance:  playerSettlement.Balance,
		})
	}

	// ✅ 通过 Repository 批量保存
	return m.gameRecordRepo.BatchCreateGamePlayers(ctx, players)
}

// checkGameEnd 检查游戏是否结束（内部方法）
func (m *Manager) checkGameEnd(ctx context.Context, roomID string, gameState *models.GameState) (bool, *models.GameState) {
	// 统计已完成玩家数
	finishedCount := 0
	for _, playerInfo := range gameState.Players {
		if playerInfo.IsFinished {
			finishedCount++
		}
	}

	// 如果只剩一个人未完成或所有人都完成了，游戏结束
	if finishedCount >= len(gameState.Players)-1 {
		// 如果还有一人未完成，标记他为最后一名
		if finishedCount == len(gameState.Players)-1 {
			for userID, playerInfo := range gameState.Players {
				if !playerInfo.IsFinished {
					playerInfo.IsFinished = true
					playerInfo.Rank = m.calculateRank(gameState)
					gameState.Players[userID] = playerInfo
					break
				}
			}
		}

		gameState.Status = 3 // 已结束
		_ = m.stateStorage.Save(ctx, gameState, 2*time.Hour)
		return true, gameState
	}

	return false, gameState
}

// settleBullGame 结算牛牛游戏
func (m *Manager) settleBullGame(ctx context.Context, roomID string, gameState *models.GameState, bullGame *services.BullGame) (*GameSettlement, error) {
	// ✅ 通过 Repository 获取房间信息
	room, err := m.gameRecordRepo.GetRoomByRoomID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("房间不存在: %w", err)
	}

	// 收集所有玩家的牛牛结果
	type PlayerBull struct {
		UserID      uint
		PlayerInfo  *models.PlayerGameInfo
		PlayedCards []int
		BullType    int
		BullNum     int
		MaxCard     int
	}

	playerBulls := make([]PlayerBull, 0)
	for userID, playerInfo := range gameState.Players {
		// 如果玩家还没有出牌，使用原始手牌计算
		var cards []int
		if len(playerInfo.PlayedCards) > 0 {
			cards = playerInfo.PlayedCards
		} else if len(playerInfo.Cards) > 0 {
			cards = playerInfo.Cards
		} else {
			continue
		}

		// 计算或使用已存储的牛牛结果
		bullType := playerInfo.BullType
		bullNum := playerInfo.BullNum
		maxCard := playerInfo.MaxCard

		// 如果还没有计算，重新计算
		if bullType == 0 && bullNum == 0 && maxCard == 0 {
			bullType, bullNum, maxCard = bullGame.CalculateBull(cards)
			playerInfo.BullType = bullType
			playerInfo.BullNum = bullNum
			playerInfo.MaxCard = maxCard
			playerInfo.PlayedCards = cards
		}

		playerBulls = append(playerBulls, PlayerBull{
			UserID:      userID,
			PlayerInfo:  playerInfo,
			PlayedCards: cards,
			BullType:    bullType,
			BullNum:     bullNum,
			MaxCard:     maxCard,
		})
	}

	// 按照牛牛类型和牛数排序（从大到小）
	sort.Slice(playerBulls, func(i, j int) bool {
		if playerBulls[i].BullType != playerBulls[j].BullType {
			return playerBulls[i].BullType > playerBulls[j].BullType
		}
		if playerBulls[i].BullNum != playerBulls[j].BullNum {
			return playerBulls[i].BullNum > playerBulls[j].BullNum
		}
		return playerBulls[i].MaxCard > playerBulls[j].MaxCard
	})

	// 分配名次
	for i, pb := range playerBulls {
		pb.PlayerInfo.Rank = i + 1
		gameState.Players[pb.UserID] = pb.PlayerInfo
	}

	// 计算结算结果
	settlement := &GameSettlement{
		RoomID:  roomID,
		Players: make(map[uint]*PlayerSettlement),
	}

	// 牛牛规则：第一名获得所有玩家的底注，其他人扣除底注
	playerCount := len(playerBulls)
	baseBet := room.BaseBet

	for _, pb := range playerBulls {
		rank := pb.PlayerInfo.Rank
		var balance float64

		if rank == 1 {
			balance = float64(playerCount-1) * baseBet
		} else {
			balance = -baseBet
		}

		settlement.Players[pb.UserID] = &PlayerSettlement{
			UserID:  pb.UserID,
			Rank:    rank,
			Balance: balance,
		}
	}

	// 执行通用结算流程
	return m.executeSettlement(ctx, roomID, room, gameState, settlement)
}

// executeSettlement 执行结算流程（通用方法）
func (m *Manager) executeSettlement(ctx context.Context, roomID string, room *models.GameRoom, gameState *models.GameState, settlement *GameSettlement) (*GameSettlement, error) {
	// 准备批量更新余额的数据
	balanceUpdates := make(map[uint]float64)
	for userID, playerSettlement := range settlement.Players {
		user, err := m.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("用户不存在: %d", userID)
		}

		newBalance := user.Balance + playerSettlement.Balance
		if newBalance < 0 {
			newBalance = 0
		}

		balanceUpdates[userID] = newBalance
		playerSettlement.FinalBalance = newBalance
	}

	// ✅ 批量更新余额（使用事务）
	if err := m.userRepo.BatchUpdateBalances(ctx, balanceUpdates); err != nil {
		return nil, fmt.Errorf("更新用户余额失败: %w", err)
	}

	// 保存游戏记录
	now := time.Now().Unix()
	startTime := gameState.StartTime
	if startTime == 0 {
		startTime = now - 300
	}

	gameRecord, err := m.saveGameRecord(ctx, roomID, room.GameType, gameState, settlement, startTime, now)
	if err != nil {
		return nil, fmt.Errorf("保存游戏记录失败: %w", err)
	}

	// 保存玩家对局记录
	if err := m.saveGamePlayers(ctx, roomID, gameState, settlement); err != nil {
		return nil, fmt.Errorf("保存玩家记录失败: %w", err)
	}

	// 更新房间状态为已结束
	room.Status = 3
	if err := m.roomRepo.Update(ctx, room); err != nil {
		return nil, fmt.Errorf("更新房间状态失败: %w", err)
	}

	// 更新排行榜
	scores := make(map[uint]float64, len(settlement.Players))
	for userID, info := range settlement.Players {
		scores[userID] = info.Balance
	}
	_ = m.leaderboardSvc.UpdateLeaderboard(ctx, room.GameType, scores)

	settlement.RecordID = gameRecord.ID
	return settlement, nil
}

// hasCards 检查是否拥有这些牌
func (m *Manager) hasCards(handCards []int, playCards []int) bool {
	cardMap := make(map[int]int)
	for _, card := range handCards {
		cardMap[card]++
	}

	for _, card := range playCards {
		if cardMap[card] <= 0 {
			return false
		}
		cardMap[card]--
	}

	return true
}

// removeCards 从手牌中移除牌
func (m *Manager) removeCards(handCards []int, playCards []int) []int {
	cardMap := make(map[int]int)
	for _, card := range playCards {
		cardMap[card]++
	}

	result := make([]int, 0, len(handCards))
	for _, card := range handCards {
		if cardMap[card] > 0 {
			cardMap[card]--
			continue
		}
		result = append(result, card)
	}

	return result
}

// getNextPlayer 获取下一个出牌玩家
func (m *Manager) getNextPlayer(gameState *models.GameState, currentUserID uint) uint {
	// 获取所有玩家ID
	players := make([]uint, 0, len(gameState.Players))
	for userID := range gameState.Players {
		players = append(players, userID)
	}

	// 找到当前玩家的位置
	currentIndex := -1
	for i, userID := range players {
		if userID == currentUserID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return 0
	}

	// 找到下一个未完成的玩家
	for i := 0; i < len(players); i++ {
		nextIndex := (currentIndex + i + 1) % len(players)
		nextUserID := players[nextIndex]

		playerInfo := gameState.Players[nextUserID]
		if !playerInfo.IsFinished {
			return nextUserID
		}
	}

	return 0
}

// getActivePlayerCount 获取活跃玩家数量
func (m *Manager) getActivePlayerCount(gameState *models.GameState) int {
	count := 0
	for _, playerInfo := range gameState.Players {
		if !playerInfo.IsFinished {
			count++
		}
	}
	return count
}

// calculateRank 计算玩家名次
func (m *Manager) calculateRank(gameState *models.GameState) int {
	rank := 1
	for _, playerInfo := range gameState.Players {
		if playerInfo.IsFinished && playerInfo.Rank > 0 {
			if playerInfo.Rank >= rank {
				rank = playerInfo.Rank + 1
			}
		}
	}
	return rank
}
