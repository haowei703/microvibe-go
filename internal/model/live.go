package model

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 直播间相关 ====================

// LiveStream 直播间模型（综合抖音直播功能）
type LiveStream struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// ========== 基本信息 ==========
	OwnerID     uint   `gorm:"index;not null" json:"owner_id"`                    // 主播用户ID
	Title       string `gorm:"size:255;not null" json:"title" binding:"required"` // 直播标题
	Description string `gorm:"type:text" json:"description"`                      // 直播描述
	Cover       string `gorm:"size:512" json:"cover"`                             // 封面图
	Notice      string `gorm:"size:500" json:"notice"`                            // 直播间公告
	CategoryID  *uint  `gorm:"index" json:"category_id"`                          // 分类ID（可选）
	Tags        string `gorm:"size:255" json:"tags"`                              // 标签（逗号分隔）

	// ========== 推流配置 ==========
	StreamKey    string `gorm:"size:64;uniqueIndex;not null" json:"stream_key,omitempty"` // 推流密钥（主播专用，不对外暴露）
	RoomID       string `gorm:"size:64;uniqueIndex;not null" json:"room_id"`              // 房间ID（用于 WebRTC 信令）
	StreamURL    string `gorm:"size:512" json:"stream_url,omitempty"`                     // RTMP 推流地址（主播专用，用于OBS）
	PlayURL      string `gorm:"size:512" json:"play_url"`                                 // HLS 播放地址
	FlvURL       string `gorm:"size:512" json:"flv_url"`                                  // FLV 播放地址
	RtmpURL      string `gorm:"size:512" json:"rtmp_url"`                                 // RTMP 播放地址
	WebRTCURL    string `gorm:"size:512" json:"webrtc_url"`                               // WebRTC 播放地址
	RecordURL    string `gorm:"size:512" json:"record_url"`                               // 录制回放地址
	PushProtocol string `gorm:"size:50;default:'rtmp'" json:"push_protocol"`              // 推流协议：rtmp, webrtc, srt

	// ========== 流类型配置 ==========
	StreamType   string `gorm:"size:50;default:'video_audio'" json:"stream_type"` // 流类型：video_only-纯视频, audio_only-纯音频, video_audio-音视频
	HasVideo     bool   `gorm:"default:true" json:"has_video"`                    // 是否包含视频流
	HasAudio     bool   `gorm:"default:true" json:"has_audio"`                    // 是否包含音频流
	VideoCodec   string `gorm:"size:20;default:'h264'" json:"video_codec"`        // 视频编码：h264, h265, vp8, vp9, av1
	AudioCodec   string `gorm:"size:20;default:'aac'" json:"audio_codec"`         // 音频编码：aac, opus, mp3
	VideoBitrate int    `gorm:"default:2500" json:"video_bitrate"`                // 视频码率（kbps）
	AudioBitrate int    `gorm:"default:128" json:"audio_bitrate"`                 // 音频码率（kbps）
	FrameRate    int    `gorm:"default:30" json:"frame_rate"`                     // 帧率：15, 24, 30, 60
	Resolution   string `gorm:"size:20;default:'720p'" json:"resolution"`         // 分辨率：360p, 480p, 720p, 1080p, 2k, 4k

	// ========== 直播设置 ==========
	Quality     int8 `gorm:"default:2" json:"quality"`        // 清晰度：1-标清，2-高清，3-超清，4-蓝光
	BeautyLevel int8 `gorm:"default:0" json:"beauty_level"`   // 美颜等级：0-关闭，1-5级
	FilterType  int8 `gorm:"default:0" json:"filter_type"`    // 滤镜类型：0-无，1-清新，2-复古等
	IsVertical  bool `gorm:"default:true" json:"is_vertical"` // 是否竖屏直播

	// ========== 统计数据 ==========
	ViewCount     int64 `gorm:"default:0;index" json:"view_count"`   // 累计观看人数
	LikeCount     int64 `gorm:"default:0;index" json:"like_count"`   // 点赞数
	GiftCount     int64 `gorm:"default:0" json:"gift_count"`         // 礼物数量
	GiftValue     int64 `gorm:"default:0;index" json:"gift_value"`   // 礼物总价值（虚拟币）
	CommentCount  int64 `gorm:"default:0" json:"comment_count"`      // 评论（弹幕）数
	ShareCount    int64 `gorm:"default:0" json:"share_count"`        // 分享次数
	OnlineCount   int   `gorm:"default:0;index" json:"online_count"` // 当前在线人数
	PeakCount     int   `gorm:"default:0" json:"peak_count"`         // 峰值在线人数
	FollowerCount int64 `gorm:"default:0" json:"follower_count"`     // 直播间新增关注数
	ProductSales  int64 `gorm:"default:0" json:"product_sales"`      // 商品销售额

	// ========== 状态控制 ==========
	Status     string     `gorm:"size:20;default:'waiting';index" json:"status"` // 状态: waiting-待开播, live-直播中, paused-暂停, ended-已结束, banned-禁播
	StartedAt  *time.Time `json:"started_at"`                                    // 开播时间
	EndedAt    *time.Time `json:"ended_at"`                                      // 结束时间
	Duration   int64      `gorm:"default:0" json:"duration"`                     // 直播时长（秒）
	IsPinned   bool       `gorm:"default:false" json:"is_pinned"`                // 是否置顶
	IsRecorded bool       `gorm:"default:true" json:"is_recorded"`               // 是否录制

	// ========== 互动控制 ==========
	AllowComment bool   `gorm:"default:true" json:"allow_comment"` // 允许评论
	AllowGift    bool   `gorm:"default:true" json:"allow_gift"`    // 允许送礼
	AllowShare   bool   `gorm:"default:true" json:"allow_share"`   // 允许分享
	IsPrivate    bool   `gorm:"default:false" json:"is_private"`   // 是否私密直播（仅粉丝可见）
	Password     string `gorm:"size:64" json:"-"`                  // 密码房间密码（加密存储）

	// ========== 商业化 ==========
	HasProducts   bool  `gorm:"default:false" json:"has_products"` // 是否挂载商品
	ProductCount  int   `gorm:"default:0" json:"product_count"`    // 商品数量
	HasReward     bool  `gorm:"default:true" json:"has_reward"`    // 是否开启打赏
	RewardGoal    int64 `gorm:"default:0" json:"reward_goal"`      // 打赏目标（虚拟币）
	RewardCurrent int64 `gorm:"default:0" json:"reward_current"`   // 当前打赏进度

	// ========== 关联数据 ==========
	Owner    *User     `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

// TableName 指定表名
func (LiveStream) TableName() string {
	return "live_streams"
}

// ==================== 观众相关 ====================

// LiveViewer 直播观众记录
type LiveViewer struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	LiveID   uint       `gorm:"index:idx_live_viewer;not null" json:"live_id"` // 直播间ID
	UserID   uint       `gorm:"index:idx_live_viewer;not null" json:"user_id"` // 用户ID
	JoinedAt time.Time  `gorm:"not null" json:"joined_at"`                     // 加入时间
	LeftAt   *time.Time `json:"left_at"`                                       // 离开时间
	Duration int        `gorm:"default:0" json:"duration"`                     // 观看时长（秒）

	// 互动统计
	CommentCount int   `gorm:"default:0" json:"comment_count"` // 发送弹幕数
	LikeCount    int   `gorm:"default:0" json:"like_count"`    // 点赞次数
	GiftCount    int   `gorm:"default:0" json:"gift_count"`    // 送礼次数
	GiftValue    int64 `gorm:"default:0" json:"gift_value"`    // 送礼总价值

	// 行为标记
	IsFollowed bool `gorm:"default:false" json:"is_followed"` // 是否新关注
	IsShared   bool `gorm:"default:false" json:"is_shared"`   // 是否分享过

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LiveViewer) TableName() string {
	return "live_viewers"
}

// ==================== 礼物相关 ====================

// LiveGift 直播礼物定义
type LiveGift struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string     `gorm:"size:50;not null" json:"name"`     // 礼物名称
	Icon        string     `gorm:"size:255" json:"icon"`             // 礼物图标URL
	Animation   string     `gorm:"size:255" json:"animation"`        // 礼物动画URL（SVGA/Lottie）
	Price       int64      `gorm:"not null" json:"price"`            // 礼物价格（虚拟币）
	Type        int8       `gorm:"default:1;index" json:"type"`      // 类型：1-普通，2-豪华，3-专属
	Level       int8       `gorm:"default:1" json:"level"`           // 等级：1-5级
	Duration    int        `gorm:"default:3" json:"duration"`        // 动画时长（秒）
	ShowBanner  bool       `gorm:"default:false" json:"show_banner"` // 是否全屏横幅
	Sound       string     `gorm:"size:255" json:"sound"`            // 音效URL
	Description string     `gorm:"size:200" json:"description"`      // 礼物描述
	Sort        int        `gorm:"default:0;index" json:"sort"`      // 排序
	Status      int8       `gorm:"default:1;index" json:"status"`    // 状态：0-禁用，1-启用
	IsLimited   bool       `gorm:"default:false" json:"is_limited"`  // 是否限时礼物
	StartTime   *time.Time `json:"start_time"`                       // 限时开始时间
	EndTime     *time.Time `json:"end_time"`                         // 限时结束时间
}

// TableName 指定表名
func (LiveGift) TableName() string {
	return "live_gifts"
}

// LiveGiftRecord 直播礼物记录
type LiveGiftRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	LiveID      uint   `gorm:"index:idx_live_gift;not null" json:"live_id"` // 直播间ID
	UserID      uint   `gorm:"index:idx_user_gift;not null" json:"user_id"` // 送礼用户ID
	TargetID    uint   `gorm:"index" json:"target_id"`                      // 目标用户ID（送给主播或其他用户）
	GiftID      uint   `gorm:"index;not null" json:"gift_id"`               // 礼物ID
	GiftName    string `gorm:"size:50" json:"gift_name"`                    // 礼物名称（冗余字段）
	Quantity    int    `gorm:"not null;default:1" json:"quantity"`          // 数量
	UnitPrice   int64  `gorm:"not null" json:"unit_price"`                  // 单价
	TotalValue  int64  `gorm:"not null;index" json:"total_value"`           // 总价值
	ComboCount  int    `gorm:"default:1" json:"combo_count"`                // 连击数
	IsAnonymous bool   `gorm:"default:false" json:"is_anonymous"`           // 是否匿名送礼
	Message     string `gorm:"size:200" json:"message"`                     // 附带消息

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Gift *LiveGift   `gorm:"foreignKey:GiftID" json:"gift,omitempty"`
}

// TableName 指定表名
func (LiveGiftRecord) TableName() string {
	return "live_gift_records"
}

// ==================== 弹幕评论相关 ====================

// LiveComment 直播弹幕评论
type LiveComment struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	LiveID  uint   `gorm:"index:idx_live_comment;not null" json:"live_id"` // 直播间ID
	UserID  uint   `gorm:"index;not null" json:"user_id"`                  // 用户ID
	Content string `gorm:"type:text;not null" json:"content"`              // 弹幕内容

	// 弹幕样式
	Color    string `gorm:"size:20;default:'#FFFFFF'" json:"color"` // 弹幕颜色
	Position int8   `gorm:"default:1" json:"position"`              // 位置：1-滚动，2-顶部，3-底部
	FontSize int8   `gorm:"default:2" json:"font_size"`             // 字号：1-小，2-中，3-大

	// 特殊标记
	IsSticky  bool `gorm:"default:false" json:"is_sticky"`  // 是否置顶
	IsPinned  bool `gorm:"default:false" json:"is_pinned"`  // 是否精选
	IsDeleted bool `gorm:"default:false" json:"is_deleted"` // 是否被删除
	LikeCount int  `gorm:"default:0" json:"like_count"`     // 点赞数

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LiveComment) TableName() string {
	return "live_comments"
}

// ==================== 商品相关 ====================

// LiveProduct 直播间商品
type LiveProduct struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	LiveID      uint    `gorm:"index;not null" json:"live_id"`    // 直播间ID
	ProductID   uint    `gorm:"index;not null" json:"product_id"` // 商品ID（关联商品表）
	Name        string  `gorm:"size:200;not null" json:"name"`    // 商品名称
	Cover       string  `gorm:"size:512" json:"cover"`            // 商品封面
	Price       float64 `gorm:"not null" json:"price"`            // 原价
	SalePrice   float64 `gorm:"not null" json:"sale_price"`       // 直播价
	Stock       int     `gorm:"not null" json:"stock"`            // 库存
	SoldCount   int     `gorm:"default:0" json:"sold_count"`      // 已售数量
	Sort        int     `gorm:"default:0" json:"sort"`            // 排序
	Status      int8    `gorm:"default:1;index" json:"status"`    // 状态：0-下架，1-上架，2-售罄
	IsHot       bool    `gorm:"default:false" json:"is_hot"`      // 是否热卖
	Discount    int     `gorm:"default:0" json:"discount"`        // 折扣（百分比）
	Description string  `gorm:"type:text" json:"description"`     // 商品描述

	// 推广信息
	ExplainedAt    *time.Time `json:"explained_at"`                     // 讲解时间
	CommissionRate float64    `gorm:"default:0" json:"commission_rate"` // 佣金比例

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
}

// TableName 指定表名
func (LiveProduct) TableName() string {
	return "live_products"
}

// ==================== 管理相关 ====================

// LiveAdmin 直播间管理员
type LiveAdmin struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	LiveID uint `gorm:"index:idx_live_admin;not null" json:"live_id"` // 直播间ID
	UserID uint `gorm:"index:idx_live_admin;not null" json:"user_id"` // 管理员用户ID
	Role   int8 `gorm:"default:1" json:"role"`                        // 角色：1-普通管理员，2-超级管理员

	// 权限
	CanBan    bool `gorm:"default:true" json:"can_ban"`     // 可以禁言
	CanKick   bool `gorm:"default:true" json:"can_kick"`    // 可以踢人
	CanPin    bool `gorm:"default:true" json:"can_pin"`     // 可以置顶消息
	CanManage bool `gorm:"default:false" json:"can_manage"` // 可以管理其他管理员

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LiveAdmin) TableName() string {
	return "live_admins"
}

// LiveBan 直播间禁言记录
type LiveBan struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	LiveID     uint       `gorm:"index:idx_live_ban;not null" json:"live_id"` // 直播间ID
	UserID     uint       `gorm:"index:idx_live_ban;not null" json:"user_id"` // 被禁言用户ID
	OperatorID uint       `gorm:"index" json:"operator_id"`                   // 操作人ID
	Reason     string     `gorm:"size:200" json:"reason"`                     // 禁言原因
	Type       int8       `gorm:"default:1" json:"type"`                      // 类型：1-禁言，2-踢出，3-拉黑
	Duration   int        `gorm:"default:0" json:"duration"`                  // 禁言时长（分钟，0表示永久）
	ExpiredAt  *time.Time `json:"expired_at"`                                 // 过期时间
	Status     int8       `gorm:"default:1;index" json:"status"`              // 状态：0-已解除，1-生效中

	// 关联
	Live     *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User     *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Operator *User       `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName 指定表名
