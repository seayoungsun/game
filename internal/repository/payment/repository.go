package payment

import (
	"context"

	"github.com/kaifa/game-platform/pkg/models"
)

// RechargeOrderRepository 充值订单数据访问接口
type RechargeOrderRepository interface {
	// Create 创建充值订单
	Create(ctx context.Context, order *models.RechargeOrder) error

	// GetByOrderID 根据订单号获取充值订单
	GetByOrderID(ctx context.Context, orderID string) (*models.RechargeOrder, error)

	// GetByOrderIDAndUser 根据订单号和用户ID获取充值订单
	GetByOrderIDAndUser(ctx context.Context, orderID string, userID uint) (*models.RechargeOrder, error)

	// Update 更新充值订单
	Update(ctx context.Context, order *models.RechargeOrder) error

	// ListByUser 获取用户的充值订单列表
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.RechargeOrder, int64, error)

	// ListPending 获取待支付的订单
	ListPending(ctx context.Context, minExpireAt int64) ([]models.RechargeOrder, error)
}

// WithdrawOrderRepository 提现订单数据访问接口
type WithdrawOrderRepository interface {
	// Create 创建提现订单
	Create(ctx context.Context, order *models.WithdrawOrder) error

	// GetByOrderID 根据订单号获取提现订单
	GetByOrderID(ctx context.Context, orderID string) (*models.WithdrawOrder, error)

	// GetByOrderIDAndUser 根据订单号和用户ID获取提现订单
	GetByOrderIDAndUser(ctx context.Context, orderID string, userID uint) (*models.WithdrawOrder, error)

	// Update 更新提现订单
	Update(ctx context.Context, order *models.WithdrawOrder) error

	// ListByUser 获取用户的提现订单列表
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.WithdrawOrder, int64, error)
}

// TransactionRepository 交易记录数据访问接口
type TransactionRepository interface {
	// Create 创建交易记录
	Create(ctx context.Context, transaction *models.Transaction) error

	// GetByOrderID 根据订单号获取交易记录
	GetByOrderID(ctx context.Context, orderID string) (*models.Transaction, error)

	// ListByUser 获取用户的交易记录
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.Transaction, int64, error)
}

// DepositAddressRepository 用户充值地址数据访问接口
type DepositAddressRepository interface {
	// Create 创建用户充值地址
	Create(ctx context.Context, address *models.UserDepositAddress) error

	// GetByUserAndChain 根据用户ID和链类型获取充值地址
	GetByUserAndChain(ctx context.Context, userID uint, chainType string) (*models.UserDepositAddress, error)

	// Update 更新充值地址
	Update(ctx context.Context, address *models.UserDepositAddress) error
}
