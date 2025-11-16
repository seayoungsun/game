package models

import (
	"gorm.io/gorm"
)

// AdminOperationLog 管理员操作日志
type AdminOperationLog struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	AdminID   uint   `gorm:"index;not null;comment:管理员ID" json:"admin_id"`
	AdminName string `gorm:"size:50;not null;comment:管理员用户名" json:"admin_name"`
	Module    string `gorm:"size:50;not null;comment:操作模块" json:"module"`
	Action    string `gorm:"size:50;not null;comment:操作动作" json:"action"`
	Method    string `gorm:"size:10;comment:HTTP方法" json:"method"`
	Path      string `gorm:"size:255;comment:请求路径" json:"path"`
	IP        string `gorm:"size:50;comment:IP地址" json:"ip"`
	UserAgent string `gorm:"size:255;comment:用户代理" json:"user_agent"`
	Request   string `gorm:"type:text;comment:请求参数" json:"request"`
	Response  string `gorm:"type:text;comment:响应结果" json:"response"`
	Status    int    `gorm:"default:1;comment:状态:1成功,2失败" json:"status"`
	ErrorMsg  string `gorm:"type:text;comment:错误信息" json:"error_msg"`
	Duration  int64  `gorm:"comment:耗时(毫秒)" json:"duration"`
	CreatedAt int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (a *AdminOperationLog) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if a.CreatedAt == 0 {
		a.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (AdminOperationLog) TableName() string {
	return "admin_operation_logs"
}

// SystemConfig 系统配置
type SystemConfig struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	ConfigKey   string `gorm:"uniqueIndex;size:100;not null;comment:配置键" json:"config_key"`
	ConfigValue string `gorm:"type:text;comment:配置值" json:"config_value"`
	ConfigType  string `gorm:"size:20;default:string;comment:配置类型:string/int/float/bool/json" json:"config_type"`
	GroupName   string `gorm:"size:50;default:'default';comment:配置分组" json:"group_name"`
	Description string `gorm:"size:255;comment:配置说明" json:"description"`
	IsPublic    bool   `gorm:"default:0;comment:是否公开" json:"is_public"`
	CreatedAt   int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt   int64  `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (s *SystemConfig) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (s *SystemConfig) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (SystemConfig) TableName() string {
	return "system_configs"
}
