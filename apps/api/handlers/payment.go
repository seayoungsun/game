package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	paymentsvc "github.com/kaifa/game-platform/internal/service/payment"
	"github.com/kaifa/game-platform/pkg/models"
)

var (
	paymentService paymentsvc.Service
)

// SetPaymentService 注入支付服务实现
func SetPaymentService(service paymentsvc.Service) {
	paymentService = service
}

func ensurePaymentService(c *gin.Context) bool {
	if paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "支付服务未初始化"})
		return false
	}
	return true
}

// GetPaymentConfig 获取支付配置（公开接口，前端可以使用）
func GetPaymentConfig(c *gin.Context) {
	// 获取支付相关配置
	var configs []models.SystemConfig
	database.DB.Where("group_name = ? AND is_public = ?", "payment", true).
		Or("config_key IN (?)", []string{"min_recharge_amount", "max_recharge_amount", "min_withdraw_amount", "max_withdraw_amount", "withdraw_fee_rate"}).
		Find(&configs)

	// 构建配置映射
	configMap := make(map[string]interface{})
	for _, config := range configs {
		// 根据配置类型转换值
		switch config.ConfigType {
		case "int":
			if val, err := strconv.ParseInt(config.ConfigValue, 10, 64); err == nil {
				configMap[config.ConfigKey] = val
			}
		case "float":
			if val, err := strconv.ParseFloat(config.ConfigValue, 64); err == nil {
				configMap[config.ConfigKey] = val
			}
		case "bool":
			configMap[config.ConfigKey] = config.ConfigValue == "true"
		default:
			configMap[config.ConfigKey] = config.ConfigValue
		}
	}

	// 设置默认值（如果配置不存在）
	if _, ok := configMap["min_recharge_amount"]; !ok {
		configMap["min_recharge_amount"] = 10.0
	}
	if _, ok := configMap["max_recharge_amount"]; !ok {
		configMap["max_recharge_amount"] = 10000.0
	}
	if _, ok := configMap["min_withdraw_amount"]; !ok {
		configMap["min_withdraw_amount"] = 50.0
	}
	if _, ok := configMap["max_withdraw_amount"]; !ok {
		configMap["max_withdraw_amount"] = 5000.0
	}
	if _, ok := configMap["withdraw_fee_rate"]; !ok {
		configMap["withdraw_fee_rate"] = 0.001
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": configMap,
	})
}

// CreateRechargeOrder 创建充值订单
func CreateRechargeOrder(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	var req struct {
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		ChainType string  `json:"chain_type" binding:"required,oneof=trc20 erc20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	// ✅ 使用新的 PaymentService
	order, err := paymentService.CreateRechargeOrder(c.Request.Context(), userID.(uint), req.Amount, req.ChainType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    order,
	})
}

// GetRechargeOrder 获取充值订单
func GetRechargeOrder(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	orderID := c.Param("orderId")

	// ✅ 使用新的 PaymentService
	order, err := paymentService.GetRechargeOrder(c.Request.Context(), orderID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    order,
	})
}

// GetUserRechargeOrders 获取用户的充值订单列表
func GetUserRechargeOrders(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// ✅ 使用新的 PaymentService
	orders, total, err := paymentService.GetUserRechargeOrders(c.Request.Context(), userID.(uint), page, pageSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"orders":    orders,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// CheckRechargeTransaction 手动检查充值交易
func CheckRechargeTransaction(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	orderID := c.Param("orderId")

	// 验证订单归属
	// ✅ 使用新的 PaymentService
	order, err := paymentService.GetRechargeOrder(c.Request.Context(), orderID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 检查交易
	// ✅ 使用新的 PaymentService
	if err := paymentService.CheckTransaction(c.Request.Context(), order.OrderID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 重新获取订单信息
	updatedOrder, _ := paymentService.GetRechargeOrder(c.Request.Context(), orderID, userID.(uint))

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "检查完成",
		"data":    updatedOrder,
	})
}

// CreateWithdrawOrder 创建提现订单
func CreateWithdrawOrder(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	var req struct {
		Amount    float64 `json:"amount" binding:"required,gt=0"`
		ChainType string  `json:"chain_type" binding:"required,oneof=trc20 erc20"`
		ToAddress string  `json:"to_address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	// ✅ 使用新的 PaymentService
	order, err := paymentService.CreateWithdrawOrder(c.Request.Context(), userID.(uint), req.Amount, req.ChainType, req.ToAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "提现订单创建成功",
		"data":    order,
	})
}

// GetWithdrawOrder 获取提现订单
func GetWithdrawOrder(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")
	orderID := c.Param("orderId")

	// ✅ 使用新的 PaymentService
	order, err := paymentService.GetWithdrawOrder(c.Request.Context(), orderID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data":    order,
	})
}

// GetUserWithdrawOrders 获取用户的提现订单列表
func GetUserWithdrawOrders(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	userID, _ := c.Get("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// ✅ 使用新的 PaymentService
	orders, total, err := paymentService.GetUserWithdrawOrders(c.Request.Context(), userID.(uint), page, pageSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询成功",
		"data": gin.H{
			"orders":    orders,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// AuditWithdrawOrder 审核提现订单（管理员操作）
func AuditWithdrawOrder(c *gin.Context) {
	if !ensurePaymentService(c) {
		return
	}
	auditorID, _ := c.Get("user_id") // 当前用户作为审核员
	orderID := c.Param("orderId")

	var req struct {
		Approve bool   `json:"approve" binding:"required"`
		Remark  string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	// ✅ 使用新的 PaymentService
	err := paymentService.AuditWithdrawOrder(c.Request.Context(), auditorID.(uint), orderID, req.Approve, req.Remark)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 重新获取订单信息（管理员可以查询所有订单，传入userID=0）
	order, _ := paymentService.GetWithdrawOrder(c.Request.Context(), orderID, 0)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "审核成功",
		"data":    order,
	})
}
