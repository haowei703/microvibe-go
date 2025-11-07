# Pion Ion SFU 集成架构指南

> **企业级 WebRTC 直播系统 - 支持水平扩展、自定义编解码和高质量流媒体**

## 📐 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        客户端层                                  │
│  ┌──────────────┐         ┌──────────────┐                     │
│  │  主播客户端    │         │  观众客户端    │                     │
│  │ (浏览器/App)  │         │ (浏览器/App)  │                     │
│  └───────┬──────┘         └───────┬──────┘                     │
│          │                        │                             │
└──────────┼────────────────────────┼─────────────────────────────┘
           │                        │
           │ ① WebSocket           │ ① WebSocket
           │   (信令)               │   (信令)
           │                        │
           ▼                        ▼
┌────────────────────────────────────────────────────────────────┐
│                     Go 后端服务层                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  WebSocket 信令服务 (live_signaling_service.go)           │  │
│  │  - 房间管理                                               │  │
│  │  - 用户认证                                               │  │
│  │  - 业务消息（聊天、礼物、点赞）                             │  │
│  └────────────────┬─────────────────────────────────────────┘  │
│                   │                                             │
│                   │ ② JSON-RPC                                 │
│                   ▼                                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  SFU 客户端服务 (sfu_client_service.go)                   │  │
│  │  - 调用 Pion Ion SFU API                                  │  │
│  │  - 会话管理                                               │  │
│  │  - 质量控制                                               │  │
│  │  - 负载均衡                                               │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          │ ② JSON-RPC
                          │   (http://ion-sfu:7001)
                          ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Pion Ion SFU 服务层                             │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Pion Ion SFU (Docker 容器)                                │  │
│  │  - WebRTC 媒体转发                                          │  │
│  │  - 自动编解码 (VP8/VP9/H264)                                │  │
│  │  - 联播 (Simulcast)                                        │  │
│  │  - 动态广播 (Dynacast)                                     │  │
│  │  - QoS 质量控制 (NACK, PLI)                                │  │
│  └────────────────────────────────────────────────────────────┘  │
│          ▲                                    ▲                   │
└──────────┼────────────────────────────────────┼───────────────────┘
           │                                    │
           │ ③ WebRTC 媒体流                   │ ③ WebRTC 媒体流
           │   (UDP 5000-5100)                 │   (UDP 5000-5100)
           │                                    │
     ┌─────┴────────┐                    ┌─────┴────────┐
     │ 主播推流      │                    │ 观众拉流      │
     │ (Publish)    │                    │ (Subscribe)  │
     └──────────────┘                    └──────────────┘
```

### 通信流程详解

#### 1️⃣ **信令通道** (WebSocket)
- **客户端 ↔ Go 后端**
- **用途**: 房间管理、认证、业务消息
- **协议**: WebSocket (ws://localhost:8080/api/v1/live/ws)

#### 2️⃣ **SFU 控制通道** (JSON-RPC)
- **Go 后端 ↔ Pion Ion SFU**
- **用途**: 创建/关闭会话、质量控制
- **协议**: JSON-RPC over HTTP (http://ion-sfu:7001)

#### 3️⃣ **媒体传输通道** (WebRTC)
- **客户端 ↔ Pion Ion SFU**
- **用途**: 音视频数据传输
- **协议**: WebRTC (UDP 端口 5000-5100)
- **流程**:
  - 主播推流：客户端 → SFU (Publish)
  - 观众拉流：SFU → 客户端 (Subscribe)

---

## 🔧 技术栈

| 组件 | 技术 | 作用 |
|-----|------|------|
| **信令服务器** | Go + Gorilla WebSocket | 房间管理、业务逻辑 |
| **SFU 服务器** | Pion Ion SFU (Docker) | 媒体转发、编解码 |
| **SFU 客户端** | Go + JSON-RPC | 调用 SFU API |
| **数据库** | PostgreSQL | 业务数据存储 |
| **缓存** | Redis | 在线状态、统计 |
| **容器编排** | Docker Compose | 服务部署 |

---

## 🚀 快速开始

### 1. 启动服务

```bash
# 启动所有服务（包括 Pion Ion SFU）
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看 SFU 日志
docker-compose logs -f ion-sfu
```

### 2. 服务端点

| 服务 | 端点 | 说明 |
|-----|------|------|
| **Go API** | http://localhost:8080 | RESTful API |
| **WebSocket** | ws://localhost:8080/api/v1/live/ws | 信令服务 |
| **Pion Ion SFU** | http://localhost:7001 | JSON-RPC API |
| **SFU UDP** | UDP 5000-5100 | WebRTC 媒体端口 |

### 3. 配置文件

#### `configs/config.yaml`

```yaml
webrtc:
  # ICE 服务器（NAT 穿透）
  ice_servers:
    - urls:
        - "stun:stun.l.google.com:19302"

  # 端口范围
  port_min: 5000
  port_max: 5100

  # 编解码器
  video_codecs: ["VP8", "VP9", "H264"]
  audio_codecs: ["Opus"]

  # 质量配置
  max_bandwidth: 3000       # 3 Mbps
  video_bitrate: 2000       # 2 Mbps
  audio_bitrate: 128        # 128 kbps
  enable_simulcast: true    # 多码率
  enable_adaptive_rate: true # 自适应

  # SFU 服务器
  sfu_address: "http://ion-sfu:7001"
  sfu_mode: "standalone"
```

---

## 💻 API 使用示例

### 主播推流流程

```javascript
// 1. 创建直播间
const createResponse = await fetch('/api/v1/live/create', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + token,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    title: '我的直播间',
    description: '测试直播'
  })
});
const { room_id } = await createResponse.json();

