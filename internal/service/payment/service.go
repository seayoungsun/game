package payment

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/kaifa/game-platform/internal/logger"
	paymentrepo "github.com/kaifa/game-platform/internal/repository/payment"
	userrepo "github.com/kaifa/game-platform/internal/repository/user"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/kaifa/game-platform/pkg/services"
	"go.uber.org/zap"
)

// Service 定义支付业务服务接口
type Service interface {
	// CreateRechargeOrder 创建充值订单
	CreateRechargeOrder(ctx context.Context, userID uint, amount float64, chainType string) (*models.RechargeOrder, error)

	// GetRechargeOrder 获取充值订单
	GetRechargeOrder(ctx context.Context, orderID string, userID uint) (*models.RechargeOrder, error)

	// GetUserRechargeOrders 获取用户的充值订单列表
	GetUserRechargeOrders(ctx context.Context, userID uint, page, pageSize int) ([]models.RechargeOrder, int64, error)

	// CheckTransaction 检查交易状态
	CheckTransaction(ctx context.Context, orderID string) error

	// CreateWithdrawOrder 创建提现订单
	CreateWithdrawOrder(ctx context.Context, userID uint, amount float64, chainType string, toAddress string) (*models.WithdrawOrder, error)

	// GetWithdrawOrder 获取提现订单
	GetWithdrawOrder(ctx context.Context, orderID string, userID uint) (*models.WithdrawOrder, error)

	// GetUserWithdrawOrders 获取用户的提现订单列表
	GetUserWithdrawOrders(ctx context.Context, userID uint, page, pageSize int) ([]models.WithdrawOrder, int64, error)

	// AuditWithdrawOrder 审核提现订单
	AuditWithdrawOrder(ctx context.Context, auditorID uint, orderID string, approve bool, remark string) error

	// StartTransactionMonitor 启动交易监控
	StartTransactionMonitor()
}

type service struct {
	rechargeOrderRepo paymentrepo.RechargeOrderRepository
	withdrawOrderRepo paymentrepo.WithdrawOrderRepository
	transactionRepo   paymentrepo.TransactionRepository
	depositAddrRepo   paymentrepo.DepositAddressRepository
	userRepo          userrepo.Repository

	// 外部服务依赖
	hdWallet        *services.HDWallet
	transferService *services.USDTTransferService

	// API 配置
	tronAPIURL      string
	etherscanAPIURL string
	etherscanAPIKey string
}

// New 创建支付服务实例
func New(
	rechargeOrderRepo paymentrepo.RechargeOrderRepository,
	withdrawOrderRepo paymentrepo.WithdrawOrderRepository,
	transactionRepo paymentrepo.TransactionRepository,
	depositAddrRepo paymentrepo.DepositAddressRepository,
	userRepo userrepo.Repository,
	hdWallet *services.HDWallet,
	transferService *services.USDTTransferService,
	etherscanAPIKey string,
) Service {
	return &service{
		rechargeOrderRepo: rechargeOrderRepo,
		withdrawOrderRepo: withdrawOrderRepo,
		transactionRepo:   transactionRepo,
		depositAddrRepo:   depositAddrRepo,
		userRepo:          userRepo,
		hdWallet:          hdWallet,
		transferService:   transferService,
		tronAPIURL:        "https://api.trongrid.io",
		etherscanAPIURL:   "https://api.etherscan.io/api",
		etherscanAPIKey:   etherscanAPIKey,
	}
}

