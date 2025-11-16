package services

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PaymentService 支付服务
type PaymentService struct {
	tronAPIURL        string
	etherscanAPIURL   string
	etherscanAPIKey   string
	hdWallet          *HDWallet            // HD钱包（用于派生地址）
	transferService   *USDTTransferService // USDT转账服务
	gasManager        *GasManager          // Gas费用管理器
	collectionService *CollectionService   // USDT归集服务
}

var paymentServiceInstance *PaymentService
var paymentServiceOnce sync.Once

// NewPaymentService 创建支付服务（单例模式）
func NewPaymentService() *PaymentService {
	paymentServiceOnce.Do(func() {
		cfg := config.Get()
		ps := &PaymentService{}

		if cfg != nil {
			// TRC20 API地址
			ps.tronAPIURL = "https://api.trongrid.io"
			// ERC20 API地址（Etherscan）
			ps.etherscanAPIURL = "https://api.etherscan.io/api"
			ps.etherscanAPIKey = cfg.Payment.EtherscanAPIKey // 从配置读取

			// 初始化HD钱包（必须配置助记词）
			if cfg.Payment.MasterMnemonic == "" {
				logger.Logger.Fatal("未配置主钱包助记词，请设置 payment.master_mnemonic 配置项")
			}

			hdWallet, err := NewHDWallet(cfg.Payment.MasterMnemonic)
			if err != nil {
				logger.Logger.Fatal("初始化HD钱包失败",
					zap.Error(err),
					zap.String("error_message", "请检查助记词格式是否正确（12或24个单词，空格分隔）"),
				)
			}

			ps.hdWallet = hdWallet
			logger.Logger.Info("HD钱包初始化成功，将使用BIP44派生地址")

			// 初始化USDT转账服务
			ps.transferService = NewUSDTTransferService(hdWallet)

			// 初始化Gas管理器（使用转账服务的客户端）
			if ps.transferService != nil {
				ps.gasManager = NewGasManager(
					ps.transferService.ethClient,
					ps.transferService.tronClient,
					hdWallet,
				)

				// 初始化归集服务
				ps.collectionService = NewCollectionService(
					ps.transferService.ethClient,
					ps.transferService.tronClient,
					ps.transferService,
					ps.gasManager,
					hdWallet,
				)
			}
		}

		paymentServiceInstance = ps

		// 启动交易监控
		ps.StartTransactionMonitor()
		logger.Logger.Info("支付服务交易监控已启动")
	})
	return paymentServiceInstance
}

// CollectUSDT 归集USDT（从派生地址归集到主钱包）
func (ps *PaymentService) CollectUSDT(userID uint, chainType string) (string, error) {
	if ps.collectionService == nil {
		return "", errors.New("归集服务未初始化")
	}
	return ps.collectionService.CollectUSDT(userID, chainType)
}

// BatchCollectUSDT 批量归集USDT
func (ps *PaymentService) BatchCollectUSDT(chainType string, limit int) error {
	if ps.collectionService == nil {
		return errors.New("归集服务未初始化")
	}
	return ps.collectionService.BatchCollectUSDT(chainType, limit)
}

// getSystemConfigFloat 获取系统配置浮点数值
func getSystemConfigFloat(key string, defaultValue float64) float64 {
	var config models.SystemConfig
	if err := database.DB.Where("config_key = ?", key).First(&config).Error; err == nil {
		value, err := strconv.ParseFloat(config.ConfigValue, 64)
		if err == nil {
			return value
		}
		logger.Logger.Warn("解析系统配置失败",
			zap.String("key", key),
			zap.String("value", config.ConfigValue),
			zap.Error(err),
		)
	}
	return defaultValue
}

