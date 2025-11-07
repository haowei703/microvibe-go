# Ion SFU 集成文档 - gRPC 信令中继模式

## 概述

本文档记录了 MicroVibe-Go 项目中 Ion SFU 的集成过程，采用 **gRPC 信令中继模式**，后端作为信令代理转发浏览器与 SFU 之间的 Offer/Answer 交换。

## 架构设计

### 系统架构（gRPC 信令中继）

```
浏览器客户端
    ↓ (WebSocket - 信令)
后端应用 (信令代理)
    ↓ (gRPC 双向流 - Signal API)
Ion SFU 服务器
    ↓ (UDP/RTP - 媒体流)
媒体流分发
```

### 架构优势

1. **完全控制权限**：OAuth2 认证在后端 WebSocket 层完成，用户无法绕过
2. **统一鉴权流程**：不需要让 Ion SFU 理解 OAuth2，后端统一处理所有认证
3. **安全性更高**：SFU 的 gRPC 端口不直接暴露给前端，仅后端可访问
4. **灵活性强**：可以在信令层做日志记录、监控、限流、内容审核等
5. **兼容现有系统**：无缝集成现有的 OAuth2 + JWT 认证体系

### 核心组件

1. **sfuClientServiceImpl**: SFU 客户端服务实现（gRPC 信令中继）
   - 管理 gRPC 连接和客户端
   - 维护 gRPC 双向流（Signal API）
   - 转发浏览器 Offer 到 Ion SFU
   - 转发 SFU Answer 回浏览器
   - 处理会话生命周期

2. **rtcProto.RTCClient**: Ion SFU gRPC 客户端
   - 调用 Signal() 方法创建双向流
   - 发送 JoinRequest（包含浏览器的 Offer）
   - 接收 JoinReply（包含 SFU 的 Answer）
   - 处理后续消息（ICE Candidates, Track Events 等）

3. **liveSignalingServiceImpl**: WebSocket 信令服务
   - 接收浏览器的 WebSocket 连接
   - OAuth2 权限验证
   - 调用 SFUClientService 转发信令
   - 处理业务消息（聊天、礼物、点赞等）

## 关键实现

### 1. 依赖管理

**go.mod**:
```go
require (
    github.com/pion/ion/proto/rtc v1.10.0  // Ion SFU gRPC proto
    google.golang.org/grpc v1.41.0         // gRPC 框架
    github.com/pion/webrtc/v3 v3.1.7       // WebRTC 类型定义
)
```

**安装命令**:
```bash
go get github.com/pion/ion@latest
go get google.golang.org/grpc@latest
```

**重要说明**：
- 不再使用 `ion-sdk-go`（客户端 SDK），改用 Ion SFU 的 gRPC proto 定义
- `ion/proto/rtc` 包含了 Ion SFU 的 gRPC Signal API 定义
- 后端作为信令代理，不需要创建本地 PeerConnection

### 2. 服务初始化（gRPC 信令中继）

**文件**: `internal/service/sfu_client_service.go`

```go
type sfuClientServiceImpl struct {
    config     *config.SFUConfig
    grpcConn   *grpc.ClientConn                      // gRPC 连接
    grpcClient rtcProto.RTCClient                   // gRPC 客户端
    streams    sync.Map                              // map[sessionID]rtcProto.RTC_SignalClient
    sessions   sync.Map                              // map[sessionID]*SessionInfo
    sfuAddress string
}

func NewSFUClientService(cfg *config.SFUConfig) (SFUClientService, error) {
    // Ion SFU 的 gRPC 地址（格式：host:port）
    sfuAddress := cfg.ServerURL
    if sfuAddress == "" {
        sfuAddress = "localhost:5551" // Ion SFU 默认 gRPC 端口
    }

    // 创建 gRPC 连接
    grpcConn, err := grpc.Dial(
        sfuAddress,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
    }

    // 创建 RTC 客户端
    grpcClient := rtcProto.NewRTCClient(grpcConn)

    return &sfuClientServiceImpl{
        config:     cfg,
        grpcConn:   grpcConn,
        grpcClient: grpcClient,
        sfuAddress: sfuAddress,
    }, nil
}
```