// 2. 连接 WebSocket 信令服务器
const ws = new WebSocket(
  `ws://localhost:8080/api/v1/live/ws?room_id=${room_id}&user_id=${userId}`
);

// 3. 获取本地媒体流
const localStream = await navigator.mediaDevices.getUserMedia({
  video: {
    width: { ideal: 1280 },
    height: { ideal: 720 }
  },
  audio: true
});

// 4. 创建 PeerConnection
const pc = new RTCPeerConnection({
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' }
  ]
});

// 5. 添加本地媒体流
localStream.getTracks().forEach(track => {
  pc.addTrack(track, localStream);
});

// 6. 创建 Offer
const offer = await pc.createOffer();
await pc.setLocalDescription(offer);

// 7. 发送 Offer 到信令服务器
ws.send(JSON.stringify({
  type: 'offer',
  room_id: room_id,
  payload: {
    type: offer.type,
    sdp: offer.sdp
  }
}));

// 8. 接收 Answer（来自 SFU）
ws.onmessage = async (event) => {
  const message = JSON.parse(event.data);

  if (message.type === 'answer') {
    const answer = new RTCSessionDescription(message.payload);
    await pc.setRemoteDescription(answer);
    console.log('推流成功！');
  }

  if (message.type === 'ice') {
    await pc.addICECandidate(message.payload);
  }
};

// 9. 处理 ICE Candidate
pc.onicecandidate = (event) => {
  if (event.candidate) {
    ws.send(JSON.stringify({
      type: 'ice',
      room_id: room_id,
      payload: event.candidate
    }));
  }
};

// 10. 开始直播
await fetch('/api/v1/live/start', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + token,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ room_id })
});
```

### 观众拉流流程

```javascript
// 1. 连接 WebSocket
const ws = new WebSocket(
  `ws://localhost:8080/api/v1/live/ws?room_id=${roomId}&user_id=${userId}`
);

// 2. 创建 PeerConnection
const pc = new RTCPeerConnection({
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' }
  ]
});

// 3. 接收远程媒体流
pc.ontrack = (event) => {
  const remoteVideo = document.getElementById('remoteVideo');
  if (event.streams && event.streams[0]) {
    remoteVideo.srcObject = event.streams[0];
    console.log('收到主播视频流');
  }
};

// 4. 发送加入请求（WebSocket）
ws.send(JSON.stringify({
  type: 'join',
  room_id: roomId
}));

// 5. 接收 Offer（来自 SFU）
ws.onmessage = async (event) => {
  const message = JSON.parse(event.data);

  if (message.type === 'offer') {
    // 设置远程描述
    const offer = new RTCSessionDescription(message.payload);
    await pc.setRemoteDescription(offer);

    // 创建 Answer
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);

    // 发送 Answer 到信令服务器
    ws.send(JSON.stringify({
      type: 'answer',
      room_id: roomId,
      payload: {
        type: answer.type,
        sdp: answer.sdp
      }
    }));
  }

  if (message.type === 'ice') {
    await pc.addICECandidate(message.payload);
  }
};

// 6. 处理 ICE Candidate
pc.onicecandidate = (event) => {
  if (event.candidate) {
    ws.send(JSON.stringify({
      type: 'ice',
      room_id: roomId,
      payload: event.candidate
    }));
  }
};

// 7. 加入直播间
await fetch(`/api/v1/live/join/${roomId}`, {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + token
  }
});
```

---

## ⚙️ 高级功能

### 1. 自定义编解码器

```go
// 在代码中动态设置
quality := service.QualityConfig{
    VideoCodec:     "H264",  // VP8, VP9, H264
    AudioCodec:     "Opus",
    VideoBitrate:   2000,    // kbps
    AudioBitrate:   128,
    EnableSimulcast: true,
}

err := sfuService.UpdateQuality(ctx, sessionID, quality)
```

### 2. 联播 (Simulcast) - 多码率适配

```yaml
# configs/config.yaml
webrtc:
  enable_simulcast: true  # 启用联播
```

**效果**:
- 主播推送 3 个质量层：高 (1080p)、中 (720p)、低 (360p)
- 观众根据网络自动选择最佳质量层

### 3. QoS 质量控制

```yaml
webrtc:
  enable_nack: true        # 丢包重传
  enable_pli: true         # 关键帧请求
  enable_adaptive_rate: true  # 自适应码率
