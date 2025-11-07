package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"microvibe-go/pkg/event"
	"microvibe-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SignalingMessageType 信令消息类型
type SignalingMessageType string

const (
	// WebRTC 信令类型
	MessageTypeJoin   SignalingMessageType = "join"   // 加入房间
	MessageTypeLeave  SignalingMessageType = "leave"  // 离开房间
	MessageTypeOffer  SignalingMessageType = "offer"  // Offer (主播推流)
	MessageTypeAnswer SignalingMessageType = "answer" // Answer (观众拉流)
	MessageTypeICE    SignalingMessageType = "ice"    // ICE Candidate

	// 直播间消息类型
	MessageTypeChat SignalingMessageType = "chat" // 聊天消息
	MessageTypeLike SignalingMessageType = "like" // 点赞
	MessageTypeGift SignalingMessageType = "gift" // 送礼物

	// 系统消息类型
	MessageTypeUserJoined SignalingMessageType = "user_joined" // 用户加入通知
	MessageTypeUserLeft   SignalingMessageType = "user_left"   // 用户离开通知
	MessageTypeError      SignalingMessageType = "error"       // 错误消息
)

// SignalingMessage WebRTC 信令消息
type SignalingMessage struct {
	Type      SignalingMessageType `json:"type"`      // 消息类型
	RoomID    string               `json:"room_id"`   // 房间ID
	UserID    uint                 `json:"user_id"`   // 用户ID
	Username  string               `json:"username"`  // 用户名（可选）
	Payload   interface{}          `json:"payload"`   // 消息内容（SDP/ICE/Chat等）
	Timestamp int64                `json:"timestamp"` // 时间戳
}

// ChatPayload 聊天消息内容
type ChatPayload struct {
	Message string `json:"message"`
}

// GiftPayload 礼物消息内容
type GiftPayload struct {
	GiftID   uint   `json:"gift_id"`
	GiftName string `json:"gift_name"`
	Amount   int    `json:"amount"`
}

// Client WebSocket 客户端信息
type Client struct {
	Conn      *websocket.Conn
	UserID    uint
	Username  string
	RoomID    string
	Role      SessionRole // 角色：publisher 或 subscriber
	SessionID string      // SFU 会话ID
	JoinTime  time.Time   // 加入时间，用于计算观看时长
	writeMu   sync.Mutex  // WebSocket 写入锁，防止并发写入
}

// LiveSignalingService 信令服务接口
type LiveSignalingService interface {
	// HandleWebSocket 处理 WebSocket 连接
	HandleWebSocket(c *gin.Context)

	// BroadcastToRoom 广播消息到房间
	BroadcastToRoom(roomID string, message *SignalingMessage, excludeUserID uint)

	// GetRoomOnlineCount 获取房间在线人数
	GetRoomOnlineCount(roomID string) int

	// CloseRoom 关闭房间（踢出所有用户）
	CloseRoom(roomID string)
}

type liveSignalingServiceImpl struct {
	// rooms 房间映射 roomID -> []*Client
	rooms      map[string][]*Client
	roomsMutex sync.RWMutex

	// upgrader WebSocket 升级器
	upgrader websocket.Upgrader

	// liveService 直播业务服务（用于更新在线人数等）
	liveService LiveStreamService

	// sfuClient SFU 客户端服务（用于 WebRTC 流分发）
	sfuClient SFUClientService

	// enableSFU 是否启用 SFU
	enableSFU bool

	// eventBus 事件总线（用于发布直播事件）
	eventBus event.EventBus
}

