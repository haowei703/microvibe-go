package model

import (
	"time"
)

// Blacklist 黑名单模型
type Blacklist struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID        uint `gorm:"index;not null" json:"user_id"`         // 主动拉黑的用户ID
	BlockedUserID uint `gorm:"index;not null" json:"blocked_user_id"` // 被拉黑的用户ID

	// 关联
	User        *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	BlockedUser *User `gorm:"foreignKey:BlockedUserID" json:"blocked_user,omitempty"`
}

// TableName 指定表名
func (Blacklist) TableName() string {
	return "blacklists"
}
