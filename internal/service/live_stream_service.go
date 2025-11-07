package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"microvibe-go/internal/config"
	"microvibe-go/internal/model"
	"microvibe-go/internal/repository"
	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CreateLiveStreamRequest 创建直播请求
type CreateLiveStreamRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	CategoryID  *uint  `json:"category_id" binding:"required"`
	Cover       string `json:"cover"`

	// 推流协议选择
	PushProtocol string `json:"push_protocol"` // rtmp, webrtc, srt (默认: rtmp)

	// 流类型配置（可选，使用默认配置如果未指定）
	StreamType   string `json:"stream_type"`   // video_only, audio_only, video_audio (默认: video_audio)
	VideoCodec   string `json:"video_codec"`   // h264, h265, vp8, vp9 (默认: h264)
	AudioCodec   string `json:"audio_codec"`   // aac, opus, mp3 (默认: aac)
	VideoBitrate int    `json:"video_bitrate"` // 视频码率 kbps (默认: 2500)
	AudioBitrate int    `json:"audio_bitrate"` // 音频码率 kbps (默认: 128)
	FrameRate    int    `json:"frame_rate"`    // 帧率 (默认: 30)
	Resolution   string `json:"resolution"`    // 360p, 480p, 720p, 1080p, 2k, 4k (默认: 720p)
}

// StartLiveStreamRequest 开始直播请求
type StartLiveStreamRequest struct {
	StreamKey string `json:"stream_key" binding:"required"`
}

// EndLiveStreamRequest 结束直播请求
type EndLiveStreamRequest struct {
	StreamKey string `json:"stream_key" binding:"required"`
}

// LiveStreamResponse 直播间响应
type LiveStreamResponse struct {
	*model.LiveStream
	IsOwner bool `json:"is_owner"` // 是否是房主
}

// BanUserRequest 禁言请求
type BanUserRequest struct {
	LiveID   uint   `json:"live_id" binding:"required"`
	UserID   uint   `json:"user_id" binding:"required"`
	Type     int8   `json:"type" binding:"required"` // 1-禁言，2-踢出，3-拉黑
	Duration int    `json:"duration"`                // 禁言时长（分钟，0表示永久）
	Reason   string `json:"reason"`
}

// LiveStreamService 直播服务接口
type LiveStreamService interface {
	// CreateLiveStream 创建直播间
	CreateLiveStream(ctx context.Context, userID uint, req *CreateLiveStreamRequest) (*model.LiveStream, error)

	// StartLiveStream 开始直播
	StartLiveStream(ctx context.Context, userID uint, streamKey string) error

	// EndLiveStream 结束直播
	EndLiveStream(ctx context.Context, userID uint, streamKey string) error

	// GetLiveStreamByID 根据ID获取直播间
	GetLiveStreamByID(ctx context.Context, id uint) (*model.LiveStream, error)

	// GetLiveStreamByRoomID 根据房间ID获取直播间
	GetLiveStreamByRoomID(ctx context.Context, roomID string) (*model.LiveStream, error)

	// GetMyLiveStream 获取用户自己的直播间
	GetMyLiveStream(ctx context.Context, userID uint) (*model.LiveStream, error)

	// ListLiveStreams 获取直播列表
	ListLiveStreams(ctx context.Context, status string, page, pageSize int) ([]*model.LiveStream, int64, error)

	// JoinLiveStream 加入直播间
	JoinLiveStream(ctx context.Context, roomID string, userID uint) error

	// LeaveLiveStream 离开直播间
	LeaveLiveStream(ctx context.Context, roomID string, userID uint) error

	// IncrementLike 增加点赞
	IncrementLike(ctx context.Context, id uint) error

	// DeleteLiveStream 删除直播间
	DeleteLiveStream(ctx context.Context, userID uint, id uint) error

	// BanUser 禁言用户
	BanUser(ctx context.Context, operatorID uint, req *BanUserRequest) error

	// UnbanUser 解除禁言
	UnbanUser(ctx context.Context, operatorID uint, liveID, userID uint) error

	// CheckBanned 检查用户是否被禁言
	CheckBanned(ctx context.Context, liveID, userID uint) (bool, error)

	// ListBans 获取禁言列表
	ListBans(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveBan, int64, error)

	// GetHotLiveStreams 获取热门直播间
	GetHotLiveStreams(ctx context.Context, limit int) ([]*model.LiveStream, error)

	// ListByCategory 根据分类获取直播间列表
	ListByCategory(ctx context.Context, categoryID uint, status string, page, pageSize int) ([]*model.LiveStream, int64, error)
}