// NewLiveSignalingService 创建信令服务
func NewLiveSignalingService(liveService LiveStreamService, sfuClient SFUClientService, enableSFU bool) LiveSignalingService {
	return &liveSignalingServiceImpl{
		rooms: make(map[string][]*Client),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// 生产环境应该检查 Origin
				return true
			},
		},
		liveService: liveService,
		sfuClient:   sfuClient,
		enableSFU:   enableSFU,
		eventBus:    event.GetGlobalEventBus(), // 使用全局事件总线
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (s *liveSignalingServiceImpl) HandleWebSocket(c *gin.Context) {
	// 从查询参数获取用户信息
	roomID := c.Query("room_id")
	userIDStr := c.Query("user_id")
	username := c.Query("username")
	roleStr := c.Query("role") // publisher 或 subscriber

	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room_id is required"})
		return
	}

	// 升级 HTTP 连接到 WebSocket
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket 升级失败", zap.Error(err))
		return
	}

	// 创建客户端对象
	var userID uint
	if userIDStr != "" {
		// 简单的字符串转 uint（生产环境应该使用更安全的方式）
		var id uint64
		_, _ = fmt.Sscanf(userIDStr, "%d", &id)
		userID = uint(id)
	}

	// 解析角色
	role := RoleSubscriber // 默认为观众
	if roleStr == "publisher" {
		role = RolePublisher
	}

	client := &Client{
		Conn:     conn,
		UserID:   userID,
		Username: username,
		RoomID:   roomID,
		Role:     role,
		JoinTime: time.Now(), // 记录加入时间
	}

	// 添加到房间
	s.addClient(client)

	// 发送欢迎消息
	welcomeMsg := &SignalingMessage{
		Type:      MessageTypeUserJoined,
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
		Timestamp: time.Now().Unix(),
	}
	s.sendToClient(client, welcomeMsg)

	// 广播用户加入消息（排除自己）
	s.BroadcastToRoom(roomID, welcomeMsg, userID)

	logger.Info("用户加入 WebSocket",
		zap.String("room_id", roomID),
		zap.Uint("user_id", userID),
		zap.String("username", username))

	// 发布用户加入事件（自动更新在线人数）
	if s.eventBus != nil {
		// 需要获取 LiveID，先从 liveService 获取
		if s.liveService != nil {
			if liveStream, err := s.liveService.GetLiveStreamByRoomID(c.Request.Context(), roomID); err == nil && liveStream != nil {
				joinEvent := event.NewLiveUserJoinedEvent(liveStream.ID, roomID, userID, username)
				_ = s.eventBus.PublishAsync(c.Request.Context(), joinEvent)
			}
		}
	}

	// 更新在线人数（保留原有调用，作为备用）
	if s.liveService != nil {
		_ = s.liveService.JoinLiveStream(c.Request.Context(), roomID, userID)
	}

	// 启动读取消息循环
	defer func() {
		// 如果启用 SFU 且有会话，关闭 SFU 会话
		if s.enableSFU && s.sfuClient != nil && client.SessionID != "" {
			ctx := context.Background()
			if err := s.sfuClient.CloseSession(ctx, client.SessionID); err != nil {
				logger.Error("关闭 SFU 会话失败",
					zap.Error(err),
					zap.String("session_id", client.SessionID))
			} else {
				logger.Info("SFU 会话已关闭", zap.String("session_id", client.SessionID))
			}
		}

		s.removeClient(client)
		conn.Close()

		// 发布用户离开事件（自动更新在线人数）
		if s.eventBus != nil {
			if s.liveService != nil {
				if liveStream, err := s.liveService.GetLiveStreamByRoomID(context.Background(), roomID); err == nil && liveStream != nil {
					// 计算观看时长（秒）
					watchDuration := int64(time.Since(client.JoinTime).Seconds())
					leaveEvent := event.NewLiveUserLeftEvent(liveStream.ID, roomID, userID, watchDuration)
					_ = s.eventBus.PublishAsync(context.Background(), leaveEvent)
				}
			}
		}

		// 更新在线人数（保留原有调用，作为备用）
		if s.liveService != nil {
			_ = s.liveService.LeaveLiveStream(c.Request.Context(), roomID, userID)
		}

		// 广播用户离开消息
		leaveMsg := &SignalingMessage{
			Type:      MessageTypeUserLeft,
			RoomID:    roomID,
			UserID:    userID,
			Username:  username,
			Timestamp: time.Now().Unix(),
		}
		s.BroadcastToRoom(roomID, leaveMsg, 0)

		logger.Info("用户离开 WebSocket",
			zap.String("room_id", roomID),
			zap.Uint("user_id", userID),
			zap.String("role", string(role)))
	}()

	// 读取消息循环
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("WebSocket 异常关闭", zap.Error(err))
			}
			break
		}

		var msg SignalingMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			logger.Error("解析信令消息失败", zap.Error(err), zap.String("message", string(msgBytes)))
			s.sendError(client, "Invalid message format")
			continue
		}

		// 填充消息信息
		msg.RoomID = roomID
		msg.UserID = userID
		msg.Username = username
		msg.Timestamp = time.Now().Unix()

		// 处理不同类型的消息
		s.handleMessage(client, &msg)
	}
}