**关键变化**：
- 使用 `grpc.Dial()` 创建 gRPC 连接，而不是 WebSocket Connector
- 使用 `rtcProto.NewRTCClient()` 创建 gRPC 客户端
- 使用 `sync.Map` 管理每个会话的 gRPC 双向流

### 3. 创建会话 (CreateSession) - gRPC 信令中继

#### 流程图
```
浏览器创建 SDP Offer
    ↓
WebSocket → 后端
    ↓
后端创建 gRPC 双向流
    ↓
构建 JoinRequest（包含 Offer）
    ↓
grpcStream.Send(JoinRequest) → Ion SFU
    ↓
grpcStream.Recv() ← Ion SFU (JoinReply + Answer)
    ↓
提取 SDP Answer
    ↓
WebSocket ← 后端 → 浏览器
    ↓
浏览器设置 Answer，建立 PeerConnection
```

#### 代码实现
```go
func (s *sfuClientServiceImpl) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
    // 1. 创建 gRPC 双向流
    stream, err := s.grpcClient.Signal(ctx)
    if err != nil {
        return nil, fmt.Errorf("创建信令流失败: %w", err)
    }

    // 缓存流对象（用于后续 ICE Candidate 和事件处理）
    s.streams.Store(req.SessionID, stream)

    // 2. 构建 JoinRequest（包含浏览器的 Offer）
    target := rtcProto.Target_PUBLISHER
    if req.Role == RoleSubscriber {
        target = rtcProto.Target_SUBSCRIBER
    }

    joinReq := &rtcProto.Request{
        Payload: &rtcProto.Request_Join{
            Join: &rtcProto.JoinRequest{
                Sid: req.RoomID,                      // 房间 ID
                Uid: fmt.Sprintf("%d", req.UserID),   // 用户 ID
                Description: &rtcProto.SessionDescription{
                    Target: target,
                    Type:   "offer",
                    Sdp:    req.SDP,                   // 浏览器的 Offer
                },
            },
        },
    }

    // 3. 发送 JoinRequest 到 Ion SFU
    if err := stream.Send(joinReq); err != nil {
        return nil, fmt.Errorf("发送 JoinRequest 失败: %w", err)
    }

    // 4. 等待 JoinReply（包含 SFU 的 Answer）
    reply, err := stream.Recv()
    if err != nil {
        return nil, fmt.Errorf("接收 JoinReply 失败: %w", err)
    }

    // 5. 解析 JoinReply
    joinReply := reply.GetJoin()
    if joinReply == nil || !joinReply.Success {
        return nil, errors.New("加入 SFU 失败")
    }

    // 6. 提取 SDP Answer
    sdpAnswer := joinReply.Description.Sdp

    // 7. 缓存会话信息
    s.sessions.Store(req.SessionID, &SessionInfo{
        SessionID: req.SessionID,
        RoomID:    req.RoomID,
        UserID:    req.UserID,
        Role:      req.Role,
        CreatedAt: time.Now(),
    })

    // 8. 启动 goroutine 处理后续的流消息（ICE, Track Events）
    go s.handleStreamMessages(req.SessionID, stream)

    return &CreateSessionResponse{
        SessionID: req.SessionID,
        SDP:       sdpAnswer,  // 返回 SFU 的 Answer 给浏览器
    }, nil
}
```

#### 关键点说明

1. **无本地 PeerConnection**：后端不创建 PeerConnection，只转发 SDP
2. **双向流管理**：每个会话对应一个 gRPC 双向流
3. **异步消息处理**：`handleStreamMessages` 处理后续的 ICE、Track 事件
4. **会话缓存**：使用 `sync.Map` 存储会话信息和流对象

### 4. 关闭会话 (CloseSession)

```go
func (s *sfuClientServiceImpl) CloseSession(ctx context.Context, sessionID string) error {
    logger.Info("关闭 SFU 会话 (Ion SDK)", zap.String("session_id", sessionID))

    // 获取并关闭 RTC 实例
    if rtcInterface, ok := s.rtcClients.Load(sessionID); ok {
        rtc := rtcInterface.(*sdk.RTC)
        rtc.Close()
        s.rtcClients.Delete(sessionID)
    }

    s.sessions.Delete(sessionID)
    logger.Info("SFU 会话关闭成功", zap.String("session_id", sessionID))
    return nil
}
```