type liveStreamServiceImpl struct {
	liveRepo repository.LiveStreamRepository
	banRepo  repository.LiveBanRepository
	cfg      *config.Config
}

// NewLiveStreamService 创建直播服务
func NewLiveStreamService(
	liveRepo repository.LiveStreamRepository,
	banRepo repository.LiveBanRepository,
	cfg *config.Config,
) LiveStreamService {
	return &liveStreamServiceImpl{
		liveRepo: liveRepo,
		banRepo:  banRepo,
		cfg:      cfg,
	}
}

// CreateLiveStream 创建直播间
func (s *liveStreamServiceImpl) CreateLiveStream(ctx context.Context, userID uint, req *CreateLiveStreamRequest) (*model.LiveStream, error) {
	// 检查用户是否已有进行中的直播间
	existingLive, err := s.liveRepo.FindByOwnerID(ctx, userID)
	if err == nil && existingLive != nil {
		logger.Warn("用户已有进行中的直播间", zap.Uint("user_id", userID), zap.Uint("live_id", existingLive.ID))
		return nil, errors.New("您已有进行中的直播间,请先结束后再创建新的直播")
	}

	// 生成唯一的 StreamKey 和 RoomID
	streamKey := generateStreamKey()
	roomID := generateRoomID()

	// 应用默认配置（如果请求中未指定）
	pushProtocol := getOrDefault(req.PushProtocol, s.cfg.Streaming.RTMPServer, "rtmp")
	streamType := getOrDefault(req.StreamType, s.cfg.Streaming.DefaultStreamType, "video_audio")
	videoCodec := getOrDefault(req.VideoCodec, s.cfg.Streaming.DefaultVideoCodec, "h264")
	audioCodec := getOrDefault(req.AudioCodec, s.cfg.Streaming.DefaultAudioCodec, "aac")
	videoBitrate := getOrDefaultInt(req.VideoBitrate, s.cfg.Streaming.DefaultVideoBitrate, 2500)
	audioBitrate := getOrDefaultInt(req.AudioBitrate, s.cfg.Streaming.DefaultAudioBitrate, 128)
	frameRate := getOrDefaultInt(req.FrameRate, s.cfg.Streaming.DefaultFrameRate, 30)
	resolution := getOrDefault(req.Resolution, s.cfg.Streaming.DefaultResolution, "720p")

	// 根据流类型设置 HasVideo 和 HasAudio 标志
	hasVideo := streamType == "video_only" || streamType == "video_audio"
	hasAudio := streamType == "audio_only" || streamType == "video_audio"

	// 生成流媒体 URL
	streamURL := generateStreamURL(s.cfg.Streaming.RTMPServer, streamKey)
	playURL := generatePlayURL(s.cfg.Streaming.HLSServer, streamKey)
	flvURL := generatePlayURL(s.cfg.Streaming.FLVServer, streamKey)
	rtmpURL := generatePlayURL(s.cfg.Streaming.RTMPPlayServer, streamKey)
	webrtcURL := generateWebRTCURL(roomID)

	liveStream := &model.LiveStream{
		// 基本信息
		Title:       req.Title,
		Description: req.Description,
		Cover:       req.Cover,
		Status:      "waiting",
		StreamKey:   streamKey,
		CategoryID:  req.CategoryID,
		RoomID:      roomID,
		OwnerID:     userID,

		// 推流配置
		StreamURL:    streamURL,
		PlayURL:      playURL,
		FlvURL:       flvURL,
		RtmpURL:      rtmpURL,
		WebRTCURL:    webrtcURL,
		PushProtocol: pushProtocol,

		// 流类型配置
		StreamType:   streamType,
		HasVideo:     hasVideo,
		HasAudio:     hasAudio,
		VideoCodec:   videoCodec,
		AudioCodec:   audioCodec,
		VideoBitrate: videoBitrate,
		AudioBitrate: audioBitrate,
		FrameRate:    frameRate,
		Resolution:   resolution,

		// 统计数据
		ViewCount:   0,
		LikeCount:   0,
		OnlineCount: 0,
	}

	// 🔥 新增：创建数据库记录
	if err := s.liveRepo.Create(ctx, liveStream); err != nil {
		logger.Error("创建直播间失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, errors.New("创建直播间失败")
	}

	// 🔥 新增：WebRTC 推流时，预创建 SFU 房间资源
	// 注意：这里不是创建会话，而是在 SFU 中预留房间资源
	// 实际的 WebRTC 会话在用户开始推流时通过信令服务创建
	if pushProtocol == "webrtc" {
		logger.Info("创建 WebRTC 直播间，推流密钥和房间ID已生成",
			zap.String("room_id", roomID),
			zap.String("stream_key", streamKey),
			zap.String("webrtc_url", webrtcURL))

		// 可选：如果你的 SFU 支持预创建房间，可以调用 SFU API
		// 例如：s.sfuClient.CreateRoom(ctx, roomID)
		// 但通常 SFU（如 Pion Ion）会在第一个 peer 加入时自动创建房间
	}

	// 🔥 新增：RTMP 推流时，可以配置 RTMP 服务器的推流认证
	if pushProtocol == "rtmp" {
		logger.Info("创建 RTMP 直播间，推流地址已生成",
			zap.String("stream_key", streamKey),
			zap.String("stream_url", streamURL),
			zap.String("play_url", playURL))

		// 可选：如果你的 RTMP 服务器（如 nginx-rtmp）支持动态认证
		// 可以将 streamKey 注册到 Redis 或认证服务器
		// 例如：s.registerRTMPAuth(ctx, streamKey, userID)
	}

	logger.Info("创建直播间成功",
		zap.Uint("live_id", liveStream.ID),
		zap.String("room_id", roomID),
		zap.String("stream_type", streamType),
		zap.String("push_protocol", pushProtocol),
		zap.Uint("owner_id", userID))

	// 重新查询以获取关联的 Owner 信息
	return s.liveRepo.FindByID(ctx, liveStream.ID)
}

// StartLiveStream 开始直播
func (s *liveStreamServiceImpl) StartLiveStream(ctx context.Context, userID uint, streamKey string) error {
	liveStream, err := s.liveRepo.FindByStreamKey(ctx, streamKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.String("stream_key", streamKey))
		return errors.New("查询直播间失败")
	}

	// 验证是否是房主
	if liveStream.OwnerID != userID {
		logger.Warn("非房主尝试开始直播", zap.Uint("user_id", userID), zap.Uint("owner_id", liveStream.OwnerID))
		return errors.New("无权限操作")
	}

	// 检查状态
	if liveStream.Status == "live" {
		return errors.New("直播已经开始")
	}

	if liveStream.Status == "ended" {
		return errors.New("直播已结束，无法重新开始")
	}

	// 更新状态
	now := time.Now()
	liveStream.Status = "live"
	liveStream.StartedAt = &now

	if err := s.liveRepo.Update(ctx, liveStream); err != nil {
		logger.Error("开始直播失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
		return errors.New("开始直播失败")
	}

	logger.Info("开始直播",
		zap.Uint("live_id", liveStream.ID),
		zap.String("room_id", liveStream.RoomID),
		zap.Uint("owner_id", userID))

	return nil
}

// EndLiveStream 结束直播
func (s *liveStreamServiceImpl) EndLiveStream(ctx context.Context, userID uint, streamKey string) error {
	liveStream, err := s.liveRepo.FindByStreamKey(ctx, streamKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.String("stream_key", streamKey))
		return errors.New("查询直播间失败")
	}

	// 验证是否是房主
	if liveStream.OwnerID != userID {
		logger.Warn("非房主尝试结束直播", zap.Uint("user_id", userID), zap.Uint("owner_id", liveStream.OwnerID))
		return errors.New("无权限操作")
	}

	// 检查状态
	if liveStream.Status == "ended" {
		return errors.New("直播已结束")
	}

	// 更新状态
	now := time.Now()
	liveStream.Status = "ended"
	liveStream.EndedAt = &now
	liveStream.OnlineCount = 0 // 重置在线人数

	if err := s.liveRepo.Update(ctx, liveStream); err != nil {
		logger.Error("结束直播失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
		return errors.New("结束直播失败")
	}

	logger.Info("结束直播",
		zap.Uint("live_id", liveStream.ID),
		zap.String("room_id", liveStream.RoomID),
		zap.Uint("owner_id", userID))

	return nil
}

// GetLiveStreamByID 根据ID获取直播间
func (s *liveStreamServiceImpl) GetLiveStreamByID(ctx context.Context, id uint) (*model.LiveStream, error) {
	liveStream, err := s.liveRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.Uint("id", id))
		return nil, errors.New("查询直播间失败")
	}

	return liveStream, nil
}

