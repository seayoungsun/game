package room

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kaifa/game-platform/internal/lock"
	roomrepo "github.com/kaifa/game-platform/internal/repository/room"
	userrepo "github.com/kaifa/game-platform/internal/repository/user"
	gamesvc "github.com/kaifa/game-platform/internal/service/game"
	"github.com/kaifa/game-platform/internal/worker"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/kaifa/game-platform/pkg/services"
	"github.com/kaifa/game-platform/pkg/utils"
	"github.com/redis/go-redis/v9"
)

// Service 抽象房间业务服务接口。
// 后续将逐步把 pkg/services/room_service.go 中的业务逻辑迁移至此。
type Service interface {
	CreateRoom(ctx context.Context, ownerID uint, req *CreateRoomRequest) (*models.GameRoom, error)
	JoinRoom(ctx context.Context, userID uint, roomID, password string) (*models.GameRoom, error)
	LeaveRoom(ctx context.Context, userID uint, roomID string) error
	GetRoom(ctx context.Context, roomID string) (*models.GameRoom, error)
	ListRooms(ctx context.Context, filter roomrepo.ListFilter) ([]*models.GameRoom, error)
	Ready(ctx context.Context, userID uint, roomID string) (*models.GameRoom, error)
	CancelReady(ctx context.Context, userID uint, roomID string) (*models.GameRoom, error)
	StartGame(ctx context.Context, userID uint, roomID string) (*models.GameRoom, error)
}

type service struct {
	// Repository 层
	repo     roomrepo.Repository
	userRepo userrepo.Repository

	// Service 依赖
	gameManager *gamesvc.Manager

	// 并发控制组件
	distLock   lock.Lock    // ✅ 分布式锁（用于关键操作）
	localLock  lock.RWLock  // ✅ 本地读写锁（用于快速操作）
	notifyPool *worker.Pool // ✅ 通知 Worker Pool

	// 其他
	redis     *redis.Client
	notifyURL string
}

// New 创建房间服务实例。
func New(
	repo roomrepo.Repository,
	userRepo userrepo.Repository,
	gameManager *gamesvc.Manager,
	redisClient *redis.Client,
	notifyURL string,
	distLock lock.Lock, // ✅ 注入分布式锁
	localLock lock.RWLock, // ✅ 注入本地锁
	notifyPool *worker.Pool, // ✅ 注入通知池
) Service {
	return &service{
		repo:        repo,
		userRepo:    userRepo,
		gameManager: gameManager,
		redis:       redisClient,
		notifyURL:   notifyURL,
		distLock:    distLock,
		localLock:   localLock,
		notifyPool:  notifyPool,
	}
}

// CreateRoomRequest 定义房间创建入参模型。
// 目前仅描述字段，具体校验与业务逻辑将在迁移阶段补充。
type CreateRoomRequest struct {
	GameType   string  `json:"game_type"`
	RoomType   string  `json:"room_type"`
	BaseBet    float64 `json:"base_bet"`
	MaxPlayers int     `json:"max_players"`
	Password   string  `json:"password"`
}