// handleMessage 处理信令消息
func (s *liveSignalingServiceImpl) handleMessage(client *Client, msg *SignalingMessage) {
	logger.Debug("收到信令消息",
		zap.String("type", string(msg.Type)),
		zap.String("room_id", msg.RoomID),
		zap.Uint("user_id", msg.UserID),
		zap.String("role", string(client.Role)))

	switch msg.Type {
	case MessageTypeOffer:
		// Offer 消息：主播推流 or 观众拉流
		s.handleOffer(client, msg)

	case MessageTypeAnswer:
		// Answer 消息（仅在非 SFU 模式下使用）
		if !s.enableSFU {
			s.BroadcastToRoom(msg.RoomID, msg, msg.UserID)
		}

	case MessageTypeICE:
		// ICE Candidate 消息
		s.handleICECandidate(client, msg)

	case MessageTypeChat:
		// 聊天消息，广播给所有人（包括自己）
		s.BroadcastToRoom(msg.RoomID, msg, 0)

		// 发布评论事件
		if s.eventBus != nil && s.liveService != nil {
			if liveStream, err := s.liveService.GetLiveStreamByRoomID(context.Background(), msg.RoomID); err == nil && liveStream != nil {
				// 提取聊天内容
				content := ""
				if payload, ok := msg.Payload.(map[string]interface{}); ok {
					if message, exists := payload["message"]; exists {
						if messageStr, ok := message.(string); ok {
							content = messageStr
						}
					}
				} else if payloadStr, ok := msg.Payload.(string); ok {
					content = payloadStr
				}

				if content != "" {
					commentEvent := event.NewLiveCommentReceivedEvent(
						liveStream.ID,
						msg.RoomID,
						msg.UserID,
						content,
					)
					_ = s.eventBus.PublishAsync(context.Background(), commentEvent)
				}
			}
		}

	case MessageTypeLike:
		// 点赞消息，广播给所有人
		s.BroadcastToRoom(msg.RoomID, msg, 0)

		// 发布点赞事件（自动增加点赞数）
		if s.eventBus != nil && s.liveService != nil {
			if liveStream, err := s.liveService.GetLiveStreamByRoomID(context.Background(), msg.RoomID); err == nil && liveStream != nil {
				// 默认点赞数为 1，如果 payload 中有 count，则使用 count
				count := 1
				if payload, ok := msg.Payload.(map[string]interface{}); ok {
					if c, exists := payload["count"]; exists {
						if countFloat, ok := c.(float64); ok {
							count = int(countFloat)
						} else if countInt, ok := c.(int); ok {
							count = countInt
						}
					}
				}

				likeEvent := event.NewLiveLikeReceivedEvent(
					liveStream.ID,
					msg.RoomID,
					msg.UserID,
					count,
				)
				_ = s.eventBus.PublishAsync(context.Background(), likeEvent)
			}
		}

	case MessageTypeGift:
		// 礼物消息，广播给所有人
		s.BroadcastToRoom(msg.RoomID, msg, 0)

		// 发布礼物事件（自动记录礼物）
		if s.eventBus != nil && s.liveService != nil {
			if liveStream, err := s.liveService.GetLiveStreamByRoomID(context.Background(), msg.RoomID); err == nil && liveStream != nil {
				// 从 payload 中提取礼物信息
				var giftID uint
				var giftName string
				var count int
				var value int64

				if payload, ok := msg.Payload.(map[string]interface{}); ok {
					if id, exists := payload["gift_id"]; exists {
						if idFloat, ok := id.(float64); ok {
							giftID = uint(idFloat)
						}
					}
					if name, exists := payload["gift_name"]; exists {
						if nameStr, ok := name.(string); ok {
							giftName = nameStr
						}
					}
					if c, exists := payload["count"]; exists {
						if countFloat, ok := c.(float64); ok {
							count = int(countFloat)
						} else if countInt, ok := c.(int); ok {
							count = countInt
						}
					}
					if v, exists := payload["value"]; exists {
						if valueFloat, ok := v.(float64); ok {
							value = int64(valueFloat)
						} else if valueInt, ok := v.(int64); ok {
							value = valueInt
						}
					}
				}

				if giftID > 0 && count > 0 {
					giftEvent := event.NewLiveGiftReceivedEvent(
						liveStream.ID,
						msg.RoomID,
						msg.UserID,
						giftID,
						giftName,
						count,
						value,
					)
					_ = s.eventBus.PublishAsync(context.Background(), giftEvent)
				}
			}
		}

	case MessageTypeLeave:
		// 主动离开，关闭连接
		s.handleLeave(client)

	default:
		logger.Warn("未知的信令消息类型", zap.String("type", string(msg.Type)))
		s.sendError(client, "Unknown message type")
	}
}

