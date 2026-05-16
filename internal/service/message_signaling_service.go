package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"microvibe-go/internal/config"
	"microvibe-go/pkg/logger"
	"microvibe-go/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MessageSignalingService 消息信令服务接口
type MessageSignalingService interface {
	// HandleWebSocket 处理 WebSocket 连接
	HandleWebSocket(c *gin.Context)
	// PushToUser 推送消息给指定用户
	PushToUser(userID uint, msgType string, payload interface{}) error
}

// clientInfo 客户端连接信息
type clientInfo struct {
	Conn    *websocket.Conn
	writeMu sync.Mutex
}

type messageSignalingServiceImpl struct {
	// clients 用户ID -> []*clientInfo (支持多端登录)
	clients      map[uint][]*clientInfo
	clientsMutex sync.RWMutex

	upgrader websocket.Upgrader
	config   *config.Config
}

// NewMessageSignalingService 创建消息信令服务
func NewMessageSignalingService(cfg *config.Config) MessageSignalingService {
	return &messageSignalingServiceImpl{
		clients: make(map[uint][]*clientInfo),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 简化处理，允许所有来源
			},
		},
		config: cfg,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (s *messageSignalingServiceImpl) HandleWebSocket(c *gin.Context) {
	// 从查询参数获取 token 并鉴权
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token is required"})
		return
	}

	claims, err := utils.ParseToken(token, s.config.JWT.Secret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID := claims.UserID

	// 升级连接
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket 升级失败", zap.Error(err), zap.Uint("user_id", userID))
		return
	}

	client := &clientInfo{
		Conn: conn,
	}

	// 注册客户端
	s.addClient(userID, client)
	logger.Info("用户已连接 WebSocket 消息中心", zap.Uint("user_id", userID))

	// 清理连接
	defer func() {
		s.removeClient(userID, client)
		conn.Close()
		logger.Info("用户已断开 WebSocket 消息中心", zap.Uint("user_id", userID))
	}()

	// 保持连接（读取心跳或关闭信号）
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// PushToUser 推送消息给指定用户
func (s *messageSignalingServiceImpl) PushToUser(userID uint, msgType string, payload interface{}) error {
	s.clientsMutex.RLock()
	clients, exists := s.clients[userID]
	s.clientsMutex.RUnlock()

	if !exists || len(clients) == 0 {
		return nil // 用户不在线，不报错
	}

	msg := map[string]interface{}{
		"type":    msgType,
		"payload": payload,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	// 推送给该用户的所有活跃连接
	for _, client := range clients {
		go func(c *clientInfo) {
			c.writeMu.Lock()
			defer c.writeMu.Unlock()
			if err := c.Conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				logger.Warn("推送消息失败", zap.Error(err), zap.Uint("user_id", userID))
			}
		}(client)
	}

	return nil
}

func (s *messageSignalingServiceImpl) addClient(userID uint, client *clientInfo) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()
	s.clients[userID] = append(s.clients[userID], client)
}

func (s *messageSignalingServiceImpl) removeClient(userID uint, client *clientInfo) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	clients := s.clients[userID]
	for i, c := range clients {
		if c == client {
			s.clients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(s.clients[userID]) == 0 {
		delete(s.clients, userID)
	}
}