// GetLiveStreamByRoomID 根据房间ID获取直播间
func (s *liveStreamServiceImpl) GetLiveStreamByRoomID(ctx context.Context, roomID string) (*model.LiveStream, error) {
	liveStream, err := s.liveRepo.FindByRoomID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.String("room_id", roomID))
		return nil, errors.New("查询直播间失败")
	}

	return liveStream, nil
}

// GetMyLiveStream 获取用户自己的直播间
func (s *liveStreamServiceImpl) GetMyLiveStream(ctx context.Context, userID uint) (*model.LiveStream, error) {
	liveStream, err := s.liveRepo.FindByOwnerID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("您还没有创建直播间")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.Uint("user_id", userID))
		return nil, errors.New("查询直播间失败")
	}

	return liveStream, nil
}

// ListLiveStreams 获取直播列表
func (s *liveStreamServiceImpl) ListLiveStreams(ctx context.Context, status string, page, pageSize int) ([]*model.LiveStream, int64, error) {
	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	liveStreams, total, err := s.liveRepo.List(ctx, status, page, pageSize)
	if err != nil {
		logger.Error("查询直播列表失败", zap.Error(err), zap.String("status", status))
		return nil, 0, errors.New("查询直播列表失败")
	}

	logger.Info("查询直播列表成功",
		zap.String("status", status),
		zap.Int("page", page),
		zap.Int("size", pageSize),
		zap.Int64("total", total))

	return liveStreams, total, nil
}

