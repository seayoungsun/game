package mysql

import (
	"context"

	userrepo "github.com/kaifa/game-platform/internal/repository/user"
	"github.com/kaifa/game-platform/pkg/models"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) userrepo.Repository {
	return &UserRepository{db: db}
}

// GetByID 根据ID获取用户
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone 根据手机号获取用户
func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// CreateWallet 创建用户钱包
func (r *UserRepository) CreateWallet(ctx context.Context, wallet *models.UserWallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

// CreateLoginLog 创建登录日志
func (r *UserRepository) CreateLoginLog(ctx context.Context, log *models.UserLogin) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetWallet 获取用户钱包
func (r *UserRepository) GetWallet(ctx context.Context, userID uint) (*models.UserWallet, error) {
	var wallet models.UserWallet
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

// UpdateBalance 更新用户余额
func (r *UserRepository) UpdateBalance(ctx context.Context, userID uint, newBalance float64) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("balance", newBalance).Error
}

// BatchUpdateBalances 批量更新用户余额（使用事务）
func (r *UserRepository) BatchUpdateBalances(ctx context.Context, balances map[uint]float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for userID, newBalance := range balances {
			if err := tx.Model(&models.User{}).
				Where("id = ?", userID).
				Update("balance", newBalance).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

var _ userrepo.Repository = (*UserRepository)(nil)