```

### 4. 集群部署（水平扩展）

#### 修改 `docker-compose.yml`

```yaml
services:
  # SFU 节点 1
  ion-sfu-1:
    image: pionwebrtc/ion-sfu:latest-allrpc
    container_name: microvibe-ion-sfu-1
    ports:
      - "5000-5100:5000-5100/udp"
      - "7001:7001"
    volumes:
      - ./configs/sfu.toml:/configs/sfu.toml
    networks:
      - microvibe-network

  # SFU 节点 2
  ion-sfu-2:
    image: pionwebrtc/ion-sfu:latest-allrpc
    container_name: microvibe-ion-sfu-2
    ports:
      - "5100-5200:5000-5100/udp"
      - "7002:7001"
    volumes:
      - ./configs/sfu.toml:/configs/sfu.toml
    networks:
      - microvibe-network

  # SFU 节点 3
  ion-sfu-3:
    image: pionwebrtc/ion-sfu:latest-allrpc
    container_name: microvibe-ion-sfu-3
    ports:
      - "5200-5300:5000-5100/udp"
      - "7003:7001"
    volumes:
      - ./configs/sfu.toml:/configs/sfu.toml
    networks:
      - microvibe-network
```

#### 修改 `configs/config.yaml`

```yaml
webrtc:
  sfu_mode: "cluster"
  cluster_nodes:
    - "ion-sfu-1:7001"
    - "ion-sfu-2:7002"
    - "ion-sfu-3:7003"
  load_balance_method: "roundrobin"  # 轮询负载均衡
```

**效果**:
- 支持 1000+ 并发直播间
- 自动故障转移
- 水平扩展

---

## 📊 监控和统计

### 获取 SFU 服务器信息

```bash
curl http://localhost:7001/info
```

响应：
```json
{
  "version": "1.11.0",
  "active_sessions": 125,
  "total_rooms": 45,
  "total_bandwidth": 150000000,
  "uptime": 86400
}
```

### 获取会话统计

```go
stats, err := sfuService.GetSessionStats(ctx, sessionID)
// stats.VideoBitrate: 视频比特率
// stats.PacketLoss: 丢包率
// stats.Jitter: 抖动
// stats.RoundTripTime: RTT
```

---

## 🔍 故障排查

### 1. SFU 连接失败

```bash
# 检查 SFU 容器状态
docker-compose ps ion-sfu

# 查看 SFU 日志
docker-compose logs -f ion-sfu

# 测试 JSON-RPC 连接
curl -X POST http://localhost:7001 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"ping","id":1}'
```

### 2. 媒体流无法建立

- 检查防火墙是否开放 UDP 5000-5100
- 检查 NAT 穿透（STUN/TURN 服务器）
- 查看浏览器控制台 WebRTC 日志

### 3. 视频质量问题

```yaml
# 提高码率
webrtc:
  max_bandwidth: 5000     # 5 Mbps
  video_bitrate: 3000     # 3 Mbps

# 使用 H264 硬件加速
webrtc:
  video_codecs: ["H264", "VP9", "VP8"]
```

---

## 📈 性能优化

### 1. 码率配置

| 分辨率 | 推荐码率 | 适用场景 |
|--------|---------|---------|
| 360p | 500 kbps | 低带宽 |
| 480p | 1000 kbps | 移动网络 |
| 720p | 2000 kbps | 标准直播 |
| 1080p | 4000 kbps | 高清直播 |
| 4K | 10000 kbps | 超清直播 |

### 2. 启用硬件加速

```yaml
# 优先使用 H264（支持硬件编解码）
webrtc:
  video_codecs: ["H264", "VP8", "VP9"]
```

### 3. 网络优化

- 启用 NACK (丢包重传)
- 启用 PLI (关键帧请求)
- 启用自适应码率

---

## 🔐 安全最佳实践

1. **JWT 认证**: WebSocket 连接必须携带有效 Token
2. **TURN 服务器**: 生产环境配置专用 TURN 服务器
3. **HTTPS/WSS**: 生产环境使用加密连接
4. **API 限流**: 防止滥用

---

## 🆚 架构对比

| 特性 | P2P Mesh | SFU (本方案) | MCU |
|-----|----------|-------------|-----|
| **服务器负载** | 低 | 中 | 高 |
| **客户端带宽** | 高 | 低 | 低 |
| **延迟** | 低 | 中 | 高 |
| **扩展性** | 差 (≤10人) | 好 (100+) | 好 (1000+) |
| **成本** | 低 | 中 | 高 |
| **适用场景** | 小会议 | 直播 | 大会议 |

---

## 📚 参考文档

- [Pion WebRTC 官方文档](https://github.com/pion/webrtc)
- [Pion Ion SFU 文档](https://github.com/pion/ion-sfu)
- [WebRTC 规范](https://www.w3.org/TR/webrtc/)
- [项目 API 文档](./openapi.json)

---

## 🎯 下一步

1. ✅ **已完成**: SFU 集成、编解码配置、QoS 控制
2. 🚧 **进行中**: 集群部署、负载均衡
3. 📋 **待实现**:
   - WebSocket 实时通知优化
   - 直播录制和回放
   - CDN 集成
   - 实时转码

---

**生成时间**: 2025-10-29
**版本**: 1.0.0
**维护者**: MicroVibe-Go Team
