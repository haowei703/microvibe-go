# 混合推流架构设计 - WebRTC + RTMP 双协议

## 架构概述

支持主播通过 **WebRTC** 或 **RTMP** 两种协议推流，通过 **SFU (Selective Forwarding Unit)** 分发 WebRTC 流，同时支持 **HLS/FLV** 播放。

## 架构图

```
                    ┌─────────────────────────────────────┐
                    │         主播端 (推流)                │
                    └─────────────────────────────────────┘
                              │           │
                    ┌─────────┴────┐     └────────┐
                    │ WebRTC 推流   │              │ RTMP 推流
                    │ (浏览器)      │              │ (OBS Studio)
                    └──────┬────────┘              └────┬────────┘
                           │                            │
                           ▼                            ▼
              ┌─────────────────────┐        ┌──────────────────┐
              │  WebRTC Signaling   │        │  RTMP Server     │
              │  (WebSocket)        │        │  (Nginx-RTMP)    │
              └──────────┬──────────┘        └────────┬─────────┘
                         │                            │
                         ▼                            │
              ┌─────────────────────┐                 │
              │   SFU Server        │                 │
              │   (Pion Ion SFU)    │                 │
              └──────────┬──────────┘                 │
                         │                            │
        ┌────────────────┼────────────────┐          │
        │                │                │          │
        ▼                ▼                ▼          ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ WebRTC 观众1 │ │ WebRTC 观众2 │ │ WebRTC 观众N │ │ HLS/FLV 观众 │
│  (浏览器)    │ │  (浏览器)    │ │  (浏览器)    │ │  (播放器)    │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
```

## 推流协议选择

### WebRTC 推流 (浏览器)
- **优点**: 超低延迟 (< 1秒)，无需额外软件
- **缺点**: 浏览器性能限制
- **适用场景**: 临时直播、移动端直播

### RTMP 推流 (OBS)
- **优点**: 稳定性好，专业功能多 (美颜、特效、多路混流)
- **缺点**: 需要专业软件，延迟稍高 (2-5秒)
- **适用场景**: 专业直播、游戏直播、长时间直播

## 播放协议选择

### WebRTC 播放
- **延迟**: < 1秒 (超低延迟)
- **适用场景**: 互动直播、实时连麦

### HLS 播放
- **延迟**: 10-30秒
- **适用场景**: 移动端、跨平台兼容性

### FLV 播放
- **延迟**: 2-5秒
- **适用场景**: PC 端、低延迟需求

## 核心组件

### 1. SFU Server (Pion Ion SFU)
- **作用**: WebRTC 流分发，避免主播上行带宽瓶颈
- **部署方式**: Docker 容器
- **通信协议**: JSON-RPC over HTTP

### 2. RTMP Server (Nginx-RTMP)
- **作用**: 接收 RTMP 推流，转换为 HLS/FLV
- **部署方式**: Nginx + rtmp 模块
- **可选**: 将 RTMP 流转为 WebRTC (通过 FFmpeg)

### 3. 信令服务 (LiveSignalingService)
- **作用**: WebRTC 信令交换，调用 SFU API
- **协议**: WebSocket
- **功能**:
  - 接收客户端 SDP Offer
  - 调用 SFU 创建会话
  - 返回 SDP Answer
  - 转发 ICE Candidates

### 4. 推流管理服务 (LiveStreamService)
- **作用**: 管理直播间状态，生成推流地址
- **功能**:
  - 创建直播间时生成 RTMP + WebRTC 推流地址
  - 记录推流协议 (rtmp/webrtc)
  - 统计推流/播放数据

## 数据流

### WebRTC 推流流程

```
1. 主播创建直播间
   POST /api/v1/live/create
   → 返回 room_id, stream_key, webrtc_url

2. 主播浏览器建立 WebSocket 连接
   WS /api/v1/live/ws?room_id=xxx&user_id=xxx&role=publisher

3. 主播发送 SDP Offer
   → 信令服务调用 SFU API: CreateSession
   → SFU 返回 SDP Answer
   → 信令服务转发给主播

4. 交换 ICE Candidates
   → 主播 ←→ 信令服务 ←→ SFU

5. WebRTC 连接建立
   主播 ←→ SFU (媒体流)

6. 观众加入
   观众1 → 信令服务 → SFU → 创建订阅会话
   观众2 → 信令服务 → SFU → 创建订阅会话
   观众N → ...

7. SFU 分发流
   主播 → SFU → 观众1
             → 观众2
             → 观众N
```