// JoinLiveStream 加入直播间
func (s *liveStreamServiceImpl) JoinLiveStream(ctx context.Context, roomID string, userID uint) error {
	liveStream, err := s.liveRepo.FindByRoomID(ctx, roomID)
	if err != nil {
		return errors.New("直播间不存在")
	}

	if liveStream.Status != "live" {
		return errors.New("直播未开始")
	}

	// 增加在线人数
	newCount := liveStream.OnlineCount + 1
	if err := s.liveRepo.UpdateOnlineCount(ctx, liveStream.ID, newCount); err != nil {
		logger.Error("更新在线人数失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
	}

	// 更新总观看人数
	if err := s.liveRepo.UpdateViewCount(ctx, liveStream.ID, liveStream.ViewCount+1); err != nil {
		logger.Error("更新观看人数失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
	}

	logger.Info("用户加入直播间",
		zap.Uint("user_id", userID),
		zap.String("room_id", roomID),
		zap.Int("online_count", newCount))

	return nil
}

// LeaveLiveStream 离开直播间
func (s *liveStreamServiceImpl) LeaveLiveStream(ctx context.Context, roomID string, userID uint) error {
	liveStream, err := s.liveRepo.FindByRoomID(ctx, roomID)
	if err != nil {
		return errors.New("直播间不存在")
	}

	// 减少在线人数
	newCount := liveStream.OnlineCount - 1
	if newCount < 0 {
		newCount = 0
	}

	if err := s.liveRepo.UpdateOnlineCount(ctx, liveStream.ID, newCount); err != nil {
		logger.Error("更新在线人数失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
	}

	logger.Info("用户离开直播间",
		zap.Uint("user_id", userID),
		zap.String("room_id", roomID),
		zap.Int("online_count", newCount))

	return nil
}

// IncrementLike 增加点赞
func (s *liveStreamServiceImpl) IncrementLike(ctx context.Context, id uint) error {
	if err := s.liveRepo.IncrementLikeCount(ctx, id, 1); err != nil {
		logger.Error("增加点赞失败", zap.Error(err), zap.Uint("live_id", id))
		return errors.New("点赞失败")
	}

	logger.Info("点赞成功", zap.Uint("live_id", id))
	return nil
}

// DeleteLiveStream 删除直播间
func (s *liveStreamServiceImpl) DeleteLiveStream(ctx context.Context, userID uint, id uint) error {
	// 查询直播间
	liveStream, err := s.liveRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("直播间不存在")
		}
		return errors.New("查询直播间失败")
	}

	// 验证权限
	if liveStream.OwnerID != userID {
		return errors.New("无权限删除")
	}

	// 只能删除已结束的直播间
	if liveStream.Status != "ended" {
		return errors.New("只能删除已结束的直播间")
	}

	if err := s.liveRepo.Delete(ctx, id); err != nil {
		logger.Error("删除直播间失败", zap.Error(err), zap.Uint("live_id", id))
		return errors.New("删除直播间失败")
	}

	logger.Info("删除直播间成功", zap.Uint("live_id", id), zap.Uint("user_id", userID))
	return nil
}

// generateStreamKey 生成推流密钥
func generateStreamKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，使用时间戳作为备选方案
		return fmt.Sprintf("stream_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// generateRoomID 生成房间ID
func generateRoomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，使用时间戳作为备选方案
		return fmt.Sprintf("room_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// generateStreamURL 生成 RTMP 推流地址（用于 OBS）
// 格式: rtmp://server/app/streamKey
func generateStreamURL(rtmpServer, streamKey string) string {
	return fmt.Sprintf("%s/%s", rtmpServer, streamKey)
}

// generatePlayURL 生成播放地址
func generatePlayURL(server, streamKey string) string {
	return fmt.Sprintf("%s/%s", server, streamKey)
}

// generateWebRTCURL 生成 WebRTC 播放地址
func generateWebRTCURL(roomID string) string {
	return fmt.Sprintf("webrtc://room/%s", roomID)
}

// getOrDefault 获取字符串值或默认值
func getOrDefault(value, configDefault, fallback string) string {
	if value != "" {
		return value
	}
	if configDefault != "" {
		return configDefault
	}
	return fallback
}

// getOrDefaultInt 获取整数值或默认值
func getOrDefaultInt(value, configDefault, fallback int) int {
	if value > 0 {
		return value
	}
	if configDefault > 0 {
		return configDefault
	}
	return fallback
}

// BanUser 禁言用户
func (s *liveStreamServiceImpl) BanUser(ctx context.Context, operatorID uint, req *BanUserRequest) error {
	// 1. 查询直播间
	liveStream, err := s.liveRepo.FindByID(ctx, req.LiveID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("直播间不存在")
		}
		logger.Error("查询直播间失败", zap.Error(err), zap.Uint("live_id", req.LiveID))
		return errors.New("查询直播间失败")
	}

	// 2. 验证权限（只有主播可以禁言）
	if liveStream.OwnerID != operatorID {
		return errors.New("无权限操作")
	}

	// 3. 不能禁言自己
	if req.UserID == operatorID {
		return errors.New("不能禁言自己")
	}

	// 4. 检查是否已经被禁言
	isBanned, err := s.banRepo.CheckBanned(ctx, req.LiveID, req.UserID)
	if err != nil {
		logger.Error("检查禁言状态失败", zap.Error(err))
	}
	if isBanned {
		return errors.New("该用户已被禁言")
	}

	// 5. 计算过期时间
	var expiredAt *time.Time
	if req.Duration > 0 {
		t := time.Now().Add(time.Duration(req.Duration) * time.Minute)
		expiredAt = &t
	}

	// 6. 创建禁言记录
	ban := &model.LiveBan{
		LiveID:     req.LiveID,
		UserID:     req.UserID,
		OperatorID: operatorID,
		Reason:     req.Reason,
		Type:       req.Type,
		Duration:   req.Duration,
		ExpiredAt:  expiredAt,
		Status:     1,
	}

	if err := s.banRepo.Create(ctx, ban); err != nil {
		logger.Error("创建禁言记录失败", zap.Error(err))
		return errors.New("禁言失败")
	}

	logger.Info("禁言用户成功",
		zap.Uint("operator_id", operatorID),
		zap.Uint("live_id", req.LiveID),
		zap.Uint("user_id", req.UserID),
		zap.Int8("type", req.Type))

	return nil
}

// UnbanUser 解除禁言
func (s *liveStreamServiceImpl) UnbanUser(ctx context.Context, operatorID uint, liveID, userID uint) error {
	// 1. 查询直播间
	liveStream, err := s.liveRepo.FindByID(ctx, liveID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("直播间不存在")
		}
		return errors.New("查询直播间失败")
	}

	// 2. 验证权限
	if liveStream.OwnerID != operatorID {
		return errors.New("无权限操作")
	}

	// 3. 解除禁言
	if err := s.banRepo.UnbanUser(ctx, liveID, userID); err != nil {
		logger.Error("解除禁言失败", zap.Error(err))
		return errors.New("解除禁言失败")
	}

	logger.Info("解除禁言成功",
		zap.Uint("operator_id", operatorID),
		zap.Uint("live_id", liveID),
		zap.Uint("user_id", userID))

	return nil
}

// CheckBanned 检查用户是否被禁言
func (s *liveStreamServiceImpl) CheckBanned(ctx context.Context, liveID, userID uint) (bool, error) {
	isBanned, err := s.banRepo.CheckBanned(ctx, liveID, userID)
	if err != nil {
		logger.Error("检查禁言状态失败", zap.Error(err))
		return false, errors.New("检查禁言状态失败")
	}

	return isBanned, nil
}

// ListBans 获取禁言列表
func (s *liveStreamServiceImpl) ListBans(ctx context.Context, liveID uint, page, pageSize int) ([]*model.LiveBan, int64, error) {
	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	bans, total, err := s.banRepo.ListByLiveID(ctx, liveID, page, pageSize)
	if err != nil {
		logger.Error("查询禁言列表失败", zap.Error(err), zap.Uint("live_id", liveID))
		return nil, 0, errors.New("查询禁言列表失败")
	}

	logger.Info("查询禁言列表成功",
		zap.Uint("live_id", liveID),
		zap.Int("page", page),
		zap.Int("size", pageSize),
		zap.Int64("total", total))

	return bans, total, nil
}

// GetHotLiveStreams 获取热门直播间
func (s *liveStreamServiceImpl) GetHotLiveStreams(ctx context.Context, limit int) ([]*model.LiveStream, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	liveStreams, err := s.liveRepo.ListHotLiveStreams(ctx, limit)
	if err != nil {
		logger.Error("查询热门直播间失败", zap.Error(err))
		return nil, errors.New("查询热门直播间失败")
	}

	logger.Info("查询热门直播间成功", zap.Int("count", len(liveStreams)))
	return liveStreams, nil
}

// ListByCategory 根据分类获取直播间列表
func (s *liveStreamServiceImpl) ListByCategory(ctx context.Context, categoryID uint, status string, page, pageSize int) ([]*model.LiveStream, int64, error) {
	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	liveStreams, total, err := s.liveRepo.ListByCategory(ctx, categoryID, status, page, pageSize)
	if err != nil {
		logger.Error("查询分类直播列表失败", zap.Error(err), zap.Uint("category_id", categoryID))
		return nil, 0, errors.New("查询分类直播列表失败")
	}

	logger.Info("查询分类直播列表成功",
		zap.Uint("category_id", categoryID),
		zap.String("status", status),
		zap.Int("page", page),
		zap.Int("size", pageSize),
		zap.Int64("total", total))

	return liveStreams, total, nil
}

// ========== 流媒体服务集成辅助方法 ==========

// registerRTMPAuth 注册 RTMP 推流认证（可选）
// 如果你的 RTMP 服务器（如 nginx-rtmp）支持动态认证，可以实现这个方法
// 例如：将 streamKey 存储到 Redis，nginx-rtmp 通过 HTTP 回调验证
func (s *liveStreamServiceImpl) registerRTMPAuth(ctx context.Context, streamKey string, userID uint) error {
	// 示例实现：将推流密钥存储到 Redis，设置过期时间
	// 格式: rtmp:auth:{streamKey} -> {userID}:{expireTime}
	//
	// 你的 nginx-rtmp 配置中可以添加 HTTP 回调：
	// on_publish http://your-api/api/v1/live/verify-stream;
	//
	// 实现示例：
	// authKey := fmt.Sprintf("rtmp:auth:%s", streamKey)
	// authValue := fmt.Sprintf("%d:%d", userID, time.Now().Add(24*time.Hour).Unix())
	// err := s.redisClient.Set(ctx, authKey, authValue, 24*time.Hour).Err()
	// if err != nil {
	//     logger.Error("注册 RTMP 认证失败", zap.Error(err))
	//     return err
	// }

	logger.Info("RTMP 推流认证已注册（功能待实现）",
		zap.String("stream_key", streamKey),
		zap.Uint("user_id", userID))
	return nil
}

// VerifyRTMPStream 验证 RTMP 推流权限（供 nginx-rtmp on_publish 回调使用）
// 这个方法应该在 Handler 层暴露为 HTTP 端点
func (s *liveStreamServiceImpl) VerifyRTMPStream(ctx context.Context, streamKey string) (bool, uint, error) {
	// 查询数据库，验证 streamKey 是否存在且直播间状态为 waiting 或 live
	liveStream, err := s.liveRepo.FindByStreamKey(ctx, streamKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("无效的推流密钥", zap.String("stream_key", streamKey))
			return false, 0, nil
		}
		logger.Error("验证推流密钥失败", zap.Error(err))
		return false, 0, err
	}

	// 检查直播间状态
	if liveStream.Status != "waiting" && liveStream.Status != "live" {
		logger.Warn("直播间状态不允许推流",
			zap.String("stream_key", streamKey),
			zap.String("status", liveStream.Status))
		return false, 0, nil
	}

	logger.Info("RTMP 推流认证通过",
		zap.String("stream_key", streamKey),
		zap.Uint("owner_id", liveStream.OwnerID),
		zap.Uint("live_id", liveStream.ID))

	return true, liveStream.OwnerID, nil
}