// CreateRechargeOrder 创建充值订单
func (ps *PaymentService) CreateRechargeOrder(userID uint, amount float64, chainType string) (*models.RechargeOrder, error) {
	if amount <= 0 {
		return nil, errors.New("充值金额必须大于0")
	}

	// 从系统配置获取最小和最大充值金额
	minAmount := getSystemConfigFloat("min_recharge_amount", 10.0)
	maxAmount := getSystemConfigFloat("max_recharge_amount", 10000.0)

	if amount < minAmount {
		return nil, fmt.Errorf("充值金额不能小于%.2f USDT", minAmount)
	}
	if amount > maxAmount {
		return nil, fmt.Errorf("充值金额不能大于%.2f USDT", maxAmount)
	}

	if chainType != "trc20" && chainType != "erc20" {
		return nil, errors.New("链类型必须是trc20或erc20")
	}

	// 生成订单号
	orderID := fmt.Sprintf("R%s", strings.ToUpper(uuid.New().String()[:15]))

	// 生成充值地址（使用HD钱包派生）
	depositAddr, err := ps.getDepositAddress(userID, chainType)
	if err != nil {
		return nil, fmt.Errorf("获取充值地址失败: %w", err)
	}

	// 计算过期时间（30分钟）
	now := time.Now().Unix()
	expireAt := now + 30*60

	// 确定渠道
	channel := fmt.Sprintf("usdt_%s", chainType)

	// 确定需要确认次数
	requiredConf := 12
	if chainType == "trc20" {
		requiredConf = 20 // TRC20需要20个确认
	}

	order := &models.RechargeOrder{
		OrderID:      orderID,
		UserID:       userID,
		Amount:       amount,
		Status:       1, // 待支付
		Channel:      channel,
		ChainType:    chainType,
		DepositAddr:  depositAddr,
		RequiredConf: requiredConf,
		ExpireAt:     expireAt,
	}

	if err := database.DB.Create(order).Error; err != nil {
		return nil, fmt.Errorf("创建充值订单失败: %w", err)
	}

	logger.Logger.Info("创建充值订单",
		zap.String("order_id", orderID),
		zap.Uint("user_id", userID),
		zap.Float64("amount", amount),
		zap.String("chain_type", chainType),
		zap.String("deposit_addr", depositAddr),
	)

	return order, nil
}

// GetRechargeOrder 获取充值订单
func (ps *PaymentService) GetRechargeOrder(orderID string, userID uint) (*models.RechargeOrder, error) {
	var order models.RechargeOrder
	if err := database.DB.Where("order_id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return nil, errors.New("订单不存在")
	}
	return &order, nil
}