// handleOffer 处理 Offer 消息（通过 SFU）
func (s *liveSignalingServiceImpl) handleOffer(client *Client, msg *SignalingMessage) {
	// 如果未启用 SFU，使用传统的 P2P 转发
	if !s.enableSFU || s.sfuClient == nil {
		logger.Info("SFU 未启用，使用 P2P 转发")
		s.BroadcastToRoom(msg.RoomID, msg, msg.UserID)
		return
	}

	// 提取 SDP Offer
	var sdpOffer string

	// 尝试直接转换为字符串
	if sdp, ok := msg.Payload.(string); ok {
		sdpOffer = sdp
	} else if payloadMap, ok := msg.Payload.(map[string]interface{}); ok {
		// 如果是对象，尝试提取 sdp 字段
		if sdp, exists := payloadMap["sdp"]; exists {
			if sdpStr, ok := sdp.(string); ok {
				sdpOffer = sdpStr
			}
		}
	}

	if sdpOffer == "" {
		logger.Error("无效的 SDP Offer 格式", zap.Any("payload", msg.Payload))
		s.sendError(client, "Invalid SDP offer format")
		return
	}

	// 生成 SFU 会话ID
	sessionID := fmt.Sprintf("%s-%d", client.RoomID, client.UserID)
	client.SessionID = sessionID

	// 构建 SFU 请求（带回调函数）
	sfuReq := &CreateSessionRequest{
		SessionID: sessionID,
		RoomID:    client.RoomID,
		UserID:    client.UserID,
		Role:      client.Role,
		SDP:       sdpOffer,
		Config: QualityConfig{
			VideoBitrate: 2500,
			AudioBitrate: 128,
		},
		// ICE Candidate 回调：转发给 WebSocket 客户端
		OnICE: func(candidate string) {
			// candidate 是 JSON 字符串，需要解析为对象避免双重编码
			var candidateObj map[string]interface{}
			if err := json.Unmarshal([]byte(candidate), &candidateObj); err != nil {
				logger.Error("解析 ICE Candidate 失败",
					zap.Error(err),
					zap.String("candidate", candidate))
				return
			}

			iceMsg := &SignalingMessage{
				Type:      MessageTypeICE,
				RoomID:    client.RoomID,
				UserID:    client.UserID,
				Payload:   candidateObj, // 使用解析后的对象
				Timestamp: time.Now().Unix(),
			}
			s.sendToClient(client, iceMsg)
		},
		// Track Event 回调：记录日志
		OnTrackEvent: func(event string) {
			logger.Info("Track Event",
				zap.String("session_id", sessionID),
				zap.String("event", event))
		},
	}

	// 调用 SFU 创建会话
	ctx := context.Background()
	sfuResp, err := s.sfuClient.CreateSession(ctx, sfuReq)
	if err != nil {
		logger.Error("SFU 创建会话失败",
			zap.Error(err),
			zap.String("session_id", sessionID),
			zap.String("room_id", client.RoomID))
		s.sendError(client, fmt.Sprintf("Failed to create SFU session: %v", err))
		return
	}

	logger.Info("SFU 会话创建成功",
		zap.String("session_id", sessionID),
		zap.String("room_id", client.RoomID),
		zap.String("role", string(client.Role)))

	// 发送 SDP Answer 给客户端
	answerMsg := &SignalingMessage{
		Type:      MessageTypeAnswer,
		RoomID:    client.RoomID,
		UserID:    client.UserID,
		Payload:   sfuResp.SDP,
		Timestamp: time.Now().Unix(),
	}
	s.sendToClient(client, answerMsg)
}

// handleICECandidate 处理 ICE Candidate
func (s *liveSignalingServiceImpl) handleICECandidate(client *Client, msg *SignalingMessage) {
	// 如果启用 SFU，ICE Candidate 应该由 SFU 自动处理
	// 这里可以选择转发给 SFU（取决于 SFU 实现）
	if s.enableSFU && s.sfuClient != nil {
		// Pion Ion SFU 通常在 Offer/Answer 交换时已经包含 ICE 信息
		// 这里记录日志即可
		logger.Debug("收到 ICE Candidate (SFU 模式)",
			zap.String("session_id", client.SessionID))
		return
	}

	// 非 SFU 模式：转发 ICE Candidate
	s.BroadcastToRoom(msg.RoomID, msg, msg.UserID)
}