// CreateRechargeOrder 创建充值订单
func (s *service) CreateRechargeOrder(ctx context.Context, userID uint, amount float64, chainType string) (*models.RechargeOrder, error) {
	// ✅ 业务逻辑：参数验证
	if amount <= 0 {
		return nil, errors.New("充值金额必须大于0")
	}

	// TODO: 从系统配置获取限额
	minAmount := 10.0
	maxAmount := 10000.0

	if amount < minAmount {
		return nil, fmt.Errorf("充值金额不能小于%.2f USDT", minAmount)
	}
	if amount > maxAmount {
		return nil, fmt.Errorf("充值金额不能大于%.2f USDT", maxAmount)
	}

	if chainType != "trc20" && chainType != "erc20" {
		return nil, errors.New("链类型必须是trc20或erc20")
	}

	// ✅ 业务逻辑：生成订单号
	orderID := fmt.Sprintf("R%s", strings.ToUpper(uuid.New().String()[:15]))

	// ✅ 业务逻辑：生成充值地址
	depositAddr, err := s.getDepositAddress(ctx, userID, chainType)
	if err != nil {
		return nil, fmt.Errorf("获取充值地址失败: %w", err)
	}

	// ✅ 业务逻辑：计算过期时间（30分钟）
	now := time.Now().Unix()
	expireAt := now + 30*60

	channel := fmt.Sprintf("usdt_%s", chainType)
	requiredConf := 12
	if chainType == "trc20" {
		requiredConf = 20
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

	// ✅ 通过 Repository 创建订单
	if err := s.rechargeOrderRepo.Create(ctx, order); err != nil {
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
func (s *service) GetRechargeOrder(ctx context.Context, orderID string, userID uint) (*models.RechargeOrder, error) {
	// ✅ 通过 Repository 查询
	if userID == 0 {
		// 管理员查询所有订单
		return s.rechargeOrderRepo.GetByOrderID(ctx, orderID)
	}
	return s.rechargeOrderRepo.GetByOrderIDAndUser(ctx, orderID, userID)
}

// GetUserRechargeOrders 获取用户的充值订单列表
func (s *service) GetUserRechargeOrders(ctx context.Context, userID uint, page, pageSize int) ([]models.RechargeOrder, int64, error) {
	// ✅ 业务逻辑：参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// ✅ 通过 Repository 查询
	return s.rechargeOrderRepo.ListByUser(ctx, userID, offset, pageSize)
}

// CheckTransaction 检查交易状态
func (s *service) CheckTransaction(ctx context.Context, orderID string) error {
	// ✅ 通过 Repository 获取订单
	order, err := s.rechargeOrderRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return errors.New("订单不存在")
	}

	// 如果已经支付，不需要再检查
	if order.Status == 2 {
		return nil
	}

	// 如果订单已过期
	if time.Now().Unix() > order.ExpireAt {
		order.Status = 3 // 已取消
		s.rechargeOrderRepo.Update(ctx, order)
		return errors.New("订单已过期")
	}

	// 根据链类型检查交易
	var txHash string
	var confirmCount int

	if order.ChainType == "trc20" {
		txHash, confirmCount, err = s.checkTRC20Transaction(order.DepositAddr, order.Amount)
	} else if order.ChainType == "erc20" {
		txHash, confirmCount, err = s.checkERC20Transaction(order.DepositAddr, order.Amount)
	} else {
		return errors.New("不支持的链类型")
	}

	if err != nil {
		return err
	}

	// 如果找到交易，更新订单
	if txHash != "" {
		order.TxHash = txHash
		order.ChannelID = txHash
		order.ConfirmCount = confirmCount

		// 如果确认次数足够，完成充值
		if confirmCount >= order.RequiredConf {
			return s.completeRecharge(ctx, order)
		}

		s.rechargeOrderRepo.Update(ctx, order)
	}

	return nil
}

// CreateWithdrawOrder 创建提现订单
func (s *service) CreateWithdrawOrder(ctx context.Context, userID uint, amount float64, chainType string, toAddress string) (*models.WithdrawOrder, error) {
	// ✅ 业务逻辑：参数验证
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

	// ✅ 通过 Repository 检查用户余额
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	// TODO: 从系统配置获取限额
	minWithdraw := 50.0
	maxWithdraw := 5000.0

	if amount < minWithdraw {
		return nil, fmt.Errorf("提现金额不能小于%.2f USDT", minWithdraw)
	}
	if amount > maxWithdraw {
		return nil, fmt.Errorf("提现金额不能大于%.2f USDT", maxWithdraw)
	}

	// ✅ 业务逻辑：计算手续费
	feeRate := 0.001 // TODO: 从系统配置获取
	fee := amount * feeRate
	if fee < 0.01 {
		fee = 0.01
	} else {
		fee = math.Ceil(fee*100) / 100
	}

	actualAmount := amount - fee

	// 检查余额是否足够
	if user.Balance < amount {
		return nil, fmt.Errorf("余额不足，需要%.2f USDT", amount)
	}

	// ✅ 业务逻辑：生成订单号
	orderID := fmt.Sprintf("W%s", strings.ToUpper(uuid.New().String()[:15]))
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

	// ✅ 通过 Repository 创建订单
	if err := s.withdrawOrderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("创建提现订单失败: %w", err)
	}

	logger.Logger.Info("创建提现订单",
		zap.String("order_id", orderID),
		zap.Uint("user_id", userID),
		zap.Float64("amount", amount),
		zap.Float64("fee", fee),
		zap.String("chain_type", chainType),
	)

	return order, nil
}

// GetWithdrawOrder 获取提现订单
func (s *service) GetWithdrawOrder(ctx context.Context, orderID string, userID uint) (*models.WithdrawOrder, error) {
	// ✅ 通过 Repository 查询
	if userID == 0 {
		// 管理员查询所有订单
		return s.withdrawOrderRepo.GetByOrderID(ctx, orderID)
	}
	return s.withdrawOrderRepo.GetByOrderIDAndUser(ctx, orderID, userID)
}

// GetUserWithdrawOrders 获取用户的提现订单列表
func (s *service) GetUserWithdrawOrders(ctx context.Context, userID uint, page, pageSize int) ([]models.WithdrawOrder, int64, error) {
	// ✅ 业务逻辑：参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// ✅ 通过 Repository 查询
	return s.withdrawOrderRepo.ListByUser(ctx, userID, offset, pageSize)
}

// AuditWithdrawOrder 审核提现订单
func (s *service) AuditWithdrawOrder(ctx context.Context, auditorID uint, orderID string, approve bool, remark string) error {
	// ✅ 通过 Repository 获取订单
	order, err := s.withdrawOrderRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return errors.New("订单不存在")
	}

	// 如果已经审核过，不允许重复审核
	if order.Status != 1 {
		return errors.New("订单已审核，无法重复审核")
	}

	now := time.Now().Unix()

	if approve {
		// 通过审核
		return s.approveWithdraw(ctx, order, auditorID, now, remark)
	} else {
		// 拒绝审核
		return s.rejectWithdraw(ctx, order, auditorID, now, remark)
	}
}

