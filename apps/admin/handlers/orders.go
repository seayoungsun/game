package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/kaifa/game-platform/pkg/services"
)

// GetRechargeOrders 获取充值订单列表
func GetRechargeOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status, _ := strconv.Atoi(c.Query("status"))
	chainType := c.Query("chain_type")

	var orders []models.RechargeOrder
	query := database.DB.Model(&models.RechargeOrder{})

	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if chainType != "" {
		query = query.Where("chain_type = ?", chainType)
	}

	offset := (page - 1) * pageSize
	var total int64
	query.Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":       orders,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetWithdrawOrders 获取提现订单列表
func GetWithdrawOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status, _ := strconv.Atoi(c.Query("status"))
	chainType := c.Query("chain_type")

	var orders []models.WithdrawOrder
	query := database.DB.Model(&models.WithdrawOrder{})

	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if chainType != "" {
		query = query.Where("chain_type = ?", chainType)
	}

	offset := (page - 1) * pageSize
	var total int64
	query.Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":       orders,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// AuditWithdrawOrder 审核提现订单
func AuditWithdrawOrder(c *gin.Context) {
	orderID := c.Param("orderId")

	var req struct {
		Approve bool   `json:"approve" binding:"required"`
		Remark  string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	// 调用支付服务的审核方法
	paymentService := getPaymentService()

	adminID, _ := c.Get("admin_id")
	adminIDUint := adminID.(uint)

	err := paymentService.AuditWithdrawOrder(adminIDUint, orderID, req.Approve, req.Remark)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "审核成功",
	})
}

// getPaymentService 延迟获取支付服务，避免在配置尚未加载时初始化
func getPaymentService() *services.PaymentService {
	return services.NewPaymentService()
}