func (LiveBan) TableName() string {
	return "live_bans"
}

// ==================== 分享相关 ====================

// LiveShare 直播分享记录
type LiveShare struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	LiveID    uint   `gorm:"index;not null" json:"live_id"` // 直播间ID
	UserID    uint   `gorm:"index;not null" json:"user_id"` // 分享用户ID
	Platform  string `gorm:"size:50" json:"platform"`       // 分享平台：wechat, qq, weibo等
	ViewCount int    `gorm:"default:0" json:"view_count"`   // 通过分享进入的人数

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LiveShare) TableName() string {
	return "live_shares"
}

// ==================== 打赏榜 ====================

// LiveRankList 直播打赏榜
type LiveRankList struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	LiveID     uint  `gorm:"index:idx_live_rank;not null" json:"live_id"` // 直播间ID
	UserID     uint  `gorm:"index:idx_live_rank;not null" json:"user_id"` // 用户ID
	Rank       int   `gorm:"not null" json:"rank"`                        // 排名
	TotalValue int64 `gorm:"not null;index" json:"total_value"`           // 总贡献值
	GiftCount  int   `gorm:"default:0" json:"gift_count"`                 // 礼物数量

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LiveRankList) TableName() string {
	return "live_rank_lists"
}

// ==================== 粉丝团 ====================

// LiveFansClub 直播间粉丝团
type LiveFansClub struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	LiveID      uint   `gorm:"index:idx_live_fans;not null" json:"live_id"` // 直播间ID
	UserID      uint   `gorm:"index:idx_live_fans;not null" json:"user_id"` // 用户ID
	Level       int    `gorm:"default:1" json:"level"`                      // 粉丝等级：1-10级
	Experience  int64  `gorm:"default:0" json:"experience"`                 // 经验值
	BadgeName   string `gorm:"size:50" json:"badge_name"`                   // 徽章名称
	BadgeIcon   string `gorm:"size:255" json:"badge_icon"`                  // 徽章图标
	Privileges  string `gorm:"type:text" json:"privileges"`                 // 特权（JSON格式）
	IsActivated bool   `gorm:"default:true" json:"is_activated"`            // 是否激活

	// 关联
	Live *LiveStream `gorm:"foreignKey:LiveID" json:"live,omitempty"`
	User *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (LiveFansClub) TableName() string {
	return "live_fans_clubs"
}