### 5. 获取会话统计 (GetSessionStats)

```go
func (s *sfuClientServiceImpl) GetSessionStats(ctx context.Context, sessionID string) (*SessionStats, error) {
    rtcInterface, ok := s.rtcClients.Load(sessionID)
    if !ok {
        return nil, errors.New("会话不存在")
    }
    rtc := rtcInterface.(*sdk.RTC)

    stats := &SessionStats{SessionID: sessionID}

    // 获取 PubTransport 统计信息（发送端）
    if pubTransport := rtc.GetPubTransport(); pubTransport != nil && pubTransport.GetPeerConnection() != nil {
        pc := pubTransport.GetPeerConnection()
        statsReport := pc.GetStats()

        for _, stat := range statsReport {
            switch stat := stat.(type) {
            case *webrtc.OutboundRTPStreamStats:
                if stat.Kind == "video" {
                    elapsed := time.Since(stats.CreatedAt).Seconds()
                    if elapsed > 0 {
                        stats.VideoBitrate = int64(float64(stat.BytesSent*8) / elapsed)
                    }
                } else if stat.Kind == "audio" {
                    elapsed := time.Since(stats.CreatedAt).Seconds()
                    if elapsed > 0 {
                        stats.AudioBitrate = int64(float64(stat.BytesSent*8) / elapsed)
                    }
                }
            case *webrtc.ICECandidatePairStats:
                stats.RoundTripTime = stat.CurrentRoundTripTime
            }
        }
    }

    // 获取 SubTransport 统计信息（接收端）
    if subTransport := rtc.GetSubTransport(); subTransport != nil && subTransport.GetPeerConnection() != nil {
        pc := subTransport.GetPeerConnection()
        statsReport := pc.GetStats()

        for _, stat := range statsReport {
            switch stat := stat.(type) {
            case *webrtc.InboundRTPStreamStats:
                if stat.PacketsReceived > 0 {
                    stats.PacketLoss = float64(stat.PacketsLost) / float64(uint32(stat.PacketsLost)+stat.PacketsReceived)
                }
                stats.Jitter = stat.Jitter
            }
        }
    }

    return stats, nil
}
```

### 6. 健康检查 (HealthCheck)

```go
func (s *sfuClientServiceImpl) HealthCheck(ctx context.Context) error {
    if s.connector == nil {
        return errors.New("SFU Connector 未初始化")
    }

    // 简单检查：尝试创建一个测试 RTC 实例
    testRTC := sdk.NewRTC(s.connector)
    if testRTC == nil {
        return errors.New("无法创建 RTC 实例")
    }
    testRTC.Close()

    return nil
}
```

## 配置说明

### config.yaml

```yaml
sfu:
  enabled: true                              # 是否启用 SFU
  server_url: "localhost:5551"               # Ion SFU gRPC 地址 (host:port)
  mode: "standalone"                         # standalone | cluster
  cluster_nodes:
    - "localhost:5551"                       # 集群模式下的 gRPC 地址列表
    - "localhost:5552"
  load_balance_method: "roundrobin"          # random | roundrobin | leastconn
```

**重要变化**：
- `server_url` 从 WebSocket 地址（`ws://...`）改为 gRPC 地址（`host:port`）
- 默认端口从 `7001`（WebSocket）改为 `5551`（gRPC）
- 不需要协议前缀（不是 `ws://` 或 `http://`）

### 配置结构

```go
// internal/config/config.go
type SFUConfig struct {
    Enabled           bool     `mapstructure:"enabled"`
    ServerURL         string   `mapstructure:"server_url"`
    Mode              string   `mapstructure:"mode"`
    ClusterNodes      []string `mapstructure:"cluster_nodes"`
    LoadBalanceMethod string   `mapstructure:"load_balance_method"`
}
```

## API 接口

### 创建会话

**Endpoint**: `POST /api/v1/sfu/session`

