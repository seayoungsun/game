package user

import (
	"context"

	"github.com/kaifa/game-platform/pkg/models"
)

// Repository 定义用户数据访问接口。
type Repository interface {
	// GetByID 根据ID获取用户
	GetByID(ctx context.Context, id uint) (*models.User, error)

	// GetByPhone 根据手机号获取用户
	GetByPhone(ctx context.Context, phone string) (*models.User, error)

	// Create 创建用户
	Create(ctx context.Context, user *models.User) error

	// Update 更新用户
	Update(ctx context.Context, user *models.User) error

	// CreateWallet 创建用户钱包
	CreateWallet(ctx context.Context, wallet *models.UserWallet) error

	// CreateLoginLog 创建登录日志
	CreateLoginLog(ctx context.Context, log *models.UserLogin) error

	// GetWallet 获取用户钱包
	GetWallet(ctx context.Context, userID uint) (*models.UserWallet, error)

	// UpdateBalance 更新用户余额
	UpdateBalance(ctx context.Context, userID uint, newBalance float64) error

	// BatchUpdateBalances 批量更新用户余额（使用事务）
	BatchUpdateBalances(ctx context.Context, balances map[uint]float64) error
}