// GetUserRechargeOrders 获取用户的充值订单列表
func (ps *PaymentService) GetUserRechargeOrders(userID uint, page, pageSize int) ([]models.RechargeOrder, int64, error) {
	var orders []models.RechargeOrder
	var total int64

	query := database.DB.Model(&models.RechargeOrder{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// CheckTransaction 检查交易状态
func (ps *PaymentService) CheckTransaction(orderID string) error {
	var order models.RechargeOrder
	if err := database.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return errors.New("订单不存在")
	}

	// 如果已经支付，不需要再检查
	if order.Status == 2 {
		return nil
	}

	// 如果订单已过期
	if time.Now().Unix() > order.ExpireAt {
		order.Status = 3 // 已取消
		database.DB.Save(&order)
		return errors.New("订单已过期")
	}

	// 根据链类型检查交易
	var txHash string
	var confirmCount int
	var err error

	if order.ChainType == "trc20" {
		txHash, confirmCount, err = ps.checkTRC20Transaction(order.DepositAddr, order.Amount)
	} else if order.ChainType == "erc20" {
		txHash, confirmCount, err = ps.checkERC20Transaction(order.DepositAddr, order.Amount)
	} else {
		return errors.New("不支持的链类型")
	}

	if err != nil {
		logger.Logger.Debug("检查交易失败",
			zap.String("order_id", orderID),
			zap.String("chain_type", order.ChainType),
			zap.Error(err),
		)
		return err
	}

	// 如果找到交易，更新订单
	if txHash != "" {
		order.TxHash = txHash
		order.ChannelID = txHash
		order.ConfirmCount = confirmCount

		// 如果确认次数足够，标记为已支付
		if confirmCount >= order.RequiredConf {
			return ps.completeRecharge(order)
		}

		database.DB.Save(&order)
	}

	return nil
}

// completeRecharge 完成充值
func (ps *PaymentService) completeRecharge(order models.RechargeOrder) error {
	// 使用事务确保原子性
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查订单状态
	if order.Status == 2 {
		tx.Rollback()
		return errors.New("订单已处理")
	}

	now := time.Now().Unix()

	// 更新订单状态
	order.Status = 2 // 已支付
	order.PaidAt = &now
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 更新用户余额
	var user models.User
	if err := tx.First(&user, order.UserID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("用户不存在: %w", err)
	}

	newBalance := user.Balance + order.Amount
	if err := tx.Model(&user).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新用户余额失败: %w", err)
	}

	// 更新用户钱包统计
	var wallet models.UserWallet
	if err := tx.Where("user_id = ?", order.UserID).FirstOrCreate(&wallet, models.UserWallet{
		UserID: order.UserID,
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新钱包失败: %w", err)
	}

	wallet.Balance = newBalance
	wallet.TotalIn += order.Amount
	if err := tx.Save(&wallet).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新钱包统计失败: %w", err)
	}

	// 创建交易记录
	transaction := models.Transaction{
		OrderID:   order.OrderID,
		UserID:    order.UserID,
		Type:      "recharge",
		Amount:    order.Amount,
		Status:    2, // 成功
		Channel:   order.Channel,
		ChannelID: order.TxHash,
		Remark:    fmt.Sprintf("USDT充值 - %s", order.ChainType),
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建交易记录失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 发送充值成功通知
	SendOrderNotification(order.UserID, "recharge", order.OrderID, "paid", order.Amount, "")

	logger.Logger.Info("充值完成",
		zap.String("order_id", order.OrderID),
		zap.Uint("user_id", order.UserID),
		zap.Float64("amount", order.Amount),
		zap.String("tx_hash", order.TxHash),
	)

	return nil
}

// getDepositAddress 获取充值地址
// 为每个用户生成唯一的充值地址（使用HD钱包派生）
func (ps *PaymentService) getDepositAddress(userID uint, chainType string) (string, error) {
	// 检查HD钱包是否已初始化
	if ps.hdWallet == nil {
		return "", errors.New("HD钱包未初始化，请配置 payment.master_mnemonic")
	}

	// 先从数据库查询是否已有地址
	var depositAddr models.UserDepositAddress
	err := database.DB.Where("user_id = ? AND chain_type = ?", userID, chainType).First(&depositAddr).Error

	// 如果已存在地址，直接返回
	if err == nil && depositAddr.Address != "" {
		return depositAddr.Address, nil
	}

	// 使用HD钱包派生地址
	var address string
	var path string

	if chainType == "trc20" {
		address, err = ps.hdWallet.DeriveTronAddressByUserID(userID)
		if err != nil {
			return "", fmt.Errorf("派生波场地址失败: %w", err)
		}
		path = GetTronPath(0, uint32(userID))
	} else if chainType == "erc20" {
		ethAddr, err := ps.hdWallet.DeriveEthereumAddressByUserID(userID)
		if err != nil {
			return "", fmt.Errorf("派生以太坊地址失败: %w", err)
		}
		address = ethAddr.Hex()
		path = GetEthereumPath(0, uint32(userID))
	} else {
		return "", fmt.Errorf("不支持的链类型: %s", chainType)
	}

	logger.Logger.Info("使用HD钱包派生地址",
		zap.Uint("user_id", userID),
		zap.String("chain_type", chainType),
		zap.String("path", path),
		zap.String("address", address),
	)

	// 保存到数据库
	newAddr := models.UserDepositAddress{
		UserID:    userID,
		ChainType: chainType,
		Address:   address,
	}

	// 使用事务确保唯一性
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 再次检查是否已有地址（防止并发创建）
		var existing models.UserDepositAddress
		if err := tx.Where("user_id = ? AND chain_type = ?", userID, chainType).First(&existing).Error; err == nil {
			// 已有地址，使用已存在的地址
			address = existing.Address
			return nil
		}

		// 创建新地址
		if err := tx.Create(&newAddr).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Logger.Error("创建用户充值地址失败",
			zap.Uint("user_id", userID),
			zap.String("chain_type", chainType),
			zap.Error(err),
		)
		return "", fmt.Errorf("保存用户充值地址失败: %w", err)
	}

	logger.Logger.Info("为用户生成充值地址",
		zap.Uint("user_id", userID),
		zap.String("chain_type", chainType),
		zap.String("address", address),
		zap.String("method", "HD钱包派生"),
	)

	return address, nil
}

// checkTRC20Transaction 检查TRC20交易
func (ps *PaymentService) checkTRC20Transaction(depositAddr string, amount float64) (string, int, error) {
	// TRC20 USDT 合约地址
	usdtContract := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	// 调用TronGrid API查询账户的TRC20交易
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20?limit=10&only_confirmed=true", ps.tronAPIURL, depositAddr)

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			TransactionID string `json:"transaction_id"`
			TokenInfo     struct {
				Address string `json:"address"`
			} `json:"token_info"`
			BlockTimestamp int64  `json:"block_timestamp"`
			From           string `json:"from"`
			To             string `json:"to"`
			Type           string `json:"type"`
			Value          string `json:"value"`
			BlockNumber    int64  `json:"block_number"`
			Confirmations  int    `json:"confirmations"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}

	if !result.Success {
		return "", 0, errors.New("API返回失败")
	}

	// 查找匹配的交易（USDT合约、转入地址、金额匹配）
	for _, tx := range result.Data {
		if tx.TokenInfo.Address == usdtContract &&
			strings.EqualFold(tx.To, depositAddr) &&
			tx.Type == "Transfer" {
			// 解析金额（TRC20使用6位小数）
			value, _ := strconv.ParseFloat(tx.Value, 64)
			usdtAmount := value / 1000000

			// 金额匹配（允许小误差）
			if usdtAmount >= amount*0.99 && usdtAmount <= amount*1.01 {
				return tx.TransactionID, tx.Confirmations, nil
			}
		}
	}

	return "", 0, errors.New("未找到匹配的交易")
}

// checkERC20Transaction 检查ERC20交易
func (ps *PaymentService) checkERC20Transaction(depositAddr string, amount float64) (string, int, error) {
	// ERC20 USDT 合约地址（主网）
	usdtContract := "0xdAC17F958D2ee523a2206206994597C13D831ec7"

	// 调用Etherscan API查询ERC20转账
	url := fmt.Sprintf("%s?module=account&action=tokentx&contractaddress=%s&address=%s&page=1&offset=10&startblock=0&endblock=99999999&sort=desc&apikey=%s",
		ps.etherscanAPIURL, usdtContract, depositAddr, ps.etherscanAPIKey)

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			Hash          string `json:"hash"`
			From          string `json:"from"`
			To            string `json:"to"`
			Value         string `json:"value"`
			Confirmations string `json:"confirmations"`
			BlockNumber   string `json:"blockNumber"`
			TimeStamp     string `json:"timeStamp"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}

	if result.Status != "1" {
		return "", 0, errors.New("API返回失败: " + result.Message)
	}

	// 查找匹配的交易（转入地址、金额匹配）
	for _, tx := range result.Result {
		if strings.EqualFold(tx.To, depositAddr) {
			// 解析金额（ERC20使用6位小数）
			value, _ := strconv.ParseFloat(tx.Value, 64)
			usdtAmount := value / 1000000

			// 金额匹配（允许小误差）
			if usdtAmount >= amount*0.99 && usdtAmount <= amount*1.01 {
				confirmCount, _ := strconv.Atoi(tx.Confirmations)
				return tx.Hash, confirmCount, nil
			}
		}
	}

	return "", 0, errors.New("未找到匹配的交易")
}