### RTMP 推流流程

```
1. 主播创建直播间
   POST /api/v1/live/create
   → 返回 stream_url: rtmp://server/live/streamKey

2. OBS 推流
   OBS → RTMP Server (Nginx-RTMP)

3. Nginx-RTMP 处理
   → 生成 HLS (m3u8 + ts 文件)
   → 生成 FLV 流

4. 观众播放
   HLS: http://server/hls/streamKey.m3u8
   FLV: http://server/flv/streamKey.flv
```

### 混合流场景 (RTMP → WebRTC)

```
OBS → RTMP Server → FFmpeg 转码 → WebRTC → SFU → WebRTC 观众
                  ↓
                HLS/FLV → HLS/FLV 观众
```

## 实现细节

### 1. 修改 LiveSignalingService

**原始实现** (点对点):
```go
// 直接转发 Offer/Answer
func (s *liveSignalingServiceImpl) handleMessage(client *Client, msg *SignalingMessage) {
    // 广播给房间内所有人
    s.BroadcastToRoom(client.RoomID, msg, client.UserID)
}
```

**新实现** (通过 SFU):
```go
// 调用 SFU API
func (s *liveSignalingServiceImpl) handleOffer(client *Client, msg *SignalingMessage) {
    // 1. 调用 SFU 创建会话
    sfuReq := &CreateSessionRequest{
        SessionID: fmt.Sprintf("%s-%d", client.RoomID, client.UserID),
        RoomID:    client.RoomID,
        UserID:    client.UserID,
        Role:      RolePublisher, // 或 RoleSubscriber
        SDP:       msg.Payload.(string),
    }

    // 2. 调用 SFU Client Service
    sfuResp, err := s.sfuClient.CreateSession(ctx, sfuReq)
    if err != nil {
        // 发送错误消息给客户端
        return
    }

    // 3. 返回 SDP Answer
    answerMsg := &SignalingMessage{
        Type:    MessageTypeAnswer,
        Payload: sfuResp.SDP,
    }
    client.Conn.WriteJSON(answerMsg)
}
```

### 2. 修改 CreateLiveStreamRequest

添加推流方式选择:

```go
type CreateLiveStreamRequest struct {
    Title        string `json:"title" binding:"required"`

    // 推流方式选择
    StreamingMode string `json:"streaming_mode"` // "rtmp", "webrtc", "hybrid"

    // 其他配置...
}
```

### 3. 生成推流地址

```go
func (s *liveStreamServiceImpl) CreateLiveStream(...) {
    // 根据 streaming_mode 生成对应的地址

    if req.StreamingMode == "rtmp" || req.StreamingMode == "hybrid" {
        streamURL = fmt.Sprintf("rtmp://%s/live/%s",
            s.cfg.Streaming.RTMPServer, streamKey)
    }

    if req.StreamingMode == "webrtc" || req.StreamingMode == "hybrid" {
        webrtcURL = fmt.Sprintf("webrtc://room/%s", roomID)
    }
}
```

## 配置文件

### configs/config.yaml

```yaml
# SFU 服务器配置
sfu:
  enabled: true                           # 是否启用 SFU
  server_url: "http://localhost:7000"     # SFU JSON-RPC 地址（完整 URL）
  mode: "standalone"                      # standalone 或 cluster

  # 集群模式
  cluster_nodes:                          # 完整 URL 格式
    - "http://sfu-1:7000"
    - "http://sfu-2:7000"

  load_balance_method: "roundrobin"       # random, roundrobin, leastconn

# RTMP 服务器配置
streaming:
  rtmp_server: "rtmp://localhost:1935/live"
  hls_server: "http://localhost:8080/hls"
  flv_server: "http://localhost:8080/flv"

  # RTMP to WebRTC 桥接 (可选)
  rtmp_to_webrtc: false
  ffmpeg_path: "/usr/bin/ffmpeg"

# WebRTC 配置
webrtc:
  ice_servers:
    - urls:
        - "stun:stun.l.google.com:19302"
  enable_turn: false
  turn_servers:
    - urls: "turn:turn.example.com:3478"
      username: "user"
      credential: "pass"
```

## 部署方案

### Docker Compose 部署

