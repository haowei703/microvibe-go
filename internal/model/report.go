package model

import (
	"time"
)

// Report 举报模型
type Report struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ReporterID  uint   `gorm:"index;not null" json:"reporter_id"` // 举报人ID
	TargetID    uint   `gorm:"index;not null" json:"target_id"`   // 举报对象ID (用户/视频/评论)
	TargetType  int8   `gorm:"index;not null" json:"target_type"` // 举报类型：1-用户，2-视频，3-评论
	Reason      string `gorm:"size:255;not null" json:"reason"`   // 举报原因
	Description string `gorm:"type:text" json:"description"`      // 具体描述
	Status      int8   `gorm:"default:0;index" json:"status"`     // 状态：0-待处理，1-已处理，2-已驳回

	// 关联
	Reporter *User `gorm:"foreignKey:ReporterID" json:"reporter,omitempty"`
}

// TableName 指定表名
func (Report) TableName() string {
	return "reports"
}