// handleLeave 处理离开消息
func (s *liveSignalingServiceImpl) handleLeave(client *Client) {
	// 如果启用 SFU 且有会话，关闭 SFU 会话
	if s.enableSFU && s.sfuClient != nil && client.SessionID != "" {
		ctx := context.Background()
		if err := s.sfuClient.CloseSession(ctx, client.SessionID); err != nil {
			logger.Error("关闭 SFU 会话失败",
				zap.Error(err),
				zap.String("session_id", client.SessionID))
		} else {
			logger.Info("SFU 会话已关闭", zap.String("session_id", client.SessionID))
		}
	}

	// 关闭 WebSocket 连接
	client.Conn.Close()
}

// addClient 添加客户端到房间
func (s *liveSignalingServiceImpl) addClient(client *Client) {
	s.roomsMutex.Lock()
	defer s.roomsMutex.Unlock()

	if s.rooms[client.RoomID] == nil {
		s.rooms[client.RoomID] = make([]*Client, 0)
	}
	s.rooms[client.RoomID] = append(s.rooms[client.RoomID], client)
}

// removeClient 从房间移除客户端
func (s *liveSignalingServiceImpl) removeClient(client *Client) {
	s.roomsMutex.Lock()
	defer s.roomsMutex.Unlock()

	clients := s.rooms[client.RoomID]
	for i, c := range clients {
		if c == client {
			// 移除客户端
			s.rooms[client.RoomID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	// 如果房间为空，删除房间
	if len(s.rooms[client.RoomID]) == 0 {
		delete(s.rooms, client.RoomID)
	}
}

// BroadcastToRoom 广播消息到房间
func (s *liveSignalingServiceImpl) BroadcastToRoom(roomID string, message *SignalingMessage, excludeUserID uint) {
	s.roomsMutex.RLock()
	defer s.roomsMutex.RUnlock()

	clients := s.rooms[roomID]
	if clients == nil {
		return
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	// 广播给房间内所有客户端（排除指定用户）
	for _, client := range clients {
		if excludeUserID != 0 && client.UserID == excludeUserID {
			continue
		}

		if err := client.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			logger.Error("发送消息失败",
				zap.Error(err),
				zap.Uint("user_id", client.UserID),
				zap.String("room_id", roomID))
		}
	}
}

// sendToClient 发送消息给指定客户端
func (s *liveSignalingServiceImpl) sendToClient(client *Client, message *SignalingMessage) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		logger.Error("序列化消息失败", zap.Error(err))
		return
	}

	// 使用互斥锁保护 WebSocket 写入，防止并发写入导致 panic
	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	if err := client.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		logger.Error("发送消息失败", zap.Error(err), zap.Uint("user_id", client.UserID))
	}
}

// sendError 发送错误消息给客户端
func (s *liveSignalingServiceImpl) sendError(client *Client, errorMsg string) {
	msg := &SignalingMessage{
		Type:      MessageTypeError,
		RoomID:    client.RoomID,
		Payload:   map[string]string{"error": errorMsg},
		Timestamp: time.Now().Unix(),
	}
	s.sendToClient(client, msg)
}

// GetRoomOnlineCount 获取房间在线人数
func (s *liveSignalingServiceImpl) GetRoomOnlineCount(roomID string) int {
	s.roomsMutex.RLock()
	defer s.roomsMutex.RUnlock()

	return len(s.rooms[roomID])
}

// CloseRoom 关闭房间（踢出所有用户）
func (s *liveSignalingServiceImpl) CloseRoom(roomID string) {
	s.roomsMutex.Lock()
	defer s.roomsMutex.Unlock()

	clients := s.rooms[roomID]
	if clients == nil {
		return
	}

	// 发送房间关闭消息
	closeMsg := &SignalingMessage{
		Type:      MessageTypeError,
		RoomID:    roomID,
		Payload:   map[string]string{"error": "Room closed"},
		Timestamp: time.Now().Unix(),
	}

	msgBytes, _ := json.Marshal(closeMsg)

	// 关闭所有连接
	for _, client := range clients {
		_ = client.Conn.WriteMessage(websocket.TextMessage, msgBytes)
		client.Conn.Close()
	}

	// 删除房间
	delete(s.rooms, roomID)

	logger.Info("房间已关闭", zap.String("room_id", roomID))
}