```yaml
version: '3.8'

services:
  # 应用服务
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SFU_SERVER_URL=http://ion-sfu:7000
      - RTMP_SERVER=rtmp://nginx-rtmp:1935/live
    depends_on:
      - postgres
      - redis
      - ion-sfu
      - nginx-rtmp

  # SFU 服务器
  ion-sfu:
    image: pion/ion-sfu:latest
    ports:
      - "7000:7000"    # JSON-RPC
      - "5000-5100:5000-5100/udp"  # WebRTC UDP
    volumes:
      - ./configs/sfu.toml:/configs/sfu.toml

  # RTMP 服务器
  nginx-rtmp:
    image: tiangolo/nginx-rtmp
    ports:
      - "1935:1935"    # RTMP
      - "8081:80"      # HLS/FLV HTTP
    volumes:
      - ./configs/nginx.conf:/etc/nginx/nginx.conf
      - ./data/hls:/tmp/hls
      - ./data/recordings:/tmp/recordings

  # 数据库
  postgres:
    image: postgres:16
    # ...

  redis:
    image: redis:7
    # ...
```

## API 使用示例

### 1. 创建支持双协议的直播间

```bash
POST /api/v1/live/create
{
  "title": "我的直播",
  "streaming_mode": "hybrid",  # 支持 RTMP + WebRTC
  "push_protocol": "rtmp",     # 默认推流方式
  "stream_type": "video_audio"
}
```

**响应:**

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "room_id": "room_abc123",
    "stream_key": "xyz789",

    // RTMP 推流地址
    "stream_url": "rtmp://localhost:1935/live/xyz789",

    // WebRTC 推流地址
    "webrtc_url": "webrtc://room/room_abc123",

    // 播放地址
    "play_url": "http://localhost:8080/hls/xyz789.m3u8",
    "flv_url": "http://localhost:8080/flv/xyz789.flv",
    "webrtc_play_url": "webrtc://room/room_abc123",

    "streaming_mode": "hybrid"
  }
}
```

### 2. WebRTC 推流 (浏览器端)

```javascript
// 1. 建立 WebSocket 连接
const ws = new WebSocket('ws://localhost:8080/api/v1/live/ws?room_id=room_abc123&user_id=1&role=publisher');

// 2. 获取本地媒体流
const stream = await navigator.mediaDevices.getUserMedia({
  video: true,
  audio: true
});

// 3. 创建 PeerConnection
const pc = new RTCPeerConnection({
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
});

// 4. 添加本地流
stream.getTracks().forEach(track => pc.addTrack(track, stream));

// 5. 创建 Offer
const offer = await pc.createOffer();
await pc.setLocalDescription(offer);

// 6. 发送 Offer 到服务器
ws.send(JSON.stringify({
  type: 'offer',
  room_id: 'room_abc123',
  payload: offer.sdp
}));

// 7. 接收 Answer
ws.onmessage = async (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'answer') {
    await pc.setRemoteDescription({
      type: 'answer',
      sdp: msg.payload
    });
  }
  if (msg.type === 'ice') {
    await pc.addIceCandidate(msg.payload);
  }
};

// 8. 发送 ICE Candidates
pc.onicecandidate = (event) => {
  if (event.candidate) {
    ws.send(JSON.stringify({
      type: 'ice',
      payload: event.candidate
    }));
  }
};
```

## 性能指标

### WebRTC (SFU 分发)
- **延迟**: < 500ms
- **并发观众**: 1000+ (单 SFU 节点)
- **带宽**: 主播上行 2-5 Mbps，SFU 下行按观众数量

### RTMP + HLS
- **延迟**: 10-30秒
- **并发观众**: 10000+ (CDN 加速)
- **带宽**: 主播上行 2-5 Mbps，CDN 分发

## 故障处理

### SFU 不可用
- 自动降级为 RTMP 推流
- 提示主播切换推流方式

### RTMP 服务器不可用
- 提示主播使用 WebRTC 推流

### 网络抖动
- WebRTC 自适应码率
- RTMP 缓冲区调整

## 未来扩展

1. **RTMP to WebRTC 桥接**: 使用 FFmpeg 将 RTMP 流转为 WebRTC
2. **多路混流**: 支持多个主播同时推流到一个房间
3. **AI 处理**: 实时美颜、虚拟背景、声音增强
4. **录制和回放**: 自动录制直播，生成回放
5. **多清晰度**: 实时转码生成多档清晰度

---

**文档版本**: v1.0
**更新日期**: 2025-11-03
