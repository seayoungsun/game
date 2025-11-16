package models

import (
	"gorm.io/gorm"
)

// Announcement 系统公告
type Announcement struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	Title       string `gorm:"size:200;not null;comment:公告标题" json:"title"`
	Content     string `gorm:"type:text;not null;comment:公告内容" json:"content"`
	Type        string `gorm:"size:20;default:'info';comment:公告类型:info/warning/error/success" json:"type"`
	Priority    int    `gorm:"default:0;comment:优先级:0普通,1重要,2紧急" json:"priority"`
	Status      int    `gorm:"default:1;comment:状态:1发布,2下架" json:"status"`
	StartTime   *int64 `gorm:"type:bigint;comment:开始时间" json:"start_time"`
	EndTime     *int64 `gorm:"type:bigint;comment:结束时间" json:"end_time"`
	TargetUsers string `gorm:"type:text;comment:目标用户:all=全部,user_id1,user_id2=指定用户" json:"target_users"`
	CreatedBy   uint   `gorm:"comment:创建人ID" json:"created_by"`
	CreatedAt   int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt   int64  `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (a *Announcement) BeforeCreate(tx *gorm.DB) error {
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
func (a *Announcement) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (Announcement) TableName() string {
	return "announcements"
}

// UserMessage 用户消息
type UserMessage struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	UserID    uint   `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Type      string `gorm:"size:20;default:'info';comment:消息类型:info/warning/error/success/system/order" json:"type"`
	Title     string `gorm:"size:200;not null;comment:消息标题" json:"title"`
	Content   string `gorm:"type:text;not null;comment:消息内容" json:"content"`
	RelatedID string `gorm:"size:64;comment:关联ID(如订单号)" json:"related_id"`
	IsRead    bool   `gorm:"default:0;comment:是否已读" json:"is_read"`
	ReadAt    *int64 `gorm:"type:bigint;comment:阅读时间" json:"read_at"`
	CreatedAt int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt int64  `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (u *UserMessage) BeforeCreate(tx *gorm.DB) error {
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
func (u *UserMessage) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (UserMessage) TableName() string {
	return "user_messages"
}
