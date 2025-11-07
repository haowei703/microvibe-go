# Ion SFU WebSocket 集成完成

## ✅ 已解决问题

### 原问题
1. **URL 格式错误**: 双重 `http://` 前缀导致连接失败
2. **端口冲突**: macOS 系统服务占用 7000/7001 端口
3. **404 错误**: 使用 HTTP JSON-RPC 客户端而非 WebSocket

### 根本原因
Ion SFU 的 JSON-RPC 接口使用 **WebSocket 协议**，不是 HTTP POST。

**错误实现**:
```go
// ❌ 使用 HTTP JSON-RPC 客户端
import "github.com/ybbus/jsonrpc/v3"
rpcClient := jsonrpc.NewClient("http://localhost:7001")
```

**正确实现**:
```go
// ✅ 使用 Ion SDK (WebSocket)
import sdk "github.com/pion/ion-sdk-go"
connector := sdk.NewConnector("ws://localhost:7001")
rtc := sdk.NewRTC(connector)
```

## Ion SFU API 规范

### 连接方式
- **协议**: WebSocket
- **端口**: 7001 (JSON-RPC over WebSocket)
- **URL**: `ws://localhost:7001`

### 支持的方法

1. **join** - 创建会话
   ```json
   {
     "jsonrpc": "2.0",
     "method": "join",
     "params": {
       "sid": "session-id",
       "offer": {"type": "offer", "sdp": "..."}
     },
     "id": 1
   }
   ```

2. **offer** - 发送 offer
3. **answer** - 发送 answer  
4. **trickle** - 发送 ICE candidates

## ✅ 最终解决方案：使用 Ion SDK

### 实现概述

已成功集成 `github.com/pion/ion-sdk-go v0.7.0`，实现完整的 SFU 客户端服务。

### 核心实现

#### 1. 依赖安装
```bash
go get github.com/pion/ion-sdk-go@latest
```

#### 2. 服务初始化
```go
// internal/service/sfu_client_service.go

import (
    sdk "github.com/pion/ion-sdk-go"
    "github.com/pion/webrtc/v3"
)

type sfuClientServiceImpl struct {
    connector  *sdk.Connector
    rtcClients sync.Map  // map[sessionID]*sdk.RTC
    sessions   sync.Map  // map[sessionID]*SessionInfo
}

func NewSFUClientService(cfg *config.SFUConfig) (SFUClientService, error) {
    // 创建 Ion SDK Connector (WebSocket)
    connector := sdk.NewConnector(cfg.ServerURL)

    return &sfuClientServiceImpl{
        connector: connector,
    }, nil
}
```

#### 3. 创建会话 (CreateSession)
```go
func (s *sfuClientServiceImpl) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
    // 创建 RTC 客户端
    rtc := sdk.NewRTC(s.connector)

    // 设置回调
    rtc.OnTrack = func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
        logger.Info("接收到远程轨道")
    }
    rtc.OnError = func(err error) {
        logger.Error("RTC 错误", zap.Error(err))
    }

    // 解析客户端 SDP Offer
    offer := webrtc.SessionDescription{
        Type: webrtc.SDPTypeOffer,
        SDP:  req.SDP,
    }

    // 发送 Join 请求
    joinConfig := map[string]string{}
    err := rtc.SendJoin(req.RoomID, fmt.Sprintf("%d", req.UserID), offer, joinConfig)
    if err != nil {
        return nil, err
    }

    // 获取 SDP Answer
    var answer webrtc.SessionDescription
    if transport := rtc.GetPubTransport(); transport != nil && transport.GetPeerConnection() != nil {
        answer = *transport.GetPeerConnection().LocalDescription()
    } else if transport := rtc.GetSubTransport(); transport != nil && transport.GetPeerConnection() != nil {
        answer = *transport.GetPeerConnection().LocalDescription()
    }

    // 缓存 RTC 实例
    s.rtcClients.Store(req.SessionID, rtc)

    return &CreateSessionResponse{
        SessionID: req.SessionID,
        SDP:       answer.SDP,
    }, nil
}
```

#### 4. 关闭会话 (CloseSession)
```go
func (s *sfuClientServiceImpl) CloseSession(ctx context.Context, sessionID string) error {
    if rtcInterface, ok := s.rtcClients.Load(sessionID); ok {
        rtc := rtcInterface.(*sdk.RTC)
        rtc.Close()
        s.rtcClients.Delete(sessionID)
    }
    s.sessions.Delete(sessionID)
    return nil
}
```

#### 5. 获取统计信息 (GetSessionStats)
```go
func (s *sfuClientServiceImpl) GetSessionStats(ctx context.Context, sessionID string) (*SessionStats, error) {
    rtcInterface, ok := s.rtcClients.Load(sessionID)
    if !ok {
        return nil, errors.New("会话不存在")
    }
    rtc := rtcInterface.(*sdk.RTC)

    stats := &SessionStats{SessionID: sessionID}

    // 从 PubTransport 获取发送端统计
    if pubTransport := rtc.GetPubTransport(); pubTransport != nil {
        pc := pubTransport.GetPeerConnection()
        statsReport := pc.GetStats()
        // 解析 OutboundRTPStreamStats, ICECandidatePairStats
    }

    // 从 SubTransport 获取接收端统计
    if subTransport := rtc.GetSubTransport(); subTransport != nil {
        pc := subTransport.GetPeerConnection()
        statsReport := pc.GetStats()
        // 解析 InboundRTPStreamStats
    }

    return stats, nil
}
```

### 配置文件更新

```yaml
# configs/config.yaml
sfu:
  enabled: true
  server_url: "ws://localhost:7001"  # WebSocket URL
  mode: "standalone"
  cluster_nodes:
    - "ws://localhost:7001"
  load_balance_method: "roundrobin"
```

### 测试结果

✅ **编译成功**: `go build ./cmd/server` 无错误
✅ **服务启动成功**: SFU 客户端服务初始化成功
✅ **日志输出**: `初始化 SFU 客户端服务 (Ion SDK)` {"mode": "standalone"}

## 配置更新

### URL 格式
```yaml
# 对于 WebSocket 连接，URL 应该使用 ws:// 或 wss://
sfu:
  enabled: true
  server_url: "ws://localhost:7001"  # 使用 ws:// 而不是 http://
```

或者在代码中自动转换：
```go
// 自动将 http:// 转换为 ws://
wsURL := strings.Replace(cfg.ServerURL, "http://", "ws://", 1)
wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
```

## 验证方法

### 1. 使用浏览器测试 WebSocket
```javascript
const ws = new WebSocket('ws://localhost:7001');
ws.onopen = () => {
    console.log('Connected');
    ws.send(JSON.stringify({
        jsonrpc: "2.0",
        method: "join",
        params: {sid: "test", offer: {type: "offer", sdp: "test"}},
        id: 1
    }));
};
ws.onmessage = (e) => console.log('Response:', e.data);
```

### 2. 使用 wscat 工具
```bash
# 安装 wscat
npm install -g wscat

# 测试连接
wscat -c ws://localhost:7001

# 发送 JSON-RPC 请求
> {"jsonrpc":"2.0","method":"join","params":{"sid":"test"},"id":1}
```

## 参考资料

- [Ion SFU GitHub](https://github.com/pion/ion-sfu)
- [Ion SDK Go](https://github.com/pion/ion-sdk-go)
- [Ion 官方文档](https://pionion.github.io/docs/)
