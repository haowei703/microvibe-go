package event

// 事件名称常量
const (
	// 用户相关事件
	EventUserRegistered = "user.registered" // 用户注册
	EventUserLoggedIn   = "user.logged_in"  // 用户登录
	EventUserUpdated    = "user.updated"    // 用户更新
	EventUserDeleted    = "user.deleted"    // 用户删除

	// 视频相关事件
	EventVideoUploaded  = "video.uploaded"  // 视频上传
	EventVideoPublished = "video.published" // 视频发布
	EventVideoDeleted   = "video.deleted"   // 视频删除
	EventVideoViewed    = "video.viewed"    // 视频观看

	// 互动相关事件
	EventVideoLiked     = "video.liked"     // 视频点赞
	EventVideoCommented = "video.commented" // 视频评论
	EventVideoShared    = "video.shared"    // 视频分享
	EventUserFollowed   = "user.followed"   // 用户关注
	EventUserUnfollowed = "user.unfollowed" // 用户取消关注

	// 直播相关事件
	EventLiveStreamCreated   = "live.stream.created"   // 直播间创建
	EventLiveStreamStarted   = "live.stream.started"   // 开始直播
	EventLiveStreamEnded     = "live.stream.ended"     // 结束直播
	EventLiveUserJoined      = "live.user.joined"      // 用户加入直播间
	EventLiveUserLeft        = "live.user.left"        // 用户离开直播间
	EventLiveLikeReceived    = "live.like.received"    // 收到点赞
	EventLiveGiftReceived    = "live.gift.received"    // 收到礼物
	EventLiveCommentReceived = "live.comment.received" // 收到评论
	EventLiveShareReceived   = "live.share.received"   // 收到分享

	// 系统相关事件
	EventSystemError   = "system.error"   // 系统错误
	EventSystemWarning = "system.warning" // 系统警告
)

// ========================================
// 用户事件
// ========================================

