package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/cache"
	roomrepo "github.com/kaifa/game-platform/internal/repository/room"
	gamesvc "github.com/kaifa/game-platform/internal/service/game"
	gamerecordsvc "github.com/kaifa/game-platform/internal/service/gamerecord"
	roomsvc "github.com/kaifa/game-platform/internal/service/room"
	"github.com/kaifa/game-platform/pkg/models"
)

var (
	roomService       roomsvc.Service
	gameManager       *gamesvc.Manager // ✅ 使用新的 GameManager
	gameRecordService gamerecordsvc.Service
)

// SetRoomService 注入房间服务实现。
func SetRoomService(service roomsvc.Service) {
	roomService = service
}

// SetGameManager 注入游戏管理器实现（使用新的重构版本）
func SetGameManager(manager *gamesvc.Manager) {
	gameManager = manager
}

func ensureRoomService(c *gin.Context) bool {
	if roomService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "房间服务未初始化"})
		return false
	}
	return true
}

func ensureGameManager(c *gin.Context) bool {
	if gameManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "游戏管理器未初始化"})
		return false
	}
	return true
}

// SetGameRecordService 注入游戏记录服务实现。
func SetGameRecordService(service gamerecordsvc.Service) {
	gameRecordService = service
}

func ensureGameRecordService(c *gin.Context) bool {
	if gameRecordService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "游戏记录服务未初始化"})
		return false
	}
	return true
}

// GameList 游戏列表
func GameList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"games": []map[string]interface{}{
			{"id": 1, "name": "德州扑克", "type": "texas"},
			{"id": 2, "name": "牛牛", "type": "bull"},
			{"id": 3, "name": "跑得快", "type": "running"},
		},
	})
}

// CreateRoom 创建房间
func CreateRoom(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	var req roomsvc.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	room, err := roomService.CreateRoom(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "创建房间成功", "data": room})
}

// JoinRoom 加入房间
func JoinRoom(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")

	var req struct {
		Password string `json:"password"` // 房间密码（可选）
	}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		return
	}

	room, err := roomService.JoinRoom(c.Request.Context(), userID.(uint), roomID, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "加入房间成功", "data": room})
}

// LeaveRoom 离开房间
func LeaveRoom(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")
	if err := roomService.LeaveRoom(c.Request.Context(), userID.(uint), roomID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "离开房间成功"})
}

// GetRoom 房间信息
func GetRoom(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	roomID := c.Param("roomId")
	room, err := roomService.GetRoom(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": room})
}

// RoomList 房间列表
func RoomList(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	gameType := c.Query("game_type")
	statusStr := c.DefaultQuery("status", "1")
	limitStr := c.DefaultQuery("limit", "20")
	var status int8 = 1
	var limit = 20
	fmt.Sscanf(statusStr, "%d", &status)
	fmt.Sscanf(limitStr, "%d", &limit)
	rooms, err := roomService.ListRooms(c.Request.Context(), roomrepo.ListFilter{
		GameType: gameType,
		Status:   status,
		Limit:    limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": rooms})
}

// Ready 玩家准备
func Ready(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")
	room, err := roomService.Ready(c.Request.Context(), userID.(uint), roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "准备成功", "data": room})
}

// CancelReady 取消准备
func CancelReady(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")
	room, err := roomService.CancelReady(c.Request.Context(), userID.(uint), roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "取消准备成功", "data": room})
}

// StartGame 开始游戏
func StartGame(c *gin.Context) {
	if !ensureRoomService(c) {
		return
	}
	if !ensureGameManager(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")

	// ✅ 使用 RoomService 启动游戏流程
	room, err := roomService.StartGame(c.Request.Context(), userID.(uint), roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// ✅ 使用新的 GameManager 获取游戏状态（过滤当前用户的手牌）
	gameState, err := gameManager.GetGameStateForUser(c.Request.Context(), roomID, userID.(uint))
	if err == nil && gameState != nil {
		// 返回游戏状态和房间信息
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "游戏开始",
			"data": gin.H{
				"room":       room,
				"game_state": gameState,
			},
		})
	} else {
		// 如果获取游戏状态失败，只返回房间信息
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "游戏开始", "data": room})
	}
}

