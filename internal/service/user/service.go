package user

import (
	"context"
	"errors"
	"fmt"

	userrepo "github.com/kaifa/game-platform/internal/repository/user"
	"github.com/kaifa/game-platform/pkg/models"
	"github.com/kaifa/game-platform/pkg/utils"
	"gorm.io/gorm"
)

// Service 定义用户业务服务接口
type Service interface {
	// Register 用户注册
	Register(ctx context.Context, req *RegisterRequest) (*models.User, string, error)

	// Login 用户登录
	Login(ctx context.Context, req *LoginRequest, ip string) (*models.User, string, error)

	// GetUserByID 根据ID获取用户
	GetUserByID(ctx context.Context, userID uint) (*models.User, error)

	// GetUserProfile 获取用户信息（包含钱包）
	GetUserProfile(ctx context.Context, userID uint) (map[string]interface{}, error)
}

type service struct {
	repo userrepo.Repository
}

// New 创建用户服务实例
func New(repo userrepo.Repository) Service {
	return &service{
		repo: repo,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" binding:"required"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 用户注册
func (s *service) Register(ctx context.Context, req *RegisterRequest) (*models.User, string, error) {
	// ✅ 通过 Repository 查询用户是否存在
	existingUser, err := s.repo.GetByPhone(ctx, req.Phone)
	if err == nil && existingUser != nil {
		return nil, "", errors.New("手机号已被注册")
	}
	// 如果错误不是 RecordNotFound，说明是其他数据库错误
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", fmt.Errorf("查询用户失败: %w", err)
	}

	// ✅ 业务逻辑：生成UID
	uid, err := utils.GenerateUID()
	if err != nil {
		return nil, "", fmt.Errorf("生成用户ID失败: %w", err)
	}

	// ✅ 业务逻辑：加密密码
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, "", fmt.Errorf("密码加密失败: %w", err)
	}

	// ✅ 创建用户对象
	user := models.User{
		UID:      uid,
		Phone:    req.Phone,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Balance:  0,
		Status:   1,
	}

	// ✅ 通过 Repository 创建用户
	if err := s.repo.Create(ctx, &user); err != nil {
		return nil, "", fmt.Errorf("创建用户失败: %w", err)
	}

	// ✅ 创建钱包
	wallet := models.UserWallet{
		UserID:   user.ID,
		Balance:  0,
		Frozen:   0,
		TotalIn:  0,
		TotalOut: 0,
	}
	if err := s.repo.CreateWallet(ctx, &wallet); err != nil {
		// 记录错误但不影响注册流程
		// 可以考虑使用日志记录
		fmt.Printf("创建钱包失败: %v\n", err)
	}

	// ✅ 业务逻辑：生成Token
	token, err := utils.GenerateToken(user.ID, user.UID, user.Phone)
	if err != nil {
		return nil, "", fmt.Errorf("生成Token失败: %w", err)
	}

	return &user, token, nil
}

// Login 用户登录
func (s *service) Login(ctx context.Context, req *LoginRequest, ip string) (*models.User, string, error) {
	// ✅ 通过 Repository 查找用户
	user, err := s.repo.GetByPhone(ctx, req.Phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("手机号或密码错误")
		}
		return nil, "", fmt.Errorf("查询用户失败: %w", err)
	}

	// ✅ 业务逻辑：检查状态
	if user.Status != 1 {
		return nil, "", errors.New("账号已被封禁")
	}

	// ✅ 业务逻辑：验证密码
	if err := utils.CheckPassword(user.Password, req.Password); err != nil {
		return nil, "", errors.New("手机号或密码错误")
	}

	// ✅ 通过 Repository 记录登录日志
	loginLog := models.UserLogin{
		UserID: user.ID,
		IP:     ip,
		Device: "", // 可以从Header获取
	}
	_ = s.repo.CreateLoginLog(ctx, &loginLog)

	// ✅ 业务逻辑：生成Token
	token, err := utils.GenerateToken(user.ID, user.UID, user.Phone)
	if err != nil {
		return nil, "", fmt.Errorf("生成Token失败: %w", err)
	}

	return user, token, nil
}

// GetUserByID 根据ID获取用户
func (s *service) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	// ✅ 通过 Repository 查询
	return s.repo.GetByID(ctx, userID)
}

// GetUserProfile 获取用户信息（包含钱包）
func (s *service) GetUserProfile(ctx context.Context, userID uint) (map[string]interface{}, error) {
	// ✅ 通过 Repository 查询用户
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// ✅ 通过 Repository 查询钱包
	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 如果钱包不存在，返回空钱包
	if wallet == nil {
		wallet = &models.UserWallet{
			UserID: userID,
		}
	}

	return map[string]interface{}{
		"user":   user,
		"wallet": wallet,
	}, nil
}