func (s *service) CreateRoom(ctx context.Context, ownerID uint, req *CreateRoomRequest) (*models.GameRoom, error) {
	validGameTypes := map[string]bool{"texas": true, "bull": true, "running": true}
	if !validGameTypes[req.GameType] {
		return nil, errors.New("无效的游戏类型")
	}

	validRoomTypes := map[string]bool{"quick": true, "middle": true, "high": true}
	if !validRoomTypes[req.RoomType] {
		return nil, errors.New("无效的房间类型")
	}

	if req.MaxPlayers < 2 || req.MaxPlayers > 10 {
		return nil, errors.New("人数必须在2-10之间")
	}

	user, err := s.userRepo.GetByID(ctx, ownerID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	player := services.PlayerInfo{
		UserID:   user.ID,
		UID:      user.UID,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Position: 1,
		Ready:    false,
	}
	playersJSON, _ := json.Marshal([]services.PlayerInfo{player})

	roomID := fmt.Sprintf("R%s", uuid.New().String()[:8])

	var passwordHash string
	hasPassword := req.Password != ""
	if hasPassword {
		passwordHash, err = utils.HashPassword(req.Password)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
	}

	room := models.GameRoom{
		RoomID:         roomID,
		GameType:       req.GameType,
		RoomType:       req.RoomType,
		BaseBet:        req.BaseBet,
		MaxPlayers:     req.MaxPlayers,
		CurrentPlayers: 1,
		Status:         1,
		Password:       passwordHash,
		HasPassword:    hasPassword,
		Players:        models.JSON(playersJSON),
		CreatorID:      ownerID,
	}

	if err := s.repo.Create(ctx, &room); err != nil {
		return nil, fmt.Errorf("创建房间失败: %w", err)
	}

	s.syncRoomToRedis(ctx, &room)
	go s.notifyGameServer(ctx, roomID, "room_created", ownerID, &room)

	return &room, nil
}

func (s *service) JoinRoom(ctx context.Context, userID uint, roomID, password string) (*models.GameRoom, error) {
	// ✅ 使用本地写锁保护加入房间操作（防止并发加入导致超员）
	var finalRoom *models.GameRoom
	var finalErr error

	err := s.localLock.WithLock(roomID, func() error {
		room, err := s.repo.GetByRoomID(ctx, roomID)
		if err != nil {
			finalErr = errors.New("房间不存在")
			return finalErr
		}

		if room.HasPassword {
			if password == "" {
				finalErr = errors.New("房间需要密码")
				return finalErr
			}
			if err := utils.CheckPassword(room.Password, password); err != nil {
				finalErr = errors.New("房间密码错误")
				return finalErr
			}
		}

		if room.Status != 1 {
			finalErr = errors.New("房间已开始或已结束")
			return finalErr
		}

		// ✅ 在锁保护下检查人数（防止竞态条件）
		if room.CurrentPlayers >= room.MaxPlayers {
			finalErr = errors.New("房间已满")
			return finalErr
		}

		var players []services.PlayerInfo
		if err := json.Unmarshal(room.Players, &players); err != nil {
			finalErr = fmt.Errorf("解析玩家列表失败: %w", err)
			return finalErr
		}

		// 检查是否已在房间中
		for _, p := range players {
			if p.UserID == userID {
				finalRoom = room
				return nil
			}
		}

		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			finalErr = errors.New("用户不存在")
			return finalErr
		}

		players = append(players, services.PlayerInfo{
			UserID:   user.ID,
			UID:      user.UID,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			Position: len(players) + 1,
			Ready:    false,
		})

		playersJSON, _ := json.Marshal(players)
		room.Players = models.JSON(playersJSON)
		room.CurrentPlayers = len(players)

		// ✅ 在锁保护下更新（原子操作）
		if err := s.repo.Update(ctx, room); err != nil {
			finalErr = fmt.Errorf("加入房间失败: %w", err)
			return finalErr
		}

		s.syncRoomToRedis(ctx, room)

		// ✅ 使用 Worker Pool 异步发送通知（不阻塞）
		s.asyncNotifyGameServer(ctx, roomID, "join", userID, room)

		finalRoom = room
		return nil
	})

	if err != nil {
		return nil, finalErr
	}

	return finalRoom, nil
}

func (s *service) LeaveRoom(ctx context.Context, userID uint, roomID string) error {
	room, err := s.repo.GetByRoomID(ctx, roomID)
	if err != nil {
		return errors.New("房间不存在")
	}
	if room.Status == 2 {
		return errors.New("游戏中不能离开")
	}

	var players []services.PlayerInfo
	if err := json.Unmarshal(room.Players, &players); err != nil {
		return fmt.Errorf("解析玩家列表失败: %w", err)
	}

	newPlayers := make([]services.PlayerInfo, 0, len(players))
	removed := false
	for _, p := range players {
		if p.UserID != userID {
			newPlayers = append(newPlayers, p)
		} else {
			removed = true
		}
	}
	if !removed {
		return errors.New("不在该房间中")
	}

	if len(newPlayers) == 0 {
		if err := s.repo.DeleteByRoomID(ctx, roomID); err != nil {
			return err
		}
		s.deleteRoomFromRedis(ctx, roomID)
		go s.notifyGameServer(ctx, roomID, "room_deleted", userID, nil)
		return nil
	}

	playersJSON, _ := json.Marshal(newPlayers)
	room.Players = models.JSON(playersJSON)
	room.CurrentPlayers = len(newPlayers)
	if room.CreatorID == userID {
		room.CreatorID = newPlayers[0].UserID
	}

	if err := s.repo.Update(ctx, room); err != nil {
		return fmt.Errorf("离开房间失败: %w", err)
	}

	s.syncRoomToRedis(ctx, room)
	go s.notifyGameServer(ctx, roomID, "leave", userID, nil)
	return nil
}

func (s *service) GetRoom(ctx context.Context, roomID string) (*models.GameRoom, error) {
	room, err := s.repo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, errors.New("房间不存在")
	}
	return room, nil
}

func (s *service) ListRooms(ctx context.Context, filter roomrepo.ListFilter) ([]*models.GameRoom, error) {
	return s.repo.List(ctx, filter)
}

