package models

import (
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	UID       int64          `gorm:"uniqueIndex;not null;comment:用户ID" json:"uid"`
	Phone     string         `gorm:"uniqueIndex;size:20;not null;comment:手机号" json:"phone"`
	Password  string         `gorm:"size:255;not null;comment:密码(加密后)" json:"-"`
	Nickname  string         `gorm:"size:50;not null;default:'';comment:昵称" json:"nickname"`
	Avatar    string         `gorm:"size:255;default:'';comment:头像" json:"avatar"`
	Balance   float64        `gorm:"type:decimal(10,2);default:0;comment:余额" json:"balance"`
	Status    int8           `gorm:"default:1;comment:状态:1正常,2封禁" json:"status"`
	CreatedAt int64          `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt int64          `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate GORM创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if u.CreatedAt == 0 {
		u.CreatedAt = now
	}
	if u.UpdatedAt == 0 {
		u.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// UserWallet 用户钱包
type UserWallet struct {
	ID        uint    `gorm:"primarykey" json:"id"`
	UserID    uint    `gorm:"uniqueIndex;not null;comment:用户ID" json:"user_id"`
	Balance   float64 `gorm:"type:decimal(10,2);default:0;comment:余额" json:"balance"`
	Frozen    float64 `gorm:"type:decimal(10,2);default:0;comment:冻结金额" json:"frozen"`
	TotalIn   float64 `gorm:"type:decimal(10,2);default:0;comment:累计充值" json:"total_in"`
	TotalOut  float64 `gorm:"type:decimal(10,2);default:0;comment:累计提现" json:"total_out"`
	UpdatedAt int64   `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (u *UserWallet) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if u.UpdatedAt == 0 {
		u.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (u *UserWallet) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (UserWallet) TableName() string {
	return "user_wallets"
}

// UserLogin 用户登录记录
type UserLogin struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	UserID    uint   `gorm:"index;not null;comment:用户ID" json:"user_id"`
	IP        string `gorm:"size:50;comment:IP地址" json:"ip"`
	Device    string `gorm:"size:100;comment:设备信息" json:"device"`
	CreatedAt int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (u *UserLogin) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if u.CreatedAt == 0 {
		u.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (UserLogin) TableName() string {
	return "user_logins"
}