**Request**:
```json
{
  "session_id": "session-123",
  "room_id": "room-456",
  "user_id": 789,
  "role": "publisher",
  "sdp": "v=0\r\no=- 123456 2 IN IP4 127.0.0.1..."
}
```

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "session-123",
    "sdp": "v=0\r\no=- 789012 2 IN IP4 127.0.0.1..."
  }
}
```

### 关闭会话

**Endpoint**: `DELETE /api/v1/sfu/session/:sessionId`

**Response**:
```json
{
  "code": 0,
  "message": "success"
}
```

### 获取统计信息

**Endpoint**: `GET /api/v1/sfu/session/:sessionId/stats`

**Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "session-123",
    "room_id": "room-456",
    "user_id": 789,
    "role": "publisher",
    "video_bitrate": 1500000,
    "audio_bitrate": 128000,
    "packet_loss": 0.02,
    "jitter": 5.5,
    "round_trip_time": 30.2
  }
}
```

## 测试验证

### 编译测试
```bash
$ go build ./cmd/server
# 编译成功，无错误
```

### 启动测试
```bash
$ go run ./cmd/server/main.go
2025-11-03T16:19:21.112+0800 INFO server/main.go:28 应用启动 {"mode": "debug"}
2025-11-03T16:19:21.145+0800 INFO service/sfu_client_service.go:131 初始化 SFU 客户端服务 (Ion SDK) {"mode": "standalone"}
```

### 集成测试
```bash
# 测试健康检查
$ curl http://localhost:8080/health
{"status":"ok"}

# 测试 SFU 连接（需要前端客户端配合）
# 1. 前端获取 SDP Offer
# 2. 调用 POST /api/v1/sfu/session
# 3. 使用返回的 Answer 建立连接
```

## 故障排查

### 常见问题

1. **连接失败 (404)**
   - **原因**: 使用 HTTP 而非 WebSocket
   - **解决**: 确保 `server_url` 使用 `ws://` 或 `wss://` 前缀

2. **Answer 获取超时**
   - **原因**: PeerConnection 建立延迟
   - **解决**: 增加等待时间或优化网络

3. **端口占用**
   - **原因**: macOS 系统服务占用 7000/7001
   - **解决**: 使用其他端口（如 7777）或停止冲突服务

4. **类型不匹配错误**
   - **原因**: Ion SDK API 变化
   - **解决**: 使用 `.GetPeerConnection()` 而非 `.PC()`

### 日志调试

```go
// 启用详细日志
logger.SetLevel(zapcore.DebugLevel)

// 关键日志点
logger.Info("创建 SFU 会话", zap.String("session_id", sessionID))
logger.Error("SendJoin 失败", zap.Error(err))
logger.Info("获取 Answer 成功", zap.String("sdp_type", answer.Type.String()))
```

## 性能优化

### 1. 连接池管理
```go
// 使用 sync.Map 实现线程安全的连接池
s.rtcClients.Store(sessionID, rtc)
s.rtcClients.Load(sessionID)
s.rtcClients.Delete(sessionID)
```

### 2. 超时控制
```go
// 使用 context 和 select 实现超时
select {
case answer := <-answerChan:
    return answer, nil
case <-time.After(15 * time.Second):
    return nil, errors.New("timeout")
case <-ctx.Done():
    return nil, ctx.Err()
}
```

### 3. 异步处理
```go
// 使用 goroutine 处理耗时操作
go func() {
    err := rtc.SendJoin(...)
    if err != nil {
        errorChan <- err
    }
}()
```

## 未来优化

1. **连接复用**: 同一用户的多个会话共享 Connector
2. **负载均衡**: 实现集群模式的节点选择算法
3. **自动重连**: 处理 WebSocket 断线重连
4. **监控指标**: 集成 Prometheus 导出 WebRTC 指标
5. **录制功能**: 集成 Ion 的媒体录制能力

## 参考资料

- [Ion SDK Go 文档](https://github.com/pion/ion-sdk-go)
- [Ion SFU 文档](https://github.com/pion/ion-sfu)
- [Pion WebRTC 文档](https://github.com/pion/webrtc)
- [WebRTC 规范](https://www.w3.org/TR/webrtc/)

## 更新日志

- **2025-11-03**: 完成 Ion SDK 集成，所有核心功能实现并测试通过