// StartTransactionMonitor 启动交易监控（定时检查待支付订单）
func (ps *PaymentService) StartTransactionMonitor() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	go func() {
		for range ticker.C {
			ps.checkPendingOrders()
		}
	}()
}

// checkPendingOrders 检查待支付的订单
func (ps *PaymentService) checkPendingOrders() {
	var orders []models.RechargeOrder
	if err := database.DB.Where("status = ? AND expire_at > ?", 1, time.Now().Unix()).Find(&orders).Error; err != nil {
		return
	}

	for _, order := range orders {
		go func(o models.RechargeOrder) {
			if err := ps.CheckTransaction(o.OrderID); err != nil {
				logger.Logger.Debug("检查交易失败",
					zap.String("order_id", o.OrderID),
					zap.Error(err),
				)
			}
		}(order)
	}
}

// ==================== 提现相关功能 ====================

// CreateWithdrawOrder 创建提现订单
func (ps *PaymentService) CreateWithdrawOrder(userID uint, amount float64, chainType string, toAddress string) (*models.WithdrawOrder, error) {
	if amount <= 0 {
		return nil, errors.New("提现金额必须大于0")
	}

	if chainType != "trc20" && chainType != "erc20" {
		return nil, errors.New("链类型必须是trc20或erc20")
	}

	// 验证地址格式
	if chainType == "trc20" {
		if !strings.HasPrefix(toAddress, "T") || len(toAddress) != 34 {
			return nil, errors.New("TRC20地址格式错误，应为T开头的34位地址")
		}
	} else {
		if !strings.HasPrefix(toAddress, "0x") || len(toAddress) != 42 {
			return nil, errors.New("ERC20地址格式错误，应为0x开头的42位地址")
		}
	}

	// 检查用户余额
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	// 从系统配置获取最小和最大提现金额
	minWithdraw := getSystemConfigFloat("min_withdraw_amount", 50.0)
	maxWithdraw := getSystemConfigFloat("max_withdraw_amount", 5000.0)

	if amount < minWithdraw {
		return nil, fmt.Errorf("提现金额不能小于%.2f USDT", minWithdraw)
	}
	if amount > maxWithdraw {
		return nil, fmt.Errorf("提现金额不能大于%.2f USDT", maxWithdraw)
	}

	// 计算手续费（从系统配置获取手续费率）
	feeRate := getSystemConfigFloat("withdraw_fee_rate", 0.001)
	fee := amount * feeRate
	// 手续费保留2位小数，向上取整（最小0.01）
	if fee < 0.01 {
		fee = 0.01
	} else {
		fee = math.Ceil(fee*100) / 100
	}

	// 实际到账金额 = 提现金额 - 手续费
	actualAmount := amount - fee

	// 总扣除金额 = 提现金额（手续费从提现金额中扣除，不从余额中额外扣除）
	// 注意：用户提现时，只需要从余额中扣除提现金额，手续费已包含在提现金额中
	totalDeduct := amount

	// 检查用户余额是否足够（需要支付提现金额）
	if user.Balance < totalDeduct {
		return nil, fmt.Errorf("余额不足，需要%.2f USDT（提现金额%.2f，手续费%.2f，实际到账%.2f）", totalDeduct, amount, fee, actualAmount)
	}

	// 生成订单号
	orderID := fmt.Sprintf("W%s", strings.ToUpper(uuid.New().String()[:15]))

	// 确定渠道
	channel := fmt.Sprintf("usdt_%s", chainType)

	order := &models.WithdrawOrder{
		OrderID:      orderID,
		UserID:       userID,
		Amount:       amount,
		Fee:          fee,
		ActualAmount: actualAmount,
		Status:       1, // 待审核
		Channel:      channel,
		ChainType:    chainType,
		ToAddress:    toAddress,
	}

	if err := database.DB.Create(order).Error; err != nil {
		return nil, fmt.Errorf("创建提现订单失败: %w", err)
	}

	logger.Logger.Info("创建提现订单",
		zap.String("order_id", orderID),
		zap.Uint("user_id", userID),
		zap.Float64("amount", amount),
		zap.Float64("fee", fee),
		zap.Float64("actual_amount", actualAmount),
		zap.String("chain_type", chainType),
		zap.String("to_address", toAddress),
	)

	return order, nil
}

