package models

import (
	"gorm.io/gorm"
)

// Admin 管理员模型
type Admin struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Username    string         `gorm:"uniqueIndex;size:50;not null;comment:管理员用户名" json:"username"`
	Password    string         `gorm:"size:255;not null;comment:密码(加密后)" json:"-"`
	Nickname    string         `gorm:"size:50;not null;default:'';comment:昵称" json:"nickname"`
	Email       string         `gorm:"size:100;default:'';comment:邮箱" json:"email"`
	Avatar      string         `gorm:"size:255;default:'';comment:头像" json:"avatar"`
	Status      int8           `gorm:"default:1;comment:状态:1正常,2禁用" json:"status"`
	LastLoginAt *int64         `gorm:"type:bigint;default:0;comment:最后登录时间" json:"last_login_at"`
	LastLoginIP string         `gorm:"size:50;default:'';comment:最后登录IP" json:"last_login_ip"`
	CreatedAt   int64          `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt   int64          `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Roles []AdminRole `gorm:"many2many:admin_role_relations;joinForeignKey:admin_id;joinReferences:role_id" json:"roles,omitempty"`
}

// BeforeCreate GORM创建前钩子
func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if a.CreatedAt == 0 {
		a.CreatedAt = now
	}
	if a.UpdatedAt == 0 {
		a.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (a *Admin) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (Admin) TableName() string {
	return "admins"
}

// AdminRole 管理员角色模型
type AdminRole struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	RoleCode    string `gorm:"uniqueIndex;size:50;not null;comment:角色代码" json:"role_code"`
	RoleName    string `gorm:"size:50;not null;comment:角色名称" json:"role_name"`
	Description string `gorm:"size:255;default:'';comment:角色描述" json:"description"`
	Status      int8   `gorm:"default:1;comment:状态:1启用,2禁用" json:"status"`
	CreatedAt   int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt   int64  `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`

	// 关联关系
	Permissions []AdminPermission `gorm:"many2many:role_permission_relations;joinForeignKey:role_id;joinReferences:permission_id" json:"permissions,omitempty"`
}

// BeforeCreate GORM创建前钩子
func (r *AdminRole) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (r *AdminRole) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (AdminRole) TableName() string {
	return "admin_roles"
}

// AdminPermission 权限模型
type AdminPermission struct {
	ID             uint   `gorm:"primarykey" json:"id"`
	PermissionCode string `gorm:"uniqueIndex;size:100;not null;comment:权限代码" json:"permission_code"`
	PermissionName string `gorm:"size:100;not null;comment:权限名称" json:"permission_name"`
	Resource       string `gorm:"size:50;not null;comment:资源类型" json:"resource"`
	Action         string `gorm:"size:50;not null;comment:操作类型" json:"action"`
	ParentID       uint   `gorm:"default:0;comment:父权限ID" json:"parent_id"`
	SortOrder      int    `gorm:"default:0;comment:排序" json:"sort_order"`
	Description    string `gorm:"size:255;default:'';comment:权限描述" json:"description"`
	CreatedAt      int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (p *AdminPermission) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (AdminPermission) TableName() string {
	return "admin_permissions"
}

// AdminRoleRelation 管理员角色关联表
type AdminRoleRelation struct {
	ID        uint  `gorm:"primarykey" json:"id"`
	AdminID   uint  `gorm:"uniqueIndex:uk_admin_role;not null;comment:管理员ID" json:"admin_id"`
	RoleID    uint  `gorm:"uniqueIndex:uk_admin_role;not null;comment:角色ID" json:"role_id"`
	CreatedAt int64 `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (r *AdminRoleRelation) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (AdminRoleRelation) TableName() string {
	return "admin_role_relations"
}

// RolePermissionRelation 角色权限关联表
type RolePermissionRelation struct {
	ID           uint  `gorm:"primarykey" json:"id"`
	RoleID       uint  `gorm:"uniqueIndex:uk_role_permission;not null;comment:角色ID" json:"role_id"`
	PermissionID uint  `gorm:"uniqueIndex:uk_role_permission;not null;comment:权限ID" json:"permission_id"`
	CreatedAt    int64 `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (r *RolePermissionRelation) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (RolePermissionRelation) TableName() string {
	return "role_permission_relations"
}
