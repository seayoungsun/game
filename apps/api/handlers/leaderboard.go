package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	leaderboardsvc "github.com/kaifa/game-platform/internal/service/leaderboard"
)

var (
	leaderboardService leaderboardsvc.Service
)

// SetLeaderboardService 注入排行榜服务实现。
func SetLeaderboardService(service leaderboardsvc.Service) {
	leaderboardService = service
}

func ensureLeaderboardService(c *gin.Context) bool {
	if leaderboardService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "排行榜服务未初始化"})
		return false
	}
	return true
}

// GetLeaderboard 获取排行榜
func GetLeaderboard(c *gin.Context) {
	if !ensureLeaderboardService(c) {
		return
	}
	gameType := c.Query("game_type")
	if gameType == "" {
		gameType = "running" // 默认跑得快
	}

	period := c.Query("period")
	if period == "" {
		period = "total" // 默认总榜
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	leaderboard, err := leaderboardService.GetLeaderboard(c.Request.Context(), gameType, period, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询排行榜失败",
			"error":   err.Error(),
		})
		return
	}

	// 如果用户已登录，查询用户排名
	userID, exists := c.Get("user_id")
	if exists {
		rank, _, err := leaderboardService.GetUserRank(c.Request.Context(), gameType, period, userID.(uint))
		if err == nil {
			leaderboard.MyRank = rank
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    leaderboard,
	})
}

// GetUserRank 获取我的排名
func GetUserRank(c *gin.Context) {
	if !ensureLeaderboardService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	gameType := c.Query("game_type")
	if gameType == "" {
		gameType = "running"
	}

	period := c.Query("period")
	if period == "" {
		period = "total"
	}

	rank, score, err := leaderboardService.GetUserRank(c.Request.Context(), gameType, period, userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询排名失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"rank":  rank,
			"score": score,
		},
	})
}