func (s *service) Ready(ctx context.Context, userID uint, roomID string) (*models.GameRoom, error) {
	room, err := s.repo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, errors.New("房间不存在")
	}
	if room.Status != 1 {
		return nil, errors.New("只能等待中房间准备")
	}

	var players []services.PlayerInfo
	if err := json.Unmarshal(room.Players, &players); err != nil {
		return nil, fmt.Errorf("解析玩家列表失败: %w", err)
	}

	found := false
	for i := range players {
		if players[i].UserID == userID {
			players[i].Ready = true
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("不在该房间中")
	}

	playersJSON, _ := json.Marshal(players)
	room.Players = models.JSON(playersJSON)

	if err := s.repo.Update(ctx, room); err != nil {
		return nil, fmt.Errorf("准备失败: %w", err)
	}

	s.syncRoomToRedis(ctx, room)
	go s.notifyGameServer(ctx, roomID, "ready", userID, room)
	return room, nil
}

func (s *service) CancelReady(ctx context.Context, userID uint, roomID string) (*models.GameRoom, error) {
	room, err := s.repo.GetByRoomID(ctx, roomID)
	if err != nil {
		return nil, errors.New("房间不存在")
	}
	if room.Status != 1 {
		return nil, errors.New("只能等待中房间取消准备")
	}

	var players []services.PlayerInfo
	if err := json.Unmarshal(room.Players, &players); err != nil {
		return nil, fmt.Errorf("解析玩家列表失败: %w", err)
	}

	found := false
	for i := range players {
		if players[i].UserID == userID {
			players[i].Ready = false
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("不在该房间中")
	}

	playersJSON, _ := json.Marshal(players)
	room.Players = models.JSON(playersJSON)

	if err := s.repo.Update(ctx, room); err != nil {
		return nil, fmt.Errorf("取消准备失败: %w", err)
	}

	s.syncRoomToRedis(ctx, room)
	go s.notifyGameServer(ctx, roomID, "cancel_ready", userID, room)
	return room, nil
}

func (s *service) StartGame(ctx context.Context, userID uint, roomID string) (*models.GameRoom, error) {
	// ✅ 使用分布式锁保护开始游戏操作（防止重复开始）
	lockKey := fmt.Sprintf("room:%s:start", roomID)

	var finalRoom *models.GameRoom
	var finalErr error

	err := s.distLock.WithLock(ctx, lockKey, 10*time.Second, func() error {
		room, err := s.repo.GetByRoomID(ctx, roomID)
		if err != nil {
			finalErr = errors.New("房间不存在")
			return finalErr
		}

		if room.CreatorID != userID {
			finalErr = errors.New("只有创建者可以开始游戏")
			return finalErr
		}

		canStart, err := s.canStartGame(room)
		if err != nil {
			finalErr = err
			return finalErr
		}
		if !canStart {
			finalErr = errors.New("还有玩家未准备")
			return finalErr
		}

		// ✅ 在锁保护下检查状态（防止重复开始）
		if room.Status != 1 {
			finalErr = errors.New("房间状态不正确")
			return finalErr
		}

		// ✅ 使用注入的 GameManager
		if s.gameManager == nil {
			finalErr = errors.New("游戏管理器未初始化")
			return finalErr
		}

		gameState, err := s.gameManager.StartGame(ctx, roomID)
		if err != nil {
			finalErr = fmt.Errorf("开始游戏失败: %w", err)
			return finalErr
		}

		updatedRoom, err := s.repo.GetByRoomID(ctx, roomID)
		if err == nil {
			s.syncRoomToRedis(ctx, updatedRoom)
			s.pushGameStarted(ctx, roomID, userID, updatedRoom, gameState)
			finalRoom = updatedRoom
			return nil
		}

		s.pushGameStarted(ctx, roomID, userID, room, gameState)
		finalRoom = room
		return nil
	})

	if err != nil {
		return nil, finalErr
	}

	return finalRoom, nil
}

func (s *service) canStartGame(room *models.GameRoom) (bool, error) {
	if room.Status != 1 {
		return false, errors.New("房间状态不正确")
	}
	if room.CurrentPlayers < 2 {
		return false, errors.New("至少需要2人才能开始")
	}

	var players []services.PlayerInfo
	if err := json.Unmarshal(room.Players, &players); err != nil {
		return false, fmt.Errorf("解析玩家列表失败: %w", err)
	}
	for _, p := range players {
		if !p.Ready {
			return false, nil
		}
	}
	return true, nil
}

func (s *service) pushGameStarted(ctx context.Context, roomID string, userID uint, room *models.GameRoom, gameState interface{}) {
	if gameState == nil {
		return
	}
	gameStateJSON, err := json.Marshal(gameState)
	if err != nil {
		return
	}
	var gameStateMap map[string]interface{}
	if err := json.Unmarshal(gameStateJSON, &gameStateMap); err != nil {
		return
	}
	data := map[string]interface{}{
		"game_state": gameStateMap,
		"room": map[string]interface{}{
			"id":              room.ID,
			"room_id":         room.RoomID,
			"game_type":       room.GameType,
			"room_type":       room.RoomType,
			"base_bet":        room.BaseBet,
			"max_players":     room.MaxPlayers,
			"current_players": room.CurrentPlayers,
			"status":          room.Status,
			"players":         room.Players,
		},
	}
	go s.notifyGameServerWithData(ctx, roomID, "game_started", userID, data)
}

// notifyGameServer 发送通知（同步，保持兼容旧代码）
func (s *service) notifyGameServer(ctx context.Context, roomID, action string, userID uint, room *models.GameRoom) {
	s.asyncNotifyGameServer(ctx, roomID, action, userID, room)
}

// asyncNotifyGameServer 异步发送通知到游戏服务器（使用 Worker Pool）
func (s *service) asyncNotifyGameServer(ctx context.Context, roomID, action string, userID uint, room *models.GameRoom) {
	if s.notifyURL == "" {
		return
	}

	// 构建请求数据
	req := map[string]interface{}{
		"room_id": roomID,
		"action":  action,
		"user_id": userID,
	}
	if room != nil {
		var players []services.PlayerInfo
		if err := json.Unmarshal(room.Players, &players); err == nil {
			req["room_data"] = map[string]interface{}{
				"id":              room.ID,
				"room_id":         room.RoomID,
				"game_type":       room.GameType,
				"room_type":       room.RoomType,
				"base_bet":        room.BaseBet,
				"max_players":     room.MaxPlayers,
				"current_players": room.CurrentPlayers,
				"status":          room.Status,
				"has_password":    room.HasPassword,
				"players":         players,
			}
		}
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return
	}

	// ✅ 使用 Worker Pool 提交任务（限制并发，防止过载）
	if s.notifyPool != nil {
		s.notifyPool.Submit(func(taskCtx context.Context) error {
			// ✅ 创建带超时的 HTTP 请求
			httpReq, err := http.NewRequestWithContext(taskCtx, "POST", s.notifyURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return err
			}
			httpReq.Header.Set("Content-Type", "application/json")

			// ✅ 使用带超时的 HTTP 客户端
			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := client.Do(httpReq)
			if err != nil {
				return fmt.Errorf("通知游戏服务器失败: %w", err)
			}
			defer resp.Body.Close()

			return nil
		})
	} else {
		// 降级方案：直接发送（如果 Worker Pool 未初始化）
		go func() {
			_, _ = http.Post(s.notifyURL, "application/json", bytes.NewBuffer(jsonData))
		}()
	}
}

func (s *service) notifyGameServerWithData(ctx context.Context, roomID, action string, userID uint, roomData map[string]interface{}) {
	if s.notifyURL == "" {
		return
	}
	req := map[string]interface{}{
		"room_id":   roomID,
		"action":    action,
		"user_id":   userID,
		"room_data": roomData,
	}
	jsonData, err := json.Marshal(req)
	if err != nil {
		return
	}
	go func() {
		_, _ = http.Post(s.notifyURL, "application/json", bytes.NewBuffer(jsonData))
	}()
}

func (s *service) syncRoomToRedis(ctx context.Context, room *models.GameRoom) {
	if s.redis == nil {
		return
	}
	key := fmt.Sprintf("room:%s", room.RoomID)
	roomData := map[string]interface{}{
		"room_id":         room.RoomID,
		"game_type":       room.GameType,
		"room_type":       room.RoomType,
		"base_bet":        room.BaseBet,
		"max_players":     room.MaxPlayers,
		"current_players": room.CurrentPlayers,
		"status":          room.Status,
		"creator_id":      room.CreatorID,
		"updated_at":      room.UpdatedAt,
	}
	for field, value := range roomData {
		_ = s.redis.HSet(ctx, key, field, fmt.Sprintf("%v", value)).Err()
	}
	_ = s.redis.HSet(ctx, key, "players", string(room.Players)).Err()
	_ = s.redis.Expire(ctx, key, time.Hour).Err()
}

func (s *service) deleteRoomFromRedis(ctx context.Context, roomID string) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, fmt.Sprintf("room:%s", roomID)).Err()
}