// StartTransactionMonitor 启动交易监控
func (s *service) StartTransactionMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			s.checkPendingOrders()
		}
	}()
}

// ==================== 私有方法 ====================

// getDepositAddress 获取充值地址
func (s *service) getDepositAddress(ctx context.Context, userID uint, chainType string) (string, error) {
	if s.hdWallet == nil {
		return "", errors.New("HD钱包未初始化")
	}

	// ✅ 通过 Repository 查询是否已有地址
	existingAddr, err := s.depositAddrRepo.GetByUserAndChain(ctx, userID, chainType)
	if err == nil && existingAddr != nil && existingAddr.Address != "" {
		return existingAddr.Address, nil
	}

	// 使用HD钱包派生地址
	var address string
	var path string

	if chainType == "trc20" {
		address, err = s.hdWallet.DeriveTronAddressByUserID(userID)
		if err != nil {
			return "", fmt.Errorf("派生波场地址失败: %w", err)
		}
		path = services.GetTronPath(0, uint32(userID))
	} else if chainType == "erc20" {
		ethAddr, err := s.hdWallet.DeriveEthereumAddressByUserID(userID)
		if err != nil {
			return "", fmt.Errorf("派生以太坊地址失败: %w", err)
		}
		address = ethAddr.Hex()
		path = services.GetEthereumPath(0, uint32(userID))
	} else {
		return "", fmt.Errorf("不支持的链类型: %s", chainType)
	}

	logger.Logger.Info("使用HD钱包派生地址",
		zap.Uint("user_id", userID),
		zap.String("chain_type", chainType),
		zap.String("path", path),
		zap.String("address", address),
	)

	// ✅ 通过 Repository 保存地址
	newAddr := &models.UserDepositAddress{
		UserID:    userID,
		ChainType: chainType,
		Address:   address,
	}

	// 再次检查（防止并发）
	existingAddr, err = s.depositAddrRepo.GetByUserAndChain(ctx, userID, chainType)
	if err == nil && existingAddr != nil {
		return existingAddr.Address, nil
	}

	if err := s.depositAddrRepo.Create(ctx, newAddr); err != nil {
		// 如果是唯一键冲突，再次查询返回
		if strings.Contains(err.Error(), "Duplicate") {
			existingAddr, _ = s.depositAddrRepo.GetByUserAndChain(ctx, userID, chainType)
			if existingAddr != nil {
				return existingAddr.Address, nil
			}
		}
		return "", fmt.Errorf("保存充值地址失败: %w", err)
	}

	return address, nil
}

