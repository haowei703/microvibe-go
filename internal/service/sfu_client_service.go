package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"microvibe-go/internal/config"
	"microvibe-go/pkg/logger"
	"sync"
	"time"

	rtcProto "github.com/pion/ion/proto/rtc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SFUClientService SFU 客户端服务接口（调用 Pion Ion SFU）
type SFUClientService interface {
	// CreateSession 创建 SFU 会话（主播推流或观众拉流）
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error)
	// CloseSession 关闭 SFU 会话
	CloseSession(ctx context.Context, sessionID string) error
	// GetSessionStats 获取会话统计信息
	GetSessionStats(ctx context.Context, sessionID string) (*SessionStats, error)
	// UpdateQuality 更新视频质量设置
	UpdateQuality(ctx context.Context, sessionID string, quality QualityConfig) error
	// GetSFUInfo 获取 SFU 服务器信息
	GetSFUInfo(ctx context.Context) (*SFUInfo, error)
	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	SessionID    string                 `json:"session_id"` // 会话ID（通常是 roomID-userID）
	RoomID       string                 `json:"room_id"`    // 房间ID
	UserID       uint                   `json:"user_id"`    // 用户ID
	Role         SessionRole            `json:"role"`       // 角色：publisher 或 subscriber
	SDP          string                 `json:"sdp"`        // SDP Offer（客户端提供）
	Config       QualityConfig          `json:"config"`     // 质量配置
	OnICE        func(candidate string) `json:"-"`          // ICE Candidate 回调
	OnTrackEvent func(event string)     `json:"-"`          // Track Event 回调
}

// CreateSessionResponse 创建会话响应
type CreateSessionResponse struct {
	SessionID string `json:"session_id"` // 会话ID
	SDP       string `json:"sdp"`        // SDP Answer（SFU 返回）
}

// SessionRole 会话角色
type SessionRole string

const (
	RolePublisher  SessionRole = "publisher"  // 主播（推流）
	RoleSubscriber SessionRole = "subscriber" // 观众（拉流）
)

// QualityConfig 质量配置
type QualityConfig struct {
	VideoCodec      string `json:"video_codec"`      // 视频编解码器：VP8, VP9, H264
	AudioCodec      string `json:"audio_codec"`      // 音频编解码器：Opus
	VideoBitrate    int    `json:"video_bitrate"`    // 视频比特率 (kbps)
	AudioBitrate    int    `json:"audio_bitrate"`    // 音频比特率 (kbps)
	EnableSimulcast bool   `json:"enable_simulcast"` // 启用联播
	Layer           string `json:"layer,omitempty"`  // 订阅层：low, medium, high
}

// SessionStats 会话统计信息
type SessionStats struct {
	SessionID       string    `json:"session_id"`
	RoomID          string    `json:"room_id"`
	UserID          uint      `json:"user_id"`
	Role            string    `json:"role"`
	VideoBitrate    int64     `json:"video_bitrate"`   // bps
	AudioBitrate    int64     `json:"audio_bitrate"`   // bps
	PacketLoss      float64   `json:"packet_loss"`     // 丢包率 (%)
	Jitter          float64   `json:"jitter"`          // 抖动 (ms)
	RoundTripTime   float64   `json:"round_trip_time"` // RTT (ms)
	BytesReceived   uint64    `json:"bytes_received"`
	BytesSent       uint64    `json:"bytes_sent"`
	PacketsReceived uint64    `json:"packets_received"`
	PacketsSent     uint64    `json:"packets_sent"`
	CreatedAt       time.Time `json:"created_at"`
}

// SFUInfo SFU 服务器信息
type SFUInfo struct {
	Version        string    `json:"version"`
	ActiveSessions int       `json:"active_sessions"`
	TotalRooms     int       `json:"total_rooms"`
	TotalBandwidth int64     `json:"total_bandwidth"` // bps
	CPUUsage       float64   `json:"cpu_usage"`       // %
	MemoryUsage    int64     `json:"memory_usage"`    // bytes
	Uptime         int64     `json:"uptime"`          // seconds
	StartTime      time.Time `json:"start_time"`
}

// sfuClientServiceImpl SFU 客户端服务实现（使用 gRPC 信令中继）
type sfuClientServiceImpl struct {
	config *config.SFUConfig

	// gRPC 连接和客户端
	grpcConn   *grpc.ClientConn
	grpcClient rtcProto.RTCClient

	// gRPC 双向流管理
	streams sync.Map // map[sessionID]rtcProto.RTC_SignalClient

	// 会话管理（本地缓存）
	sessions sync.Map // map[sessionID]*SessionInfo

	// SFU 服务器地址
	sfuAddress string

	// 连接池和负载均衡
	sfuNodes    []string // SFU 集群节点列表
	currentNode int      // 当前使用的节点索引
	nodesMutex  sync.RWMutex
}