// OnStreamPublish RTMP 推流开始回调（供 nginx-rtmp on_publish 使用）
func (s *liveStreamServiceImpl) OnStreamPublish(ctx context.Context, streamKey string) error {
	// 查找直播间
	liveStream, err := s.liveRepo.FindByStreamKey(ctx, streamKey)
	if err != nil {
		logger.Error("查找直播间失败", zap.Error(err), zap.String("stream_key", streamKey))
		return err
	}

	// 更新直播间状态为 live
	now := time.Now()
	liveStream.Status = "live"
	liveStream.StartedAt = &now

	if err := s.liveRepo.Update(ctx, liveStream); err != nil {
		logger.Error("更新直播间状态失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
		return err
	}

	logger.Info("直播推流已开始",
		zap.Uint("live_id", liveStream.ID),
		zap.String("stream_key", streamKey),
		zap.Uint("owner_id", liveStream.OwnerID))

	return nil
}

// OnStreamUnpublish RTMP 推流结束回调（供 nginx-rtmp on_publish_done 使用）
func (s *liveStreamServiceImpl) OnStreamUnpublish(ctx context.Context, streamKey string) error {
	// 查找直播间
	liveStream, err := s.liveRepo.FindByStreamKey(ctx, streamKey)
	if err != nil {
		logger.Error("查找直播间失败", zap.Error(err), zap.String("stream_key", streamKey))
		return err
	}

	// 计算直播时长
	now := time.Now()
	if liveStream.StartedAt != nil {
		duration := now.Sub(*liveStream.StartedAt).Seconds()
		liveStream.Duration = int64(duration)
	}

	// 更新直播间状态为 ended
	liveStream.Status = "ended"
	liveStream.EndedAt = &now

	if err := s.liveRepo.Update(ctx, liveStream); err != nil {
		logger.Error("更新直播间状态失败", zap.Error(err), zap.Uint("live_id", liveStream.ID))
		return err
	}

	logger.Info("直播推流已结束",
		zap.Uint("live_id", liveStream.ID),
		zap.String("stream_key", streamKey),
		zap.Int64("duration", liveStream.Duration))

	return nil
}