// GetWithdrawOrder 获取提现订单
// userID为0时，允许管理员查询所有订单
func (ps *PaymentService) GetWithdrawOrder(orderID string, userID uint) (*models.WithdrawOrder, error) {
	var order models.WithdrawOrder
	query := database.DB.Where("order_id = ?", orderID)
	if userID != 0 {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.First(&order).Error; err != nil {
		return nil, errors.New("订单不存在")
	}
	return &order, nil
}

// GetUserWithdrawOrders 获取用户的提现订单列表
func (ps *PaymentService) GetUserWithdrawOrders(userID uint, page, pageSize int) ([]models.WithdrawOrder, int64, error) {
	var orders []models.WithdrawOrder
	var total int64

	query := database.DB.Model(&models.WithdrawOrder{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// AuditWithdrawOrder 审核提现订单（管理员操作）
// auditorID: 审核员ID
// orderID: 订单ID
// approve: true=通过, false=拒绝
// remark: 审核备注
func (ps *PaymentService) AuditWithdrawOrder(auditorID uint, orderID string, approve bool, remark string) error {
	var order models.WithdrawOrder
	if err := database.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return errors.New("订单不存在")
	}

	// 如果已经审核过，不允许重复审核
	if order.Status != 1 {
		return errors.New("订单已审核，无法重复审核")
	}

	now := time.Now().Unix()

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if approve {
		// 通过审核：冻结用户余额，更新订单状态
		var user models.User
		if err := tx.Where("id = ?", order.UserID).First(&user).Error; err != nil {
			tx.Rollback()
			return errors.New("用户不存在")
		}

		// 检查余额是否足够（需要支付提现金额）
		if user.Balance < order.Amount {
			tx.Rollback()
			return errors.New("用户余额不足")
		}

		// 冻结余额（减少可用余额，扣除提现金额）
		newBalance := user.Balance - order.Amount
		if err := tx.Model(&user).Update("balance", newBalance).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("冻结余额失败: %w", err)
		}

		// 更新用户钱包统计
		var wallet models.UserWallet
		if err := tx.Where("user_id = ?", order.UserID).FirstOrCreate(&wallet, models.UserWallet{
			UserID: order.UserID,
		}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("查询钱包失败: %w", err)
		}

		wallet.Balance = newBalance
		wallet.Frozen += order.Amount
		wallet.TotalOut += order.Amount
		if err := tx.Save(&wallet).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("更新钱包统计失败: %w", err)
		}

		// 更新订单状态为已通过
		order.Status = 2 // 已通过
		order.AuditAt = &now
		order.AuditorID = auditorID
		order.Remark = remark
		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("更新订单状态失败: %w", err)
		}

		// TODO: 这里应该触发实际的USDT转账
		// 当前先标记为已通过，实际转账功能待实现
		logger.Logger.Info("提现订单审核通过，等待执行转账",
			zap.String("order_id", orderID),
			zap.Uint("user_id", order.UserID),
			zap.Float64("amount", order.Amount),
		)

		// 执行USDT转账（使用实际到账金额，即扣除手续费后的金额）
		txHash, err := ps.transferUSDT(&order)
		if err != nil {
			// 转账失败，回滚事务
			tx.Rollback()
			logger.Logger.Error("USDT转账失败，已回滚",
				zap.String("order_id", orderID),
				zap.Error(err),
			)
			return fmt.Errorf("转账失败: %w", err)
		}

		// 更新订单交易哈希
		order.TxHash = txHash
		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("更新交易哈希失败: %w", err)
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("提交事务失败: %w", err)
		}

		// 发送提现审核通过通知
		SendOrderNotification(order.UserID, "withdraw", orderID, "approved", order.Amount, "")

		logger.Logger.Info("提现订单审核通过",
			zap.String("order_id", orderID),
			zap.Uint("auditor_id", auditorID),
		)
	} else {
		// 拒绝审核：只需更新订单状态
		order.Status = 3 // 已拒绝
		order.AuditAt = &now
		order.AuditorID = auditorID
		order.Remark = remark

		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("更新订单状态失败: %w", err)
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("提交事务失败: %w", err)
		}

		// 发送提现审核拒绝通知
		SendOrderNotification(order.UserID, "withdraw", orderID, "rejected", order.Amount, remark)

		logger.Logger.Info("提现订单审核拒绝",
			zap.String("order_id", orderID),
			zap.Uint("auditor_id", auditorID),
			zap.String("remark", remark),
		)
	}

	return nil
}