// completeRecharge 完成充值（使用事务）
func (s *service) completeRecharge(ctx context.Context, order *models.RechargeOrder) error {
	// 检查订单状态
	if order.Status == 2 {
		return errors.New("订单已处理")
	}

	now := time.Now().Unix()

	// 更新订单状态
	order.Status = 2
	order.PaidAt = &now
	if err := s.rechargeOrderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// ✅ 通过 Repository 获取用户
	user, err := s.userRepo.GetByID(ctx, order.UserID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// ✅ 通过 Repository 更新余额
	newBalance := user.Balance + order.Amount
	if err := s.userRepo.UpdateBalance(ctx, order.UserID, newBalance); err != nil {
		return fmt.Errorf("更新用户余额失败: %w", err)
	}

	// ✅ 通过 Repository 创建交易记录
	transaction := &models.Transaction{
		OrderID:   order.OrderID,
		UserID:    order.UserID,
		Type:      "recharge",
		Amount:    order.Amount,
		Status:    2,
		Channel:   order.Channel,
		ChannelID: order.TxHash,
		Remark:    fmt.Sprintf("USDT充值 - %s", order.ChainType),
	}
	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return fmt.Errorf("创建交易记录失败: %w", err)
	}

	// 发送充值成功通知
	services.SendOrderNotification(order.UserID, "recharge", order.OrderID, "paid", order.Amount, "")

	logger.Logger.Info("充值完成",
		zap.String("order_id", order.OrderID),
		zap.Uint("user_id", order.UserID),
		zap.Float64("amount", order.Amount),
		zap.String("tx_hash", order.TxHash),
	)

	return nil
}

