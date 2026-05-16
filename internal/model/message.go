package model

import (
	"time"

	"gorm.io/gorm"
)

// Message 消息模型
type Message struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	SenderID       uint   `gorm:"index;not null" json:"sender_id"`   // 发送者ID
	ReceiverID     uint   `gorm:"index;not null" json:"receiver_id"` // 接收者ID
	ConversationID *uint  `gorm:"index" json:"conversation_id"`      // 会话ID
	Type           int8   `gorm:"default:1" json:"type"`             // 消息类型：1-文本，2-图片，3-视频，4-语音
	Content        string `gorm:"type:text;not null" json:"content"` // 消息内容
	MediaURL       string `gorm:"size:255" json:"media_url"`         // 媒体URL（图片/视频/语音）
	VideoID        *uint  `gorm:"index" json:"video_id,omitempty"`   // 转发的视频ID

	// 状态
	IsRead bool       `gorm:"default:false;index" json:"is_read"` // 是否已读
	ReadAt *time.Time `json:"read_at"`                            // 读取时间

	// 关联
	Sender   *User  `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Receiver *User  `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
	Video    *Video `gorm:"foreignKey:VideoID" json:"video,omitempty"`
}

// MessageVO 消息视图对象
type MessageVO struct {
	ID             uint       `json:"id"`
	SenderID       uint       `json:"sender_id"`
	ReceiverID     uint       `json:"receiver_id"`
	ConversationID *uint      `json:"conversation_id"`
	Type           int8       `json:"type"`
	Content        string     `json:"content"`
	MediaURL       string     `json:"media_url"`
	VideoID        *uint      `json:"video_id,omitempty"`
	IsRead         bool       `json:"is_read"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time  `json:"created_at"`
	IsMine         bool       `json:"is_mine"` // 视角字段

	// 关联对象 VO
	Sender   *AuthorVO `json:"sender,omitempty"`
	Receiver *AuthorVO `json:"receiver,omitempty"`
	Video    *Video    `json:"video,omitempty"`
}

// TableName 指定表名
func (Message) TableName() string {
	return "messages"
}

// Conversation 会话模型
type Conversation struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User1ID       uint   `gorm:"index;not null" json:"user1_id"` // 用户1ID
	User2ID       uint   `gorm:"index;not null" json:"user2_id"` // 用户2ID
	LastMessageID *uint  `json:"last_message_id"`                // 最后一条消息ID
	LastContent   string `gorm:"size:500" json:"last_content"`   // 最后消息内容
	UnreadCount1  int    `gorm:"default:0" json:"unread_count1"` // 用户1未读数
	UnreadCount2  int    `gorm:"default:0" json:"unread_count2"` // 用户2未读数

	// 关联
	User1       *User    `gorm:"foreignKey:User1ID" json:"user1,omitempty"`
	User2       *User    `gorm:"foreignKey:User2ID" json:"user2,omitempty"`
	LastMessage *Message `gorm:"foreignKey:LastMessageID" json:"last_message,omitempty"`
}

// TableName 指定表名
func (Conversation) TableName() string {
	return "conversations"
}

// ConversationVO 会话响应对象 (用于 API 视角转换)
type ConversationVO struct {
	ID          uint       `json:"id"`
	UserID      uint       `json:"user_id"`      // 对方用户ID
	Nickname    string     `json:"nickname"`     // 对方昵称
	Avatar      string     `json:"avatar"`       // 对方头像
	LastMessage *MessageVO `json:"last_message"` // 最后一条消息
	UnreadCount int        `json:"unread_count"` // 当前用户的未读数
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Notification 通知模型
type Notification struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	UserID    uint   `gorm:"index;not null" json:"user_id"` // 接收通知的用户ID
	Type      int8   `gorm:"index;not null" json:"type"`    // 通知类型：1-点赞，2-评论，3-关注，4-@我的，5-系统通知
	SenderID  *uint  `gorm:"index" json:"sender_id"`        // 发送者ID
	RelatedID *uint  `json:"related_id"`                    // 关联ID（视频ID/评论ID等）
	Title     string `gorm:"size:200" json:"title"`         // 标题
	Content   string `gorm:"type:text" json:"content"`      // 内容
	Link      string `gorm:"size:255" json:"link"`          // 链接

	// 视频相关字段（用于互动消息展示）
	VideoID       *uint  `json:"video_id"`                        // 关联的视频ID
	VideoCoverURL string `gorm:"size:500" json:"video_cover_url"` // 视频封面URL
	VideoTitle    string `gorm:"size:200" json:"video_title"`     // 视频标题

	// 评论相关字段（用于评论类通知）
	CommentID      *uint  `json:"comment_id"`                       // 评论ID（用于定位评论）
	CommentContent string `gorm:"type:text" json:"comment_content"` // 评论内容预览

	// 用户相关字段
	UserTag string `gorm:"size:50" json:"user_tag"` // 用户标签（如"AI分身"）

	// 状态
	IsRead bool       `gorm:"default:false;index" json:"is_read"` // 是否已读
	ReadAt *time.Time `json:"read_at"`                            // 读取时间

	// 关联
	User   *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Sender *User  `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Video  *Video `gorm:"foreignKey:VideoID" json:"video,omitempty"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "notifications"
}
