package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/kaifa/game-platform/pkg/services"
)

// GetDepositAddresses 获取充值地址列表
func GetDepositAddresses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	chainType := c.Query("chain_type")

	var addresses []models.UserDepositAddress
	query := database.DB.Model(&models.UserDepositAddress{})

	if chainType != "" {
		query = query.Where("chain_type = ?", chainType)
	}

	offset := (page - 1) * pageSize
	var total int64
	query.Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&addresses)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":       addresses,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// CollectUSDT 执行USDT归集
func CollectUSDT(c *gin.Context) {
	paymentService := services.NewPaymentService()

	var req struct {
		UserID    uint   `json:"user_id" binding:"required"`
		ChainType string `json:"chain_type" binding:"required,oneof=trc20 erc20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	txHash, err := paymentService.CollectUSDT(req.UserID, req.ChainType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "归集成功",
		"data": gin.H{
			"tx_hash": txHash,
		},
	})
}

// BatchCollectUSDT 批量归集USDT
func BatchCollectUSDT(c *gin.Context) {
	paymentService := services.NewPaymentService()

	var req struct {
		ChainType string `json:"chain_type" binding:"required,oneof=trc20 erc20"`
		Limit     int    `json:"limit" binding:"min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	if req.Limit == 0 {
		req.Limit = 10 // 默认10个
	}

	err := paymentService.BatchCollectUSDT(req.ChainType, req.Limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "批量归集成功",
	})
}
