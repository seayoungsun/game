package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

// JSON 自定义JSON类型
type JSON json.RawMessage

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("无法转换为[]byte")
	}
	*j = JSON(bytes)
	return nil
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("json: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// GameRoom 游戏房间
type GameRoom struct {
	ID             uint    `gorm:"primarykey" json:"id"`
	RoomID         string  `gorm:"uniqueIndex;size:50;not null;comment:房间ID" json:"room_id"`
	GameType       string  `gorm:"size:20;not null;comment:游戏类型" json:"game_type"`
	RoomType       string  `gorm:"size:20;comment:房间类型:quick/middle/high" json:"room_type"`
	BaseBet        float64 `gorm:"type:decimal(10,2);comment:底注" json:"base_bet"`
	MaxPlayers     int     `gorm:"default:4;comment:最大人数" json:"max_players"`
	CurrentPlayers int     `gorm:"default:0;comment:当前人数" json:"current_players"`
	Status         int8    `gorm:"default:1;comment:状态:1等待,2游戏中,3已结束" json:"status"`
	Password       string  `gorm:"size:20;default:'';comment:房间密码" json:"-"`    // 密码不返回给客户端
	HasPassword    bool    `gorm:"default:0;comment:是否有密码" json:"has_password"` // 是否设置了密码
	Players        JSON    `gorm:"type:json;comment:玩家列表" json:"players"`
	CreatorID      uint    `gorm:"comment:创建者ID" json:"creator_id"`
	CreatedAt      int64   `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt      int64   `gorm:"type:bigint;not null;default:0;comment:更新时间" json:"updated_at"`
}

// BeforeCreate GORM创建前钩子
func (g *GameRoom) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if g.CreatedAt == 0 {
		g.CreatedAt = now
	}
	if g.UpdatedAt == 0 {
		g.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM更新前钩子
func (g *GameRoom) BeforeUpdate(tx *gorm.DB) error {
	g.UpdatedAt = tx.Statement.DB.NowFunc().Unix()
	return nil
}

// TableName 表名
func (GameRoom) TableName() string {
	return "game_rooms"
}

// GameRecord 游戏对局记录（摘要）
type GameRecord struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	RoomID    string `gorm:"index;size:50;not null;comment:房间ID" json:"room_id"`
	GameType  string `gorm:"size:20;not null;comment:游戏类型" json:"game_type"`
	Players   JSON   `gorm:"type:json;comment:玩家列表" json:"players"`
	Result    JSON   `gorm:"type:json;comment:结算结果" json:"result"`
	StartTime int64  `gorm:"type:bigint;not null;default:0;comment:开始时间" json:"start_time"`
	EndTime   int64  `gorm:"type:bigint;not null;default:0;comment:结束时间" json:"end_time"`
	Duration  int    `gorm:"default:0;comment:时长(秒)" json:"duration"`
	CreatedAt int64  `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (g *GameRecord) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if g.CreatedAt == 0 {
		g.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (GameRecord) TableName() string {
	return "game_records"
}

// GamePlayer 游戏玩家关联
type GamePlayer struct {
	ID        uint    `gorm:"primarykey" json:"id"`
	RoomID    string  `gorm:"index;size:50;not null;comment:房间ID" json:"room_id"`
	UserID    uint    `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Position  int     `gorm:"comment:位置" json:"position"`
	Balance   float64 `gorm:"type:decimal(10,2);default:0;comment:本局余额变化" json:"balance"`
	CreatedAt int64   `gorm:"type:bigint;not null;default:0;comment:创建时间" json:"created_at"`
}

// BeforeCreate GORM创建前钩子
func (g *GamePlayer) BeforeCreate(tx *gorm.DB) error {
	now := tx.Statement.DB.NowFunc().Unix()
	if g.CreatedAt == 0 {
		g.CreatedAt = now
	}
	return nil
}

// TableName 表名
func (GamePlayer) TableName() string {
	return "game_players"
}