// UserRegisteredEvent 用户注册事件
type UserRegisteredEvent struct {
	*BaseEvent
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// NewUserRegisteredEvent 创建用户注册事件
func NewUserRegisteredEvent(userID uint, username, email string) *UserRegisteredEvent {
	return &UserRegisteredEvent{
		BaseEvent: NewBaseEvent(EventUserRegistered),
		UserID:    userID,
		Username:  username,
		Email:     email,
	}
}

// UserLoggedInEvent 用户登录事件
type UserLoggedInEvent struct {
	*BaseEvent
	UserID    uint   `json:"user_id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

// NewUserLoggedInEvent 创建用户登录事件
func NewUserLoggedInEvent(userID uint, ipAddress, userAgent string) *UserLoggedInEvent {
	return &UserLoggedInEvent{
		BaseEvent: NewBaseEvent(EventUserLoggedIn),
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
}

// UserUpdatedEvent 用户更新事件
type UserUpdatedEvent struct {
	*BaseEvent
	UserID        uint     `json:"user_id"`
	UpdatedFields []string `json:"updated_fields"`
}

// NewUserUpdatedEvent 创建用户更新事件
func NewUserUpdatedEvent(userID uint, updatedFields []string) *UserUpdatedEvent {
	return &UserUpdatedEvent{
		BaseEvent:     NewBaseEvent(EventUserUpdated),
		UserID:        userID,
		UpdatedFields: updatedFields,
	}
}

// UserDeletedEvent 用户删除事件
type UserDeletedEvent struct {
	*BaseEvent
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

// NewUserDeletedEvent 创建用户删除事件
func NewUserDeletedEvent(userID uint, username string) *UserDeletedEvent {
	return &UserDeletedEvent{
		BaseEvent: NewBaseEvent(EventUserDeleted),
		UserID:    userID,
		Username:  username,
	}
}

// ========================================
// 视频事件
// ========================================

// VideoUploadedEvent 视频上传事件
type VideoUploadedEvent struct {
	*BaseEvent
	VideoID  uint   `json:"video_id"`
	UserID   uint   `json:"user_id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"` // 视频时长（秒）
}

// NewVideoUploadedEvent 创建视频上传事件
func NewVideoUploadedEvent(videoID, userID uint, title string, duration int) *VideoUploadedEvent {
	return &VideoUploadedEvent{
		BaseEvent: NewBaseEvent(EventVideoUploaded),
		VideoID:   videoID,
		UserID:    userID,
		Title:     title,
		Duration:  duration,
	}
}

// VideoPublishedEvent 视频发布事件
type VideoPublishedEvent struct {
	*BaseEvent
	VideoID    uint   `json:"video_id"`
	UserID     uint   `json:"user_id"`
	Title      string `json:"title"`
	CategoryID uint   `json:"category_id"`
}

// NewVideoPublishedEvent 创建视频发布事件
func NewVideoPublishedEvent(videoID, userID uint, title string, categoryID uint) *VideoPublishedEvent {
	return &VideoPublishedEvent{
		BaseEvent:  NewBaseEvent(EventVideoPublished),
		VideoID:    videoID,
		UserID:     userID,
		Title:      title,
		CategoryID: categoryID,
	}
}

// VideoDeletedEvent 视频删除事件
type VideoDeletedEvent struct {
	*BaseEvent
	VideoID uint `json:"video_id"`
	UserID  uint `json:"user_id"`
}

// NewVideoDeletedEvent 创建视频删除事件
func NewVideoDeletedEvent(videoID, userID uint) *VideoDeletedEvent {
	return &VideoDeletedEvent{
		BaseEvent: NewBaseEvent(EventVideoDeleted),
		VideoID:   videoID,
		UserID:    userID,
	}
}

// VideoViewedEvent 视频观看事件
type VideoViewedEvent struct {
	*BaseEvent
	VideoID       uint   `json:"video_id"`
	UserID        uint   `json:"user_id"`
	WatchDuration int    `json:"watch_duration"` // 观看时长（秒）
	IPAddress     string `json:"ip_address"`
}

// NewVideoViewedEvent 创建视频观看事件
func NewVideoViewedEvent(videoID, userID uint, watchDuration int, ipAddress string) *VideoViewedEvent {
	return &VideoViewedEvent{
		BaseEvent:     NewBaseEvent(EventVideoViewed),
		VideoID:       videoID,
		UserID:        userID,
		WatchDuration: watchDuration,
		IPAddress:     ipAddress,
	}
}

// ========================================
// 互动事件
// ========================================

// VideoLikedEvent 视频点赞事件
type VideoLikedEvent struct {
	*BaseEvent
	VideoID uint `json:"video_id"`
	UserID  uint `json:"user_id"`
}

// NewVideoLikedEvent 创建视频点赞事件
func NewVideoLikedEvent(videoID, userID uint) *VideoLikedEvent {
	return &VideoLikedEvent{
		BaseEvent: NewBaseEvent(EventVideoLiked),
		VideoID:   videoID,
		UserID:    userID,
	}
}

// VideoCommentedEvent 视频评论事件
type VideoCommentedEvent struct {
	*BaseEvent
	VideoID   uint   `json:"video_id"`
	UserID    uint   `json:"user_id"`
	CommentID uint   `json:"comment_id"`
	Content   string `json:"content"`
}

// NewVideoCommentedEvent 创建视频评论事件
func NewVideoCommentedEvent(videoID, userID, commentID uint, content string) *VideoCommentedEvent {
	return &VideoCommentedEvent{
		BaseEvent: NewBaseEvent(EventVideoCommented),
		VideoID:   videoID,
		UserID:    userID,
		CommentID: commentID,
		Content:   content,
	}
}

// VideoSharedEvent 视频分享事件
type VideoSharedEvent struct {
	*BaseEvent
	VideoID  uint   `json:"video_id"`
	UserID   uint   `json:"user_id"`
	Platform string `json:"platform"` // 分享平台（微信、QQ等）
}

// NewVideoSharedEvent 创建视频分享事件
func NewVideoSharedEvent(videoID, userID uint, platform string) *VideoSharedEvent {
	return &VideoSharedEvent{
		BaseEvent: NewBaseEvent(EventVideoShared),
		VideoID:   videoID,
		UserID:    userID,
		Platform:  platform,
	}
}

// UserFollowedEvent 用户关注事件
type UserFollowedEvent struct {
	*BaseEvent
	FollowerID  uint `json:"follower_id"`  // 关注者ID
	FollowingID uint `json:"following_id"` // 被关注者ID
}

// NewUserFollowedEvent 创建用户关注事件
func NewUserFollowedEvent(followerID, followingID uint) *UserFollowedEvent {
	return &UserFollowedEvent{
		BaseEvent:   NewBaseEvent(EventUserFollowed),
		FollowerID:  followerID,
		FollowingID: followingID,
	}
}

// UserUnfollowedEvent 用户取消关注事件
type UserUnfollowedEvent struct {
	*BaseEvent
	FollowerID  uint `json:"follower_id"`  // 关注者ID
	FollowingID uint `json:"following_id"` // 被取消关注者ID
}

// NewUserUnfollowedEvent 创建用户取消关注事件
func NewUserUnfollowedEvent(followerID, followingID uint) *UserUnfollowedEvent {
	return &UserUnfollowedEvent{
		BaseEvent:   NewBaseEvent(EventUserUnfollowed),
		FollowerID:  followerID,
		FollowingID: followingID,
	}
}

// ========================================
// 系统事件
// ========================================

// SystemErrorEvent 系统错误事件
type SystemErrorEvent struct {
	*BaseEvent
	ErrorMessage string                 `json:"error_message"`
	ErrorStack   string                 `json:"error_stack"`
	Context      map[string]interface{} `json:"context"`
}

// NewSystemErrorEvent 创建系统错误事件
func NewSystemErrorEvent(errorMessage, errorStack string, context map[string]interface{}) *SystemErrorEvent {
	return &SystemErrorEvent{
		BaseEvent:    NewBaseEvent(EventSystemError),
		ErrorMessage: errorMessage,
		ErrorStack:   errorStack,
		Context:      context,
	}
}

// SystemWarningEvent 系统警告事件
type SystemWarningEvent struct {
	*BaseEvent
	WarningMessage string                 `json:"warning_message"`
	Context        map[string]interface{} `json:"context"`
}

// NewSystemWarningEvent 创建系统警告事件
func NewSystemWarningEvent(warningMessage string, context map[string]interface{}) *SystemWarningEvent {
	return &SystemWarningEvent{
		BaseEvent:      NewBaseEvent(EventSystemWarning),
		WarningMessage: warningMessage,
		Context:        context,
	}
}

// ========================================
// 直播事件
// ========================================

// LiveStreamCreatedEvent 直播间创建事件
type LiveStreamCreatedEvent struct {
	*BaseEvent
	LiveID     uint   `json:"live_id"`
	RoomID     string `json:"room_id"`
	OwnerID    uint   `json:"owner_id"`
	Title      string `json:"title"`
	StreamType string `json:"stream_type"`
}

// NewLiveStreamCreatedEvent 创建直播间创建事件
func NewLiveStreamCreatedEvent(liveID uint, roomID string, ownerID uint, title string, streamType string) *LiveStreamCreatedEvent {
	return &LiveStreamCreatedEvent{
		BaseEvent:  NewBaseEvent(EventLiveStreamCreated),
		LiveID:     liveID,
		RoomID:     roomID,
		OwnerID:    ownerID,
		Title:      title,
		StreamType: streamType,
	}
}

// LiveStreamStartedEvent 开始直播事件
type LiveStreamStartedEvent struct {
	*BaseEvent
	LiveID  uint   `json:"live_id"`
	RoomID  string `json:"room_id"`
	OwnerID uint   `json:"owner_id"`
}

// NewLiveStreamStartedEvent 创建开始直播事件
func NewLiveStreamStartedEvent(liveID uint, roomID string, ownerID uint) *LiveStreamStartedEvent {
	return &LiveStreamStartedEvent{
		BaseEvent: NewBaseEvent(EventLiveStreamStarted),
		LiveID:    liveID,
		RoomID:    roomID,
		OwnerID:   ownerID,
	}
}

// LiveStreamEndedEvent 结束直播事件
type LiveStreamEndedEvent struct {
	*BaseEvent
	LiveID    uint   `json:"live_id"`
	RoomID    string `json:"room_id"`
	OwnerID   uint   `json:"owner_id"`
	Duration  int64  `json:"duration"`   // 直播时长（秒）
	ViewCount int    `json:"view_count"` // 观看人数
	LikeCount int    `json:"like_count"` // 点赞数
	GiftValue int64  `json:"gift_value"` // 礼物价值
}

// NewLiveStreamEndedEvent 创建结束直播事件
func NewLiveStreamEndedEvent(liveID uint, roomID string, ownerID uint, duration int64, viewCount, likeCount int, giftValue int64) *LiveStreamEndedEvent {
	return &LiveStreamEndedEvent{
		BaseEvent: NewBaseEvent(EventLiveStreamEnded),
		LiveID:    liveID,
		RoomID:    roomID,
		OwnerID:   ownerID,
		Duration:  duration,
		ViewCount: viewCount,
		LikeCount: likeCount,
		GiftValue: giftValue,
	}
}

// LiveUserJoinedEvent 用户加入直播间事件
type LiveUserJoinedEvent struct {
	*BaseEvent
	LiveID   uint   `json:"live_id"`
	RoomID   string `json:"room_id"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

// NewLiveUserJoinedEvent 创建用户加入直播间事件
func NewLiveUserJoinedEvent(liveID uint, roomID string, userID uint, username string) *LiveUserJoinedEvent {
	return &LiveUserJoinedEvent{
		BaseEvent: NewBaseEvent(EventLiveUserJoined),
		LiveID:    liveID,
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
	}
}

// LiveUserLeftEvent 用户离开直播间事件
type LiveUserLeftEvent struct {
	*BaseEvent
	LiveID        uint   `json:"live_id"`
	RoomID        string `json:"room_id"`
	UserID        uint   `json:"user_id"`
	WatchDuration int64  `json:"watch_duration"` // 观看时长（秒）
}

// NewLiveUserLeftEvent 创建用户离开直播间事件
func NewLiveUserLeftEvent(liveID uint, roomID string, userID uint, watchDuration int64) *LiveUserLeftEvent {
	return &LiveUserLeftEvent{
		BaseEvent:     NewBaseEvent(EventLiveUserLeft),
		LiveID:        liveID,
		RoomID:        roomID,
		UserID:        userID,
		WatchDuration: watchDuration,
	}
}

// LiveLikeReceivedEvent 收到点赞事件
type LiveLikeReceivedEvent struct {
	*BaseEvent
	LiveID uint   `json:"live_id"`
	RoomID string `json:"room_id"`
	UserID uint   `json:"user_id"`
	Count  int    `json:"count"` // 点赞数量
}

// NewLiveLikeReceivedEvent 创建收到点赞事件
func NewLiveLikeReceivedEvent(liveID uint, roomID string, userID uint, count int) *LiveLikeReceivedEvent {
	return &LiveLikeReceivedEvent{
		BaseEvent: NewBaseEvent(EventLiveLikeReceived),
		LiveID:    liveID,
		RoomID:    roomID,
		UserID:    userID,
		Count:     count,
	}
}

// LiveGiftReceivedEvent 收到礼物事件
type LiveGiftReceivedEvent struct {
	*BaseEvent
	LiveID   uint   `json:"live_id"`
	RoomID   string `json:"room_id"`
	UserID   uint   `json:"user_id"`
	GiftID   uint   `json:"gift_id"`
	GiftName string `json:"gift_name"`
	Amount   int    `json:"amount"` // 礼物数量
	Value    int64  `json:"value"`  // 礼物价值
}

// NewLiveGiftReceivedEvent 创建收到礼物事件
func NewLiveGiftReceivedEvent(liveID uint, roomID string, userID uint, giftID uint, giftName string, amount int, value int64) *LiveGiftReceivedEvent {
	return &LiveGiftReceivedEvent{
		BaseEvent: NewBaseEvent(EventLiveGiftReceived),
		LiveID:    liveID,
		RoomID:    roomID,
		UserID:    userID,
		GiftID:    giftID,
		GiftName:  giftName,
		Amount:    amount,
		Value:     value,
	}
}

// LiveCommentReceivedEvent 收到评论事件
type LiveCommentReceivedEvent struct {
	*BaseEvent
	LiveID  uint   `json:"live_id"`
	RoomID  string `json:"room_id"`
	UserID  uint   `json:"user_id"`
	Content string `json:"content"`
}

// NewLiveCommentReceivedEvent 创建收到评论事件
func NewLiveCommentReceivedEvent(liveID uint, roomID string, userID uint, content string) *LiveCommentReceivedEvent {
	return &LiveCommentReceivedEvent{
		BaseEvent: NewBaseEvent(EventLiveCommentReceived),
		LiveID:    liveID,
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
	}
}

// LiveShareReceivedEvent 收到分享事件
type LiveShareReceivedEvent struct {
	*BaseEvent
	LiveID   uint   `json:"live_id"`
	RoomID   string `json:"room_id"`
	UserID   uint   `json:"user_id"`
	Platform string `json:"platform"` // 分享平台
}

// NewLiveShareReceivedEvent 创建收到分享事件
func NewLiveShareReceivedEvent(liveID uint, roomID string, userID uint, platform string) *LiveShareReceivedEvent {
	return &LiveShareReceivedEvent{
		BaseEvent: NewBaseEvent(EventLiveShareReceived),
		LiveID:    liveID,
		RoomID:    roomID,
		UserID:    userID,
		Platform:  platform,
	}
}
