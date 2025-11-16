package mysql

import (
	"context"

	paymentrepo "github.com/kaifa/game-platform/internal/repository/payment"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

// ==================== RechargeOrderRepository ====================

type RechargeOrderRepository struct {
	db *gorm.DB
}

func NewRechargeOrderRepository(db *gorm.DB) paymentrepo.RechargeOrderRepository {
	return &RechargeOrderRepository{db: db}
}

func (r *RechargeOrderRepository) Create(ctx context.Context, order *models.RechargeOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *RechargeOrderRepository) GetByOrderID(ctx context.Context, orderID string) (*models.RechargeOrder, error) {
	var order models.RechargeOrder
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *RechargeOrderRepository) GetByOrderIDAndUser(ctx context.Context, orderID string, userID uint) (*models.RechargeOrder, error) {
	var order models.RechargeOrder
	if err := r.db.WithContext(ctx).Where("order_id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *RechargeOrderRepository) Update(ctx context.Context, order *models.RechargeOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *RechargeOrderRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.RechargeOrder, int64, error) {
	var orders []models.RechargeOrder
	var total int64

	query := r.db.WithContext(ctx).Model(&models.RechargeOrder{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *RechargeOrderRepository) ListPending(ctx context.Context, minExpireAt int64) ([]models.RechargeOrder, error) {
	var orders []models.RechargeOrder
	if err := r.db.WithContext(ctx).Where("status = ? AND expire_at > ?", 1, minExpireAt).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

var _ paymentrepo.RechargeOrderRepository = (*RechargeOrderRepository)(nil)

// ==================== WithdrawOrderRepository ====================

type WithdrawOrderRepository struct {
	db *gorm.DB
}

func NewWithdrawOrderRepository(db *gorm.DB) paymentrepo.WithdrawOrderRepository {
	return &WithdrawOrderRepository{db: db}
}

func (r *WithdrawOrderRepository) Create(ctx context.Context, order *models.WithdrawOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *WithdrawOrderRepository) GetByOrderID(ctx context.Context, orderID string) (*models.WithdrawOrder, error) {
	var order models.WithdrawOrder
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *WithdrawOrderRepository) GetByOrderIDAndUser(ctx context.Context, orderID string, userID uint) (*models.WithdrawOrder, error) {
	var order models.WithdrawOrder
	if err := r.db.WithContext(ctx).Where("order_id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *WithdrawOrderRepository) Update(ctx context.Context, order *models.WithdrawOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *WithdrawOrderRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.WithdrawOrder, int64, error) {
	var orders []models.WithdrawOrder
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WithdrawOrder{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

var _ paymentrepo.WithdrawOrderRepository = (*WithdrawOrderRepository)(nil)

// ==================== TransactionRepository ====================

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) paymentrepo.TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

func (r *TransactionRepository) GetByOrderID(ctx context.Context, orderID string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&transaction).Error; err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

var _ paymentrepo.TransactionRepository = (*TransactionRepository)(nil)

// ==================== DepositAddressRepository ====================

type DepositAddressRepository struct {
	db *gorm.DB
}

func NewDepositAddressRepository(db *gorm.DB) paymentrepo.DepositAddressRepository {
	return &DepositAddressRepository{db: db}
}

func (r *DepositAddressRepository) Create(ctx context.Context, address *models.UserDepositAddress) error {
	return r.db.WithContext(ctx).Create(address).Error
}

func (r *DepositAddressRepository) GetByUserAndChain(ctx context.Context, userID uint, chainType string) (*models.UserDepositAddress, error) {
	var address models.UserDepositAddress
	if err := r.db.WithContext(ctx).Where("user_id = ? AND chain_type = ?", userID, chainType).First(&address).Error; err != nil {
		return nil, err
	}
	return &address, nil
}

func (r *DepositAddressRepository) Update(ctx context.Context, address *models.UserDepositAddress) error {
	return r.db.WithContext(ctx).Save(address).Error
}

var _ paymentrepo.DepositAddressRepository = (*DepositAddressRepository)(nil)
