package models

import (
	"gorm.io/gorm"
)

// Transaction 交易订单
type Transaction struct {
	ID        uint    `gorm:"primarykey" json:"id"`
	OrderID   string  `gorm:"uniqueIndex;size:64;not null;comment:订单号" json:"order_id"`
	UserID    uint    `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Type      string  `gorm:"size:20;not null;comment:类型:recharge/withdraw/game" json:"type"`
	Amount    float64 `gorm:"type:decimal(10,2);not null;comment:金额" json:"amount"`
	Status    int8    `gorm:"default:1;comment:状态:1待处理,2成功,3失败" json:"status"`
	Channel   string  `gorm:"size:20;comment:支付渠道:alipay/wechat" json:"channel"`
	ChannelID string  `gorm:"size:100;comment:第三方订单号" json:"channel_id"`
	Remark    string  `gorm:"size:255;comment:备注" json:"remark"`
	CreatedAt int64   `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt int64   `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	if t.UpdatedAt == 0 {
		t.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (t *Transaction) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (Transaction) TableName() string {
	return "transactions"
}

// RechargeOrder 充值订单
type RechargeOrder struct {
	ID           uint    `gorm:"primarykey" json:"id"`
	OrderID      string  `gorm:"uniqueIndex;size:64;not null;comment:订单号" json:"order_id"`
	UserID       uint    `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Amount       float64 `gorm:"type:decimal(10,2);not null;comment:充值金额" json:"amount"`
	Status       int8    `gorm:"default:1;comment:状态:1待支付,2已支付,3已取消" json:"status"`
	Channel      string  `gorm:"size:20;comment:支付渠道:usdt_trc20/usdt_erc20" json:"channel"`
	ChannelID    string  `gorm:"size:100;comment:第三方订单号" json:"channel_id"`
	ChainType    string  `gorm:"size:20;comment:链类型:trc20/erc20" json:"chain_type"`
	DepositAddr  string  `gorm:"size:100;index;comment:充值地址" json:"deposit_addr"`
	TxHash       string  `gorm:"size:128;index;comment:交易哈希" json:"tx_hash"`
	ConfirmCount int     `gorm:"default:0;comment:确认次数" json:"confirm_count"`
	RequiredConf int     `gorm:"default:12;comment:需要确认次数" json:"required_conf"`
	PaidAt       *int64  `gorm:"type:bigint;default:0;comment:支付时间" json:"paid_at"`
	ExpireAt     int64   `gorm:"type:bigint;not null;default:0;comment:过期时间" json:"expire_at"`
	CreatedAt    int64   `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt    int64   `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (r *RechargeOrder) BeforeCreate(tx *gorm.DB) error {
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
func (r *RechargeOrder) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (RechargeOrder) TableName() string {
	return "recharge_orders"
}

// WithdrawOrder 提现订单
type WithdrawOrder struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	OrderID      string         `gorm:"uniqueIndex;size:64;not null;comment:订单号" json:"order_id"`
	UserID       uint           `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Amount       float64        `gorm:"type:decimal(10,2);not null;comment:提现金额" json:"amount"`
	Fee          float64        `gorm:"type:decimal(10,2);not null;default:0;comment:手续费" json:"fee"`
	ActualAmount float64        `gorm:"type:decimal(10,2);not null;comment:实际到账金额" json:"actual_amount"`
	Status       int8           `gorm:"default:1;comment:状态:1待审核,2已通过,3已拒绝" json:"status"`
	Channel      string         `gorm:"size:20;comment:支付渠道:usdt_trc20/usdt_erc20" json:"channel"`
	ChainType    string         `gorm:"size:20;comment:链类型:trc20/erc20" json:"chain_type"`
	ToAddress    string         `gorm:"size:100;index;comment:提现地址" json:"to_address"`
	TxHash       string         `gorm:"size:128;index;comment:交易哈希" json:"tx_hash"`
	ConfirmCount int            `gorm:"default:0;comment:确认次数" json:"confirm_count"`
	BankCard     string         `gorm:"size:50;comment:银行卡号（已废弃，保留兼容）" json:"bank_card"`
	BankName     string         `gorm:"size:50;comment:银行名称（已废弃，保留兼容）" json:"bank_name"`
	RealName     string         `gorm:"size:50;comment:真实姓名（已废弃，保留兼容）" json:"real_name"`
	Remark       string         `gorm:"size:255;comment:备注" json:"remark"`
	AuditAt      *int64         `gorm:"type:bigint;default:0;comment:审核时间" json:"audit_at"`
	AuditorID    uint           `gorm:"comment:审核员ID" json:"auditor_id"`
	CreatedAt    int64          `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt    int64          `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate GORM创建前钩子
func (w *WithdrawOrder) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if w.CreatedAt == 0 {
		w.CreatedAt = now
	}
	if w.UpdatedAt == 0 {
		w.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (w *WithdrawOrder) BeforeUpdate(tx *gorm.DB) error {
	w.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (WithdrawOrder) TableName() string {
	return "withdraw_orders"
}

// UserDepositAddress 用户充值地址
type UserDepositAddress struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	UserID    uint   `gorm:"uniqueIndex:idx_user_chain;not null;comment:用户ID" json:"user_id"`
	ChainType string `gorm:"uniqueIndex:idx_user_chain;size:20;not null;comment:链类型:trc20/erc20" json:"chain_type"`
	Address   string `gorm:"size:100;not null;uniqueIndex;comment:充值地址" json:"address"`
	CreatedAt int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt int64  `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (u *UserDepositAddress) BeforeCreate(tx *gorm.DB) error {
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
func (u *UserDepositAddress) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (UserDepositAddress) TableName() string {
	return "user_deposit_addresses"
}