// PlayCards 出牌
func PlayCards(c *gin.Context) {
	if !ensureGameManager(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")

	var req struct {
		Cards []int `json:"cards" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	// ✅ 使用新的 GameManager 先获取游戏状态，判断游戏类型
	currentState, err := gameManager.GetGameState(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "获取游戏状态失败", "error": err.Error()})
		return
	}

	var gameState *models.GameState
	// 根据游戏类型调用不同的出牌方法
	if currentState.GameType == "bull" {
		// 牛牛游戏：必须选择5张牌
		if len(req.Cards) != 5 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "牛牛游戏必须选择5张牌"})
			return
		}
		// ✅ 使用新的 GameManager
		gameState, err = gameManager.PlayBullGame(c.Request.Context(), roomID, userID.(uint), req.Cards)
	} else {
		// 其他游戏（跑得快等）
		// ✅ 使用新的 GameManager
		gameState, err = gameManager.PlayCards(c.Request.Context(), roomID, userID.(uint), req.Cards)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 检查游戏状态是否已结束（PlayCards内部已经处理结算）
	if gameState.Status == 3 {
		// 游戏已结束，获取结算结果
		var settlement *gamesvc.GameSettlement
		// 尝试从Redis获取结算结果（如果PlayCards已经保存）
		settlementData, _ := cache.Get(fmt.Sprintf("game:settlement:%s", roomID))
		if settlementData != "" {
			json.Unmarshal([]byte(settlementData), &settlement)
		}

		// 过滤手牌后返回
		filteredState := gameState.FilterForUser(userID.(uint))

		response := gin.H{
			"code":    200,
			"message": "出牌成功，游戏已结束",
			"data": gin.H{
				"game_state": filteredState,
			},
			"game_end": true,
		}

		// 如果有结算结果，添加到响应中
		if settlement != nil {
			response["data"].(gin.H)["settlement"] = settlement
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// 过滤手牌后返回
	filteredState := gameState.FilterForUser(userID.(uint))
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "出牌成功", "data": filteredState})
}

// Pass 过牌
func Pass(c *gin.Context) {
	if !ensureGameManager(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")

	// ✅ 使用新的 GameManager
	gameState, err := gameManager.Pass(c.Request.Context(), roomID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 过滤手牌后返回
	filteredState := gameState.FilterForUser(userID.(uint))
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "过牌成功", "data": filteredState})
}

// GetGameState 获取游戏状态
func GetGameState(c *gin.Context) {
	if !ensureGameManager(c) {
		return
	}
	roomID := c.Param("roomId")

	// 获取用户ID（如果已登录）
	var userID uint
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}

	// 如果有用户ID，过滤手牌；否则返回完整状态（用于观察者，但通常需要认证）
	var gameState *models.GameState
	var err error
	if userID > 0 {
		// ✅ 使用新的 GameManager
		gameState, err = gameManager.GetGameStateForUser(c.Request.Context(), roomID, userID)
	} else {
		// 未登录用户只能看到基本信息（不包含手牌）
		// ✅ 使用新的 GameManager
		gameState, err = gameManager.GetGameState(c.Request.Context(), roomID)
		if err == nil {
			// 隐藏所有手牌
			filtered := gameState.FilterForUser(0) // 使用0作为特殊值，会隐藏所有手牌
			gameState = filtered
		}
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gameState})
}

// GetUserRecords 获取我的游戏记录
func GetUserRecords(c *gin.Context) {
	if !ensureGameRecordService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	// 获取查询参数
	gameType := c.Query("game_type")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	var page, pageSize int
	fmt.Sscanf(pageStr, "%d", &page)
	fmt.Sscanf(pageSizeStr, "%d", &pageSize)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := gameRecordService.GetUserRecords(c.Request.Context(), userID.(uint), gameType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"records":   records,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetRecordDetail 获取游戏记录详情
func GetRecordDetail(c *gin.Context) {
	if !ensureGameRecordService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	recordID := c.Param("id")

	var id uint
	if _, err := fmt.Sscanf(recordID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的记录ID",
		})
		return
	}

	detail, err := gameRecordService.GetRecordDetail(c.Request.Context(), id, userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    detail,
	})
}

// GetRoomRecords 获取房间的游戏记录
func GetRoomRecords(c *gin.Context) {
	if !ensureGameRecordService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	roomID := c.Param("roomId")

	records, err := gameRecordService.GetRoomRecords(c.Request.Context(), roomID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    records,
	})
}
