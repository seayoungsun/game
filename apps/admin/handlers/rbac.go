package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/pkg/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// hashPassword 加密密码
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// GetRoles 获取角色列表
func GetRoles(c *gin.Context) {
	var roles []models.AdminRole
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := database.DB.Model(&models.AdminRole{})

	// 搜索
	if search := c.Query("search"); search != "" {
		query = query.Where("role_name LIKE ? OR role_code LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	query.Count(&total)

	// 获取列表
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取角色列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  roles,
			"total": total,
		},
	})
}

// GetRole 获取角色详情
func GetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的角色ID",
		})
		return
	}

	var role models.AdminRole
	if err := database.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "角色不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取角色详情失败: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": role,
	})
}

// CreateRole 创建角色
func CreateRole(c *gin.Context) {
	var req struct {
		RoleName        string   `json:"role_name" binding:"required"`
		RoleCode        string   `json:"role_code" binding:"required"`
		Description     string   `json:"description"`
		PermissionCodes []string `json:"permission_codes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 检查角色代码是否已存在
	var existRole models.AdminRole
	if err := database.DB.Where("role_code = ?", req.RoleCode).First(&existRole).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "角色代码已存在",
		})
		return
	}

	// 创建角色
	now := time.Now().Unix()
	role := models.AdminRole{
		RoleName:    req.RoleName,
		RoleCode:    req.RoleCode,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := database.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建角色失败: " + err.Error(),
		})
		return
	}

	// 分配权限
	if len(req.PermissionCodes) > 0 {
		var permissions []models.AdminPermission
		database.DB.Where("permission_code IN ?", req.PermissionCodes).Find(&permissions)

		now := time.Now().Unix()
		for _, perm := range permissions {
			database.DB.Create(&models.RolePermissionRelation{
				RoleID:       role.ID,
				PermissionID: perm.ID,
				CreatedAt:    now,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    role,
	})
}

// UpdateRole 更新角色
func UpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的角色ID",
		})
		return
	}

	var req struct {
		RoleName        string   `json:"role_name" binding:"required"`
		Description     string   `json:"description"`
		PermissionCodes []string `json:"permission_codes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var role models.AdminRole
	if err := database.DB.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "角色不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取角色失败: " + err.Error(),
			})
		}
		return
	}

	// 更新角色信息
	role.RoleName = req.RoleName
	role.Description = req.Description
	role.UpdatedAt = time.Now().Unix()

	if err := database.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新角色失败: " + err.Error(),
		})
		return
	}

	// 更新权限关联
	database.DB.Where("role_id = ?", role.ID).Delete(&models.RolePermissionRelation{})
	if len(req.PermissionCodes) > 0 {
		var permissions []models.AdminPermission
		database.DB.Where("permission_code IN ?", req.PermissionCodes).Find(&permissions)

		now := time.Now().Unix()
		for _, perm := range permissions {
			database.DB.Create(&models.RolePermissionRelation{
				RoleID:       role.ID,
				PermissionID: perm.ID,
				CreatedAt:    now,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    role,
	})
}

// DeleteRole 删除角色
func DeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的角色ID",
		})
		return
	}

	var role models.AdminRole
	if err := database.DB.First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "角色不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取角色失败: " + err.Error(),
			})
		}
		return
	}

	// 检查是否有管理员使用此角色
	var count int64
	database.DB.Model(&models.AdminRoleRelation{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "该角色正在被使用，无法删除",
		})
		return
	}

	// 删除权限关联
	database.DB.Where("role_id = ?", id).Delete(&models.RolePermissionRelation{})
	// 删除角色
	database.DB.Delete(&role)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// GetAllPermissions 获取所有权限列表
func GetAllPermissions(c *gin.Context) {
	var permissions []models.AdminPermission
	if err := database.DB.Order("id ASC").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取权限列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": permissions,
	})
}

// GetAdmins 获取管理员列表
func GetAdmins(c *gin.Context) {
	var admins []models.Admin
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := database.DB.Model(&models.Admin{})

	// 搜索
	if search := c.Query("search"); search != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	query.Count(&total)

	// 获取列表
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&admins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取管理员列表失败: " + err.Error(),
		})
		return
	}

	// 加载每个管理员的角色（使用 GORM 的 Preload）
	for i := range admins {
		database.DB.Preload("Roles").Find(&admins[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  admins,
			"total": total,
		},
	})
}

// GetAdmin 获取管理员详情
func GetAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的管理员ID",
		})
		return
	}

	var admin models.Admin
	if err := database.DB.Preload("Roles").First(&admin, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "管理员不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取管理员详情失败: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": admin,
	})
}

// CreateAdmin 创建管理员
func CreateAdmin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		RoleIDs  []uint `json:"role_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 检查用户名是否已存在
	var existAdmin models.Admin
	if err := database.DB.Where("username = ?", req.Username).First(&existAdmin).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户名已存在",
		})
		return
	}

	// 创建管理员
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "密码加密失败: " + err.Error(),
		})
		return
	}

	now := time.Now().Unix()
	admin := models.Admin{
		Username:  req.Username,
		Password:  hashedPassword,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建管理员失败: " + err.Error(),
		})
		return
	}

	// 分配角色
	if len(req.RoleIDs) > 0 {
		for _, roleID := range req.RoleIDs {
			database.DB.Create(&models.AdminRoleRelation{
				AdminID:   admin.ID,
				RoleID:    roleID,
				CreatedAt: time.Now().Unix(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    admin,
	})
}

// UpdateAdmin 更新管理员
func UpdateAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的管理员ID",
		})
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Status   int    `json:"status"`
		RoleIDs  []uint `json:"role_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "管理员不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取管理员失败: " + err.Error(),
			})
		}
		return
	}

	// 更新信息
	admin.Nickname = req.Nickname
	admin.Email = req.Email
	admin.Status = int8(req.Status)
	admin.UpdatedAt = time.Now().Unix()

	// 更新密码（如果提供）
	if req.Password != "" {
		hashedPassword, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "密码加密失败: " + err.Error(),
			})
			return
		}
		admin.Password = hashedPassword
	}

	if err := database.DB.Save(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新管理员失败: " + err.Error(),
		})
		return
	}

	// 更新角色关联
	if req.RoleIDs != nil {
		database.DB.Where("admin_id = ?", admin.ID).Delete(&models.AdminRoleRelation{})
		for _, roleID := range req.RoleIDs {
			database.DB.Create(&models.AdminRoleRelation{
				AdminID:   admin.ID,
				RoleID:    roleID,
				CreatedAt: time.Now().Unix(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    admin,
	})
}

// DeleteAdmin 删除管理员
func DeleteAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的管理员ID",
		})
		return
	}

	var admin models.Admin
	if err := database.DB.First(&admin, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "管理员不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取管理员失败: " + err.Error(),
			})
		}
		return
	}

	// 不能删除自己
	adminID := c.GetUint("admin_id")
	if admin.ID == adminID {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "不能删除自己",
		})
		return
	}

	// 删除角色关联
	database.DB.Where("admin_id = ?", admin.ID).Delete(&models.AdminRoleRelation{})
	// 删除管理员
	database.DB.Delete(&admin)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}
