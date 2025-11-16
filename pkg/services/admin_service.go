package services

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
	"go.uber.org/zap"

	"github.com/kaifa/game-platform/internal/logger"
)

// AdminService 管理员服务
type AdminService struct{}

var adminServiceInstance *AdminService

// NewAdminService 创建管理员服务
func NewAdminService() *AdminService {
	if adminServiceInstance == nil {
		adminServiceInstance = &AdminService{}
	}
	return adminServiceInstance
}

// Login 管理员登录
func (s *AdminService) Login(username, password, ip string) (*models.Admin, error) {
	var admin models.Admin

	// 查找管理员
	if err := database.DB.Where("username = ? AND status = 1", username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, fmt.Errorf("查询管理员失败: %w", err)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 更新最后登录时间和IP
	now := time.Now().Unix()
	admin.LastLoginAt = &now
	admin.LastLoginIP = ip
	if err := database.DB.Save(&admin).Error; err != nil {
		logger.Logger.Warn("更新最后登录信息失败", zap.Error(err))
	}

	return &admin, nil
}

// GetAdminByID 根据ID获取管理员
func (s *AdminService) GetAdminByID(id uint) (*models.Admin, error) {
	var admin models.Admin
	if err := database.DB.Preload("Roles").Where("id = ?", id).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

// GetAdminWithPermissions 获取管理员及其所有权限
func (s *AdminService) GetAdminWithPermissions(adminID uint) ([]string, error) {
	var permissionCodes []string

	err := database.DB.Table("admin_permissions").
		Joins("INNER JOIN role_permission_relations ON admin_permissions.id = role_permission_relations.permission_id").
		Joins("INNER JOIN admin_roles ON role_permission_relations.role_id = admin_roles.id").
		Joins("INNER JOIN admin_role_relations ON admin_roles.id = admin_role_relations.role_id").
		Where("admin_role_relations.admin_id = ? AND admin_roles.status = 1", adminID).
		Select("DISTINCT admin_permissions.permission_code").
		Pluck("permission_code", &permissionCodes).Error

	if err != nil {
		return nil, fmt.Errorf("获取权限列表失败: %w", err)
	}

	return permissionCodes, nil
}

// GetAdminRoles 获取管理员的角色列表
func (s *AdminService) GetAdminRoles(adminID uint) ([]string, error) {
	var roleCodes []string

	err := database.DB.Table("admin_roles").
		Joins("INNER JOIN admin_role_relations ON admin_roles.id = admin_role_relations.role_id").
		Where("admin_role_relations.admin_id = ? AND admin_roles.status = 1", adminID).
		Pluck("admin_roles.role_code", &roleCodes).Error

	if err != nil {
		return nil, fmt.Errorf("获取角色列表失败: %w", err)
	}

	return roleCodes, nil
}

// HasPermission 检查管理员是否有指定权限
func (s *AdminService) HasPermission(adminID uint, permissionCode string) (bool, error) {
	var count int64

	err := database.DB.Table("admin_permissions").
		Joins("INNER JOIN role_permission_relations ON admin_permissions.id = role_permission_relations.permission_id").
		Joins("INNER JOIN admin_roles ON role_permission_relations.role_id = admin_roles.id").
		Joins("INNER JOIN admin_role_relations ON admin_roles.id = admin_role_relations.role_id").
		Where("admin_role_relations.admin_id = ? AND admin_permissions.permission_code = ? AND admin_roles.status = 1", adminID, permissionCode).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("检查权限失败: %w", err)
	}

	return count > 0, nil
}

// HasRole 检查管理员是否有指定角色
func (s *AdminService) HasRole(adminID uint, roleCode string) (bool, error) {
	var count int64

	err := database.DB.Table("admin_roles").
		Joins("INNER JOIN admin_role_relations ON admin_roles.id = admin_role_relations.role_id").
		Where("admin_role_relations.admin_id = ? AND admin_roles.role_code = ? AND admin_roles.status = 1", adminID, roleCode).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("检查角色失败: %w", err)
	}

	return count > 0, nil
}

// CreateAdmin 创建管理员
func (s *AdminService) CreateAdmin(username, password, nickname, email string) (*models.Admin, error) {
	// 检查用户名是否已存在
	var count int64
	if err := database.DB.Model(&models.Admin{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("加密密码失败: %w", err)
	}

	// 创建管理员
	admin := &models.Admin{
		Username: username,
		Password: string(hashedPassword),
		Nickname: nickname,
		Email:    email,
		Status:   1,
	}

	if err := database.DB.Create(admin).Error; err != nil {
		return nil, fmt.Errorf("创建管理员失败: %w", err)
	}

	return admin, nil
}

// UpdateAdmin 更新管理员信息
func (s *AdminService) UpdateAdmin(id uint, nickname, email string) error {
	admin := &models.Admin{
		Nickname: nickname,
		Email:    email,
	}
	return database.DB.Model(&models.Admin{}).Where("id = ?", id).Updates(admin).Error
}

// UpdateAdminPassword 更新管理员密码
func (s *AdminService) UpdateAdminPassword(id uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("加密密码失败: %w", err)
	}

	return database.DB.Model(&models.Admin{}).Where("id = ?", id).Update("password", string(hashedPassword)).Error
}

// AssignRolesToAdmin 为管理员分配角色
func (s *AdminService) AssignRolesToAdmin(adminID uint, roleIDs []uint) error {
	// 使用事务
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 先删除现有角色关联
		if err := tx.Where("admin_id = ?", adminID).Delete(&models.AdminRoleRelation{}).Error; err != nil {
			return err
		}

		// 创建新的角色关联
		now := time.Now().Unix()
		for _, roleID := range roleIDs {
			relation := &models.AdminRoleRelation{
				AdminID:   adminID,
				RoleID:    roleID,
				CreatedAt: now,
			}
			if err := tx.Create(relation).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetAllRoles 获取所有角色
func (s *AdminService) GetAllRoles() ([]models.AdminRole, error) {
	var roles []models.AdminRole
	if err := database.DB.Where("status = 1").Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetAllPermissions 获取所有权限
func (s *AdminService) GetAllPermissions() ([]models.AdminPermission, error) {
	var permissions []models.AdminPermission
	if err := database.DB.Order("sort_order ASC, created_at ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}