// approveWithdraw 通过提现审核
func (s *service) approveWithdraw(ctx context.Context, order *models.WithdrawOrder, auditorID uint, now int64, remark string) error {
	// ✅ 通过 Repository 获取用户
	user, err := s.userRepo.GetByID(ctx, order.UserID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 检查余额是否足够
	if user.Balance < order.Amount {
		return errors.New("用户余额不足")
	}

	// ✅ 通过 Repository 更新余额
	newBalance := user.Balance - order.Amount
	if err := s.userRepo.UpdateBalance(ctx, order.UserID, newBalance); err != nil {
		return fmt.Errorf("扣除余额失败: %w", err)
	}

	// 更新订单状态为已通过
	order.Status = 2
	order.AuditAt = &now
	order.AuditorID = auditorID
	order.Remark = remark

	// 执行USDT转账
	txHash, err := s.transferUSDT(order)
	if err != nil {
		// 转账失败，回滚余额
		s.userRepo.UpdateBalance(ctx, order.UserID, user.Balance)
		return fmt.Errorf("转账失败: %w", err)
	}

	order.TxHash = txHash
	if err := s.withdrawOrderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 发送通知
	services.SendOrderNotification(order.UserID, "withdraw", order.OrderID, "approved", order.Amount, "")

	logger.Logger.Info("提现订单审核通过",
		zap.String("order_id", order.OrderID),
		zap.Uint("auditor_id", auditorID),
		zap.String("tx_hash", txHash),
	)

	return nil
}

// rejectWithdraw 拒绝提现审核
func (s *service) rejectWithdraw(ctx context.Context, order *models.WithdrawOrder, auditorID uint, now int64, remark string) error {
	order.Status = 3
	order.AuditAt = &now
	order.AuditorID = auditorID
	order.Remark = remark

	if err := s.withdrawOrderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 发送通知
	services.SendOrderNotification(order.UserID, "withdraw", order.OrderID, "rejected", order.Amount, remark)

	logger.Logger.Info("提现订单审核拒绝",
		zap.String("order_id", order.OrderID),
		zap.Uint("auditor_id", auditorID),
		zap.String("remark", remark),
	)

	return nil
}

// transferUSDT 执行USDT转账
func (s *service) transferUSDT(order *models.WithdrawOrder) (string, error) {
	if s.transferService == nil || s.hdWallet == nil {
		return "", errors.New("转账服务未初始化")
	}

	// 派生主钱包地址和私钥
	var fromAddr common.Address
	var fromAddrTron string
	var privateKey *ecdsa.PrivateKey
	var err error

	if order.ChainType == "erc20" {
		fromAddr, privateKey, err = s.hdWallet.DeriveMasterEthereumAddress()
		if err != nil {
			return "", fmt.Errorf("派生主钱包地址失败: %w", err)
		}
	} else if order.ChainType == "trc20" {
		fromAddrTron, privateKey, err = s.hdWallet.DeriveMasterTronAddress()
		if err != nil {
			return "", fmt.Errorf("派生主钱包地址失败: %w", err)
		}
	} else {
		return "", fmt.Errorf("不支持的链类型: %s", order.ChainType)
	}

	// 转换金额（USDT是6位小数）
	transferAmount := order.ActualAmount
	if transferAmount == 0 {
		transferAmount = order.Amount
	}

	amountFloat := new(big.Float).SetFloat64(transferAmount)
	multiplier := new(big.Float).SetInt64(1000000)
	amountFloat.Mul(amountFloat, multiplier)

	amountInt := new(big.Int)
	amountFloat.Int(amountInt)

	// 执行转账
	var txHash string
	if order.ChainType == "erc20" {
		toAddr := common.HexToAddress(order.ToAddress)
		txHash, err = s.transferService.TransferERC20USDT(fromAddr, toAddr, amountInt, privateKey)
	} else if order.ChainType == "trc20" {
		txHash, err = s.transferService.TransferTRC20USDT(fromAddrTron, order.ToAddress, amountInt, privateKey)
	}

	if err != nil {
		return "", err
	}

	logger.Logger.Info("USDT转账成功",
		zap.String("order_id", order.OrderID),
		zap.String("chain_type", order.ChainType),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}

// checkPendingOrders 检查待支付的订单
func (s *service) checkPendingOrders() {
	ctx := context.Background()

	// ✅ 通过 Repository 查询待支付订单
	orders, err := s.rechargeOrderRepo.ListPending(ctx, time.Now().Unix())
	if err != nil {
		return
	}

	for _, order := range orders {
		go func(o models.RechargeOrder) {
			if err := s.CheckTransaction(ctx, o.OrderID); err != nil {
				logger.Logger.Debug("检查交易失败",
					zap.String("order_id", o.OrderID),
					zap.Error(err),
				)
			}
		}(order)
	}
}

// checkTRC20Transaction 检查TRC20交易
func (s *service) checkTRC20Transaction(depositAddr string, amount float64) (string, int, error) {
	// TODO: 实现 TRC20 交易检查逻辑（调用 TronGrid API）
	return "", 0, errors.New("未找到匹配的交易")
}

// checkERC20Transaction 检查ERC20交易
func (s *service) checkERC20Transaction(depositAddr string, amount float64) (string, int, error) {
	// TODO: 实现 ERC20 交易检查逻辑（调用 Etherscan API）
	return "", 0, errors.New("未找到匹配的交易")
}