// transferUSDT 执行USDT转账
func (ps *PaymentService) transferUSDT(order *models.WithdrawOrder) (string, error) {
	if ps.transferService == nil || ps.hdWallet == nil {
		return "", errors.New("转账服务未初始化")
	}

	// 派生主钱包地址和私钥
	var fromAddr common.Address
	var fromAddrTron string
	var privateKey *ecdsa.PrivateKey
	var err error

	if order.ChainType == "erc20" {
		// 派生主钱包以太坊地址（account=0, index=0）
		fromAddr, privateKey, err = ps.hdWallet.DeriveMasterEthereumAddress()
		if err != nil {
			return "", fmt.Errorf("派生主钱包地址失败: %w", err)
		}
	} else if order.ChainType == "trc20" {
		// 派生主钱包波场地址（account=0, index=0）
		fromAddrTron, privateKey, err = ps.hdWallet.DeriveMasterTronAddress()
		if err != nil {
			return "", fmt.Errorf("派生主钱包地址失败: %w", err)
		}
	} else {
		return "", fmt.Errorf("不支持的链类型: %s", order.ChainType)
	}

	// 转换金额（USDT是6位小数）
	// 使用实际到账金额（扣除手续费后的金额）进行转账
	// 如果 actual_amount 为0，则使用 amount（兼容旧数据）
	transferAmount := order.ActualAmount
	if transferAmount == 0 {
		transferAmount = order.Amount
	}

	amountFloat := new(big.Float).SetFloat64(transferAmount)
	multiplier := new(big.Float).SetInt64(1000000) // USDT是6位小数
	amountFloat.Mul(amountFloat, multiplier)

	amountInt := new(big.Int)
	amountFloat.Int(amountInt)

	// 执行转账
	var txHash string
	if order.ChainType == "erc20" {
		toAddr := common.HexToAddress(order.ToAddress)
		txHash, err = ps.transferService.TransferERC20USDT(fromAddr, toAddr, amountInt, privateKey)
		if err != nil {
			return "", fmt.Errorf("ERC20转账失败: %w", err)
		}
	} else if order.ChainType == "trc20" {
		toAddr := order.ToAddress
		txHash, err = ps.transferService.TransferTRC20USDT(fromAddrTron, toAddr, amountInt, privateKey)
		if err != nil {
			return "", fmt.Errorf("TRC20转账失败: %w", err)
		}
	}

	logger.Logger.Info("USDT转账成功",
		zap.String("order_id", order.OrderID),
		zap.String("chain_type", order.ChainType),
		zap.String("to_address", order.ToAddress),
		zap.Float64("amount", order.Amount),
		zap.Float64("fee", order.Fee),
		zap.Float64("actual_amount", order.ActualAmount),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}