// SessionInfo 本地会话信息缓存
type SessionInfo struct {
	SessionID    string
	RoomID       string
	UserID       uint
	Role         SessionRole
	CreatedAt    time.Time
	OnICE        func(candidate string) // ICE Candidate 回调
	OnTrackEvent func(event string)     // Track Event 回调
}

// NewSFUClientService 创建 SFU 客户端服务实例（使用 gRPC 信令中继）
func NewSFUClientService(cfg *config.SFUConfig) (SFUClientService, error) {
	logger.Info("初始化 SFU 客户端服务 (gRPC 信令中继)", zap.String("mode", cfg.Mode))

	// 直接使用配置中的 gRPC 地址（格式：host:port）
	sfuAddress := cfg.ServerURL
	if sfuAddress == "" {
		sfuAddress = "localhost:5551" // Ion SFU 默认 gRPC 地址
	}

	// 如果配置了集群节点，使用第一个节点
	var sfuNodes []string
	if len(cfg.ClusterNodes) > 0 {
		sfuNodes = cfg.ClusterNodes
		sfuAddress = cfg.ClusterNodes[0]
	}

	// 创建 gRPC 连接
	grpcConn, err := grpc.Dial(
		sfuAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("创建 gRPC 连接失败", zap.Error(err))
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	// 创建 RTC 客户端
	grpcClient := rtcProto.NewRTCClient(grpcConn)

	service := &sfuClientServiceImpl{
		config:     cfg,
		grpcConn:   grpcConn,
		grpcClient: grpcClient,
		sfuAddress: sfuAddress,
		sfuNodes:   sfuNodes,
	}

	logger.Info("SFU 客户端服务初始化成功 (gRPC 信令中继)", zap.String("address", sfuAddress))
	return service, nil
}

// CreateSession 创建 SFU 会话（使用 gRPC 信令中继）
func (s *sfuClientServiceImpl) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	logger.Info("创建 SFU 会话 (gRPC 信令中继)",
		zap.String("session_id", req.SessionID),
		zap.String("room_id", req.RoomID),
		zap.Uint("user_id", req.UserID),
		zap.String("role", string(req.Role)))

	// 创建 gRPC 双向流
	stream, err := s.grpcClient.Signal(ctx)
	if err != nil {
		logger.Error("创建 gRPC 信令流失败", zap.Error(err))
		return nil, fmt.Errorf("创建信令流失败: %w", err)
	}

	// 缓存流对象
	s.streams.Store(req.SessionID, stream)

	// 构建 JoinRequest
	target := rtcProto.Target_PUBLISHER
	if req.Role == RoleSubscriber {
		target = rtcProto.Target_SUBSCRIBER
	}

	joinReq := &rtcProto.Request{
		Payload: &rtcProto.Request_Join{
			Join: &rtcProto.JoinRequest{
				Sid: req.RoomID,
				Uid: fmt.Sprintf("%d", req.UserID),
				Description: &rtcProto.SessionDescription{
					Target: target,
					Type:   "offer",
					Sdp:    req.SDP,
				},
			},
		},
	}

	// 发送 JoinRequest 到 Ion SFU
	if err := stream.Send(joinReq); err != nil {
		logger.Error("发送 JoinRequest 失败", zap.Error(err))
		return nil, fmt.Errorf("发送 JoinRequest 失败: %w", err)
	}

	logger.Debug("已发送 JoinRequest 到 Ion SFU",
		zap.String("room_id", req.RoomID),
		zap.String("user_id", fmt.Sprintf("%d", req.UserID)))

	// 等待 JoinReply
	reply, err := stream.Recv()
	if err != nil {
		logger.Error("接收 JoinReply 失败", zap.Error(err))
		return nil, fmt.Errorf("接收 JoinReply 失败: %w", err)
	}

	// 解析 JoinReply
	joinReply := reply.GetJoin()
	if joinReply == nil {
		logger.Error("JoinReply 为空", zap.Any("reply", reply))
		return nil, errors.New("JoinReply 为空")
	}

	if !joinReply.Success {
		errMsg := "加入 SFU 失败"
		if joinReply.Error != nil {
			errMsg = fmt.Sprintf("%s: [%d] %s", errMsg, joinReply.Error.Code, joinReply.Error.Reason)
		}
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// 提取 SDP Answer
	if joinReply.Description == nil || joinReply.Description.Sdp == "" {
		logger.Error("SDP Answer 为空")
		return nil, errors.New("SDP Answer 为空")
	}

	sdpAnswer := joinReply.Description.Sdp

	// 缓存会话信息（包括回调函数）
	sessionInfo := &SessionInfo{
		SessionID:    req.SessionID,
		RoomID:       req.RoomID,
		UserID:       req.UserID,
		Role:         req.Role,
		CreatedAt:    time.Now(),
		OnICE:        req.OnICE,
		OnTrackEvent: req.OnTrackEvent,
	}
	s.sessions.Store(req.SessionID, sessionInfo)

	logger.Info("SFU 会话创建成功 (gRPC 信令中继)",
		zap.String("session_id", req.SessionID),
		zap.Int("sdp_length", len(sdpAnswer)))

	// 启动 goroutine 处理后续的流消息（ICE candidates, track events 等）
	go s.handleStreamMessages(req.SessionID, stream)

	return &CreateSessionResponse{
		SessionID: req.SessionID,
		SDP:       sdpAnswer,
	}, nil
}

// handleStreamMessages 处理 gRPC 流的后续消息
func (s *sfuClientServiceImpl) handleStreamMessages(sessionID string, stream rtcProto.RTC_SignalClient) {
	// 获取会话信息（包含回调函数）
	sessionInfoInterface, ok := s.sessions.Load(sessionID)
	if !ok {
		logger.Warn("会话信息不存在", zap.String("session_id", sessionID))
		return
	}
	sessionInfo := sessionInfoInterface.(*SessionInfo)

	for {
		reply, err := stream.Recv()
		if err != nil {
			logger.Info("信令流已关闭",
				zap.String("session_id", sessionID),
				zap.Error(err))
			s.streams.Delete(sessionID)
			return
		}

		// 处理不同类型的消息
		switch payload := reply.Payload.(type) {
		case *rtcProto.Reply_Trickle:
			// 转发 ICE Candidate 给客户端
			candidateInit := payload.Trickle.Init
			logger.Debug("收到 ICE Candidate，转发给客户端",
				zap.String("session_id", sessionID),
				zap.String("candidate", candidateInit))

			if sessionInfo.OnICE != nil {
				sessionInfo.OnICE(candidateInit)
			}

		case *rtcProto.Reply_TrackEvent:
			// 转发 Track Event 给客户端
			trackEvent := payload.TrackEvent.State.String()
			logger.Info("收到 Track Event，转发给客户端",
				zap.String("session_id", sessionID),
				zap.String("state", trackEvent))

			if sessionInfo.OnTrackEvent != nil {
				sessionInfo.OnTrackEvent(trackEvent)
			}

		case *rtcProto.Reply_Error:
			logger.Error("收到 SFU 错误",
				zap.String("session_id", sessionID),
				zap.Int32("code", payload.Error.Code),
				zap.String("reason", payload.Error.Reason))

		case *rtcProto.Reply_Description:
			// SFU 发送的 SDP Description（通常用于重新协商或 subscriber 的 answer）
			description := payload.Description
			logger.Info("收到 SDP Description，转发给客户端",
				zap.String("session_id", sessionID),
				zap.String("type", description.Type),
				zap.Int("sdp_length", len(description.Sdp)))

			// 这通常是 subscriber 接收到的 offer，需要转发给客户端
			// 注意: 这个 Description 可能是 offer 或 answer，取决于角色
			if sessionInfo.OnTrackEvent != nil {
				// 使用 TrackEvent 回调通知上层有新的 SDP
				sessionInfo.OnTrackEvent(fmt.Sprintf("description:%s:%s", description.Type, description.Sdp))
			}

		default:
			logger.Debug("收到未知消息类型",
				zap.String("session_id", sessionID),
				zap.String("type", fmt.Sprintf("%T", payload)))
		}
	}
}

// CloseSession 关闭 SFU 会话
func (s *sfuClientServiceImpl) CloseSession(ctx context.Context, sessionID string) error {
	logger.Info("关闭 SFU 会话 (gRPC 信令中继)", zap.String("session_id", sessionID))

	// 获取并关闭 gRPC 流
	if streamInterface, ok := s.streams.Load(sessionID); ok {
		stream := streamInterface.(rtcProto.RTC_SignalClient)
		_ = stream.CloseSend()
		s.streams.Delete(sessionID)
	}

	// 删除本地缓存
	s.sessions.Delete(sessionID)

	logger.Info("SFU 会话关闭成功", zap.String("session_id", sessionID))
	return nil
}

// GetSessionStats 获取会话统计信息
func (s *sfuClientServiceImpl) GetSessionStats(ctx context.Context, sessionID string) (*SessionStats, error) {
	logger.Info("获取会话统计信息", zap.String("session_id", sessionID))

	// 从本地缓存获取会话信息
	sessionInfoInterface, ok := s.sessions.Load(sessionID)
	if !ok {
		return nil, errors.New("会话不存在")
	}

	sessionInfo := sessionInfoInterface.(*SessionInfo)

	// 初始化统计信息（gRPC 信令中继模式下，统计由 SFU 维护）
	stats := &SessionStats{
		SessionID: sessionID,
		RoomID:    sessionInfo.RoomID,
		UserID:    sessionInfo.UserID,
		Role:      string(sessionInfo.Role),
		CreatedAt: sessionInfo.CreatedAt,
	}

	// TODO: 通过 gRPC 调用 Ion SFU 获取实时统计信息

	logger.Info("统计信息获取成功", zap.String("session_id", sessionID))
	return stats, nil
}

// UpdateQuality 更新视频质量设置
func (s *sfuClientServiceImpl) UpdateQuality(ctx context.Context, sessionID string, quality QualityConfig) error {
	logger.Info("更新视频质量",
		zap.String("session_id", sessionID),
		zap.String("video_codec", quality.VideoCodec),
		zap.Int("video_bitrate", quality.VideoBitrate))

	// 从缓存获取流对象
	streamInterface, ok := s.streams.Load(sessionID)
	if !ok {
		return errors.New("会话不存在")
	}

	stream := streamInterface.(rtcProto.RTC_SignalClient)

	// 构建 SubscriptionRequest（用于控制订阅层级）
	if quality.Layer != "" {
		subReq := &rtcProto.Request{
			Payload: &rtcProto.Request_Subscription{
				Subscription: &rtcProto.SubscriptionRequest{
					Subscriptions: []*rtcProto.Subscription{
						{
							Layer: quality.Layer,
						},
					},
				},
			},
		}

		// 发送订阅请求
		if err := stream.Send(subReq); err != nil {
			logger.Error("发送订阅请求失败", zap.Error(err))
			return fmt.Errorf("更新质量失败: %w", err)
		}
	}

	logger.Info("视频质量更新成功", zap.String("session_id", sessionID))
	return nil
}

// GetSFUInfo 获取 SFU 服务器信息
func (s *sfuClientServiceImpl) GetSFUInfo(ctx context.Context) (*SFUInfo, error) {
	// 统计本地活跃会话数
	activeSessions := 0
	rooms := make(map[string]bool)

	s.sessions.Range(func(key, value interface{}) bool {
		activeSessions++
		sessionInfo := value.(*SessionInfo)
		rooms[sessionInfo.RoomID] = true
		return true
	})

	info := &SFUInfo{
		Version:        "Ion SFU (gRPC 信令中继)",
		ActiveSessions: activeSessions,
		TotalRooms:     len(rooms),
	}

	logger.Info("获取 SFU 信息",
		zap.Int("active_sessions", info.ActiveSessions),
		zap.Int("total_rooms", info.TotalRooms))

	return info, nil
}

// HealthCheck 健康检查
func (s *sfuClientServiceImpl) HealthCheck(ctx context.Context) error {
	// 检查 gRPC 连接是否可用
	if s.grpcConn == nil || s.grpcClient == nil {
		return errors.New("SFU gRPC 客户端未初始化")
	}

	// 尝试创建一个测试流来验证连接
	testStream, err := s.grpcClient.Signal(ctx)
	if err != nil {
		logger.Error("SFU 健康检查失败", zap.Error(err))
		return fmt.Errorf("SFU 不可用: %w", err)
	}
	_ = testStream.CloseSend()

	logger.Debug("SFU 健康检查成功", zap.String("address", s.sfuAddress))
	return nil
}

// selectSFUNode 选择 SFU 节点（负载均衡）
func (s *sfuClientServiceImpl) selectSFUNode() string {
	if len(s.sfuNodes) == 0 {
		return s.sfuAddress
	}

	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()

	// 使用轮询算法
	switch s.config.LoadBalanceMethod {
	case "roundrobin":
		node := s.sfuNodes[s.currentNode]
		s.currentNode = (s.currentNode + 1) % len(s.sfuNodes)
		return node
	case "random":
		// 简单随机选择
		idx := time.Now().UnixNano() % int64(len(s.sfuNodes))
		return s.sfuNodes[idx]
	default:
		// 默认使用第一个节点
		return s.sfuNodes[0]
	}
}

// MarshalJSON 自定义 JSON 序列化（用于日志）
func (s *SessionStats) MarshalJSON() ([]byte, error) {
	type Alias SessionStats
	return json.Marshal(&struct {
		*Alias
		VideoBitrateMbps float64 `json:"video_bitrate_mbps"`
		AudioBitrateKbps float64 `json:"audio_bitrate_kbps"`
	}{
		Alias:            (*Alias)(s),
		VideoBitrateMbps: float64(s.VideoBitrate) / 1_000_000,
		AudioBitrateKbps: float64(s.AudioBitrate) / 1_000,
	})
}
