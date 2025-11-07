# MicroVibe-Go 直播功能集成指南

本文档介绍如何在前端（Web/移动端）集成 MicroVibe-Go 的 WebRTC 直播功能。

## 目录

- [架构概述](#架构概述)
- [API 接口](#api-接口)
- [WebSocket 信令协议](#websocket-信令协议)
- [Web 客户端示例](#web-客户端示例)
- [移动端集成](#移动端集成)
- [常见问题](#常见问题)

---

## 架构概述

```
┌─────────────┐      HTTP API       ┌──────────────────┐
│   客户端    │ ──────────────────> │  MicroVibe 后端  │
│  (浏览器)   │                     │   (Gin + Go)     │
└─────────────┘                     └──────────────────┘
       │                                      │
       │ WebSocket (信令)                     │
       └──────────────────────────────────────┘
       │                                      │
       │ WebRTC (媒体流)                      ▼
       └───────────────────────────> ┌──────────────┐
                                      │  Pion SFU    │
                                      │ 媒体服务器   │
                                      └──────────────┘
```

### 技术栈

- **后端**: Go + Gin + Gorilla WebSocket
- **SFU**: Pion Ion SFU
- **前端**: WebRTC API (浏览器原生)
- **协议**: WebSocket (信令) + WebRTC (媒体流)

---

## API 接口

### 基础 URL

```
http://localhost:8080/api/v1
```

### 1. 创建直播间

**请求**:
```http
POST /live/create
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "我的第一场直播",
  "description": "直播描述",
  "cover": "https://example.com/cover.jpg"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "title": "我的第一场直播",
    "room_id": "a1b2c3d4e5f6",
    "stream_key": "f6e5d4c3b2a1",
    "status": "waiting",
    "owner_id": 123,
    "created_at": "2025-10-23T10:00:00Z"
  }
}
```

**重要字段**:
- `room_id`: WebSocket 连接时需要的房间ID
- `stream_key`: 开始/结束直播时需要的密钥（主播专用）

### 2. 开始直播

**请求**:
```http
POST /live/start
Authorization: Bearer <token>
Content-Type: application/json

{
  "stream_key": "f6e5d4c3b2a1"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "直播已开始"
}
```

### 3. 获取直播列表

**请求**:
```http
GET /live/list?status=live&page=1&page_size=20
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "title": "我的第一场直播",
        "room_id": "a1b2c3d4e5f6",
        "status": "live",
        "view_count": 150,
        "like_count": 300,
        "online_count": 50,
        "owner": {
          "id": 123,
          "username": "streamer123",
          "avatar": "https://example.com/avatar.jpg"
        }
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 4. 获取直播间详情

**请求**:
```http
GET /live/room/{room_id}
```

### 5. 结束直播

**请求**:
```http
POST /live/end
Authorization: Bearer <token>
Content-Type: application/json

{
  "stream_key": "f6e5d4c3b2a1"
}
```

---

## WebSocket 信令协议

### 连接 WebSocket

**URL**:
```
ws://localhost:8080/api/v1/live/ws?room_id=a1b2c3d4e5f6&user_id=123&username=viewer1
```

**查询参数**:
- `room_id`: 房间ID（必需）
- `user_id`: 用户ID（可选）
- `username`: 用户名（可选）

### 消息格式

所有消息采用 JSON 格式：

```json
{
  "type": "offer",
  "room_id": "a1b2c3d4e5f6",
  "user_id": 123,
  "username": "streamer",
  "payload": { ... },
  "timestamp": 1698051200
}
```

### 消息类型

#### 1. WebRTC 信令消息

##### Offer (主播发送)
```json
{
  "type": "offer",
  "payload": {
    "type": "offer",
    "sdp": "v=0\r\no=- ... "
  }
}
```

##### Answer (观众发送)
```json
{
  "type": "answer",
  "payload": {
    "type": "answer",
    "sdp": "v=0\r\no=- ... "
  }
}
```

##### ICE Candidate
```json
{
  "type": "ice",
  "payload": {
    "candidate": "candidate:...",
    "sdpMid": "0",
    "sdpMLineIndex": 0
  }
}
```

#### 2. 直播间消息

##### 聊天消息
```json
{
  "type": "chat",
  "payload": {
    "message": "Hello everyone!"
  }
}
```

##### 点赞
```json
{
  "type": "like"
}
```

##### 送礼物
```json
{
  "type": "gift",
  "payload": {
    "gift_id": 1,
    "gift_name": "玫瑰",
    "amount": 10
  }
}
```

#### 3. 系统消息（服务器推送）

##### 用户加入
```json
{
  "type": "user_joined",
  "user_id": 456,
  "username": "viewer2"
}
```

##### 用户离开
```json
{
  "type": "user_left",
  "user_id": 456,
  "username": "viewer2"
}
```

##### 错误消息
```json
{
  "type": "error",
  "payload": {
    "error": "Invalid message format"
  }
}
```

---

## Web 客户端示例

### 完整的主播推流代码

```html
<!DOCTYPE html>
<html>
<head>
  <title>主播推流 - MicroVibe</title>
</head>
<body>
  <h1>主播推流</h1>
  <video id="localVideo" autoplay muted width="640" height="480"></video>
  <button id="startBtn">开始直播</button>
  <button id="stopBtn">结束直播</button>

  <script>
    const API_BASE = 'http://localhost:8080/api/v1';
    const WS_URL = 'ws://localhost:8080/api/v1/live/ws';
    const TOKEN = localStorage.getItem('auth_token'); // 从登录获取

    let localStream = null;
    let peerConnection = null;
    let ws = null;
    let roomId = null;
    let streamKey = null;

    // WebRTC 配置
    const rtcConfig = {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        // 如果需要 TURN 服务器，添加这里
      ]
    };

    // 1. 创建直播间
    async function createLiveRoom() {
      const response = await fetch(`${API_BASE}/live/create`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${TOKEN}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          title: '测试直播间',
          description: '这是一个测试直播'
        })
      });
      const data = await response.json();
      if (data.code === 0) {
        roomId = data.data.room_id;
        streamKey = data.data.stream_key;
        console.log('直播间创建成功:', roomId);
        return true;
      }
      return false;
    }

    // 2. 获取本地媒体流
    async function getLocalStream() {
      try {
        localStream = await navigator.mediaDevices.getUserMedia({
          video: {
            width: { ideal: 1280 },
            height: { ideal: 720 }
          },
          audio: true
        });
        document.getElementById('localVideo').srcObject = localStream;
        console.log('本地媒体流获取成功');
        return true;
      } catch (error) {
        console.error('无法获取媒体设备:', error);
        alert('请允许访问摄像头和麦克风');
        return false;
      }
    }

    // 3. 连接 WebSocket 信令服务器
    function connectWebSocket() {
      const userId = 123; // 从登录信息获取
      const username = 'streamer123'; // 从登录信息获取
      const wsUrl = `${WS_URL}?room_id=${roomId}&user_id=${userId}&username=${username}`;

      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket 连接成功');
        // 发送 join 消息
        ws.send(JSON.stringify({ type: 'join' }));
      };

      ws.onmessage = async (event) => {
        const message = JSON.parse(event.data);
        console.log('收到消息:', message);

        switch (message.type) {
          case 'answer':
            // 收到观众的 Answer
            await peerConnection.setRemoteDescription(
              new RTCSessionDescription(message.payload)
            );
            break;

          case 'ice':
            // 收到 ICE Candidate
            await peerConnection.addIceCandidate(
              new RTCIceCandidate(message.payload)
            );
            break;

          case 'user_joined':
            console.log('用户加入:', message.username);
            // 为每个新观众创建 Offer
            await createOffer();
            break;

          case 'chat':
            console.log('聊天消息:', message.payload.message);
            break;
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket 错误:', error);
      };

      ws.onclose = () => {
        console.log('WebSocket 连接关闭');
      };
    }

    // 4. 创建 PeerConnection
    async function setupPeerConnection() {
      peerConnection = new RTCPeerConnection(rtcConfig);

      // 添加本地媒体流到 PeerConnection
      localStream.getTracks().forEach(track => {
        peerConnection.addTrack(track, localStream);
      });

      // 监听 ICE Candidate
      peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          ws.send(JSON.stringify({
            type: 'ice',
            payload: event.candidate
          }));
        }
      };

      // 监听连接状态
      peerConnection.onconnectionstatechange = () => {
        console.log('连接状态:', peerConnection.connectionState);
      };
    }

    // 5. 创建 Offer
    async function createOffer() {
      try {
        const offer = await peerConnection.createOffer();
        await peerConnection.setLocalDescription(offer);

        // 发送 Offer 到信令服务器
        ws.send(JSON.stringify({
          type: 'offer',
          payload: offer
        }));
      } catch (error) {
        console.error('创建 Offer 失败:', error);
      }
    }

    // 6. 通知后端开始直播
    async function startLiveStreaming() {
      const response = await fetch(`${API_BASE}/live/start`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${TOKEN}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ stream_key: streamKey })
      });
      const data = await response.json();
      if (data.code === 0) {
        console.log('直播已开始');
        return true;
      }
      return false;
    }

    // 7. 结束直播
    async function stopLiveStreaming() {
      // 关闭 PeerConnection
      if (peerConnection) {
        peerConnection.close();
        peerConnection = null;
      }

      // 停止本地媒体流
      if (localStream) {
        localStream.getTracks().forEach(track => track.stop());
        localStream = null;
      }

      // 关闭 WebSocket
      if (ws) {
        ws.send(JSON.stringify({ type: 'leave' }));
        ws.close();
        ws = null;
      }

      // 通知后端结束直播
      await fetch(`${API_BASE}/live/end`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${TOKEN}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ stream_key: streamKey })
      });

      console.log('直播已结束');
    }

    // 主流程
    document.getElementById('startBtn').onclick = async () => {
      console.log('开始直播流程...');

      // 1. 创建直播间
      if (!await createLiveRoom()) {
        alert('创建直播间失败');
        return;
      }

      // 2. 获取本地媒体流
      if (!await getLocalStream()) {
        return;
      }

      // 3. 连接 WebSocket
      connectWebSocket();

      // 等待 WebSocket 连接成功
      await new Promise(resolve => {
        const checkWs = setInterval(() => {
          if (ws && ws.readyState === WebSocket.OPEN) {
            clearInterval(checkWs);
            resolve();
          }
        }, 100);
      });

      // 4. 设置 PeerConnection
      await setupPeerConnection();

      // 5. 创建 Offer
      await createOffer();

      // 6. 通知后端开始直播
      await startLiveStreaming();

      alert('直播已开始！');
    };

    document.getElementById('stopBtn').onclick = async () => {
      await stopLiveStreaming();
      alert('直播已结束！');
    };
  </script>
</body>
</html>
```

### 完整的观众拉流代码

```html
<!DOCTYPE html>
<html>
<head>
  <title>观众观看 - MicroVibe</title>
</head>
<body>
  <h1>观看直播</h1>
  <video id="remoteVideo" autoplay width="640" height="480"></video>
  <button id="joinBtn">加入直播</button>
  <button id="leaveBtn">离开直播</button>

  <div>
    <input id="chatInput" type="text" placeholder="输入聊天消息" />
    <button id="sendBtn">发送</button>
  </div>

  <script>
    const API_BASE = 'http://localhost:8080/api/v1';
    const WS_URL = 'ws://localhost:8080/api/v1/live/ws';
    const TOKEN = localStorage.getItem('auth_token');
    const ROOM_ID = 'a1b2c3d4e5f6'; // 从直播列表获取

    let peerConnection = null;
    let ws = null;

    const rtcConfig = {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    };

    // 连接 WebSocket
    function connectWebSocket() {
      const userId = 456;
      const username = 'viewer456';
      const wsUrl = `${WS_URL}?room_id=${ROOM_ID}&user_id=${userId}&username=${username}`;

      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket 连接成功');
        ws.send(JSON.stringify({ type: 'join' }));
      };

      ws.onmessage = async (event) => {
        const message = JSON.parse(event.data);
        console.log('收到消息:', message);

        switch (message.type) {
          case 'offer':
            // 收到主播的 Offer
            await handleOffer(message.payload);
            break;

          case 'ice':
            // 收到 ICE Candidate
            await peerConnection.addIceCandidate(
              new RTCIceCandidate(message.payload)
            );
            break;

          case 'chat':
            console.log('聊天:', message.username, ':', message.payload.message);
            break;

          case 'user_joined':
            console.log('用户加入:', message.username);
            break;
        }
      };
    }

    // 处理 Offer 并创建 Answer
    async function handleOffer(offer) {
      // 创建 PeerConnection
      peerConnection = new RTCPeerConnection(rtcConfig);

      // 监听远程媒体流
      peerConnection.ontrack = (event) => {
        console.log('收到远程媒体流');
        document.getElementById('remoteVideo').srcObject = event.streams[0];
      };

      // 监听 ICE Candidate
      peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          ws.send(JSON.stringify({
            type: 'ice',
            payload: event.candidate
          }));
        }
      };

      // 设置远程 SDP
      await peerConnection.setRemoteDescription(
        new RTCSessionDescription(offer)
      );

      // 创建 Answer
      const answer = await peerConnection.createAnswer();
      await peerConnection.setLocalDescription(answer);

      // 发送 Answer
      ws.send(JSON.stringify({
        type: 'answer',
        payload: answer
      }));
    }

    // 发送聊天消息
    function sendChat(message) {
      ws.send(JSON.stringify({
        type: 'chat',
        payload: { message }
      }));
    }

    // 事件绑定
    document.getElementById('joinBtn').onclick = () => {
      connectWebSocket();
    };

    document.getElementById('leaveBtn').onclick = () => {
      if (peerConnection) {
        peerConnection.close();
      }
      if (ws) {
        ws.send(JSON.stringify({ type: 'leave' }));
        ws.close();
      }
    };

    document.getElementById('sendBtn').onclick = () => {
      const input = document.getElementById('chatInput');
      sendChat(input.value);
      input.value = '';
    };
  </script>
</body>
</html>
```

---

## 移动端集成

### Android (Kotlin + WebRTC)

```kotlin
// build.gradle
dependencies {
    implementation 'org.webrtc:google-webrtc:1.0.+'
}

// LiveStreamActivity.kt
class LiveStreamActivity : AppCompatActivity() {
    private lateinit var peerConnectionFactory: PeerConnectionFactory
    private lateinit var peerConnection: PeerConnection
    private lateinit var webSocket: WebSocket

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // 初始化 PeerConnectionFactory
        PeerConnectionFactory.initialize(
            PeerConnectionFactory.InitializationOptions.builder(this)
                .createInitializationOptions()
        )

        peerConnectionFactory = PeerConnectionFactory.builder()
            .createPeerConnectionFactory()

        // 创建 PeerConnection
        val rtcConfig = PeerConnection.RTCConfiguration(listOf(
            PeerConnection.IceServer.builder("stun:stun.l.google.com:19302").createIceServer()
        ))

        peerConnection = peerConnectionFactory.createPeerConnection(
            rtcConfig,
            object : PeerConnection.Observer {
                override fun onIceCandidate(candidate: IceCandidate?) {
                    // 发送 ICE Candidate 到信令服务器
                    sendIceCandidate(candidate)
                }

                override fun onAddStream(stream: MediaStream?) {
                    // 收到远程媒体流
                    stream?.videoTracks?.get(0)?.addSink(remoteVideoView)
                }
            }
        )!!
    }
}
```

### iOS (Swift + WebRTC)

```swift
// Podfile
pod 'GoogleWebRTC'

// LiveStreamViewController.swift
import WebRTC

class LiveStreamViewController: UIViewController {
    let peerConnectionFactory: RTCPeerConnectionFactory = {
        RTCInitializeSSL()
        return RTCPeerConnectionFactory()
    }()

    var peerConnection: RTCPeerConnection?
    var webSocket: URLSessionWebSocketTask?

    override func viewDidLoad() {
        super.viewDidLoad()

        // 创建 PeerConnection
        let config = RTCConfiguration()
        config.iceServers = [RTCIceServer(urlStrings: ["stun:stun.l.google.com:19302"])]

        peerConnection = peerConnectionFactory.peerConnection(
            with: config,
            constraints: RTCMediaConstraints(mandatoryConstraints: nil, optionalConstraints: nil),
            delegate: self
        )
    }
}

extension LiveStreamViewController: RTCPeerConnectionDelegate {
    func peerConnection(_ peerConnection: RTCPeerConnection, didGenerate candidate: RTCIceCandidate) {
        // 发送 ICE Candidate
        sendIceCandidate(candidate)
    }

    func peerConnection(_ peerConnection: RTCPeerConnection, didAdd stream: RTCMediaStream) {
        // 收到远程媒体流
        if let videoTrack = stream.videoTracks.first {
            videoTrack.add(remoteVideoView)
        }
    }
}
```

---

## 常见问题

### 1. WebRTC 连接失败

**可能原因**:
- 防火墙阻止 UDP 端口（5000-5100）
- NAT 穿透失败，需要配置 TURN 服务器

**解决方案**:
```javascript
// 添加 TURN 服务器配置
const rtcConfig = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    {
      urls: 'turn:your-turn-server.com:3478',
      username: 'user',
      credential: 'pass'
    }
  ]
};
```

### 2. 音视频不同步

检查网络延迟和编码参数：
```javascript
peerConnection.addTransceiver('video', {
  direction: 'sendrecv',
  streams: [localStream],
  sendEncodings: [
    { maxBitrate: 2000000 } // 2 Mbps
  ]
});
```

### 3. 移动端性能优化

```javascript
// 降低分辨率
const constraints = {
  video: {
    width: { ideal: 640 },
    height: { ideal: 480 },
    frameRate: { ideal: 30 }
  },
  audio: true
};
```

### 4. 断线重连

```javascript
ws.onclose = () => {
  console.log('连接断开，3秒后重连...');
  setTimeout(() => {
    connectWebSocket();
  }, 3000);
};
```

---

## 性能建议

1. **编码参数优化**
   - 视频: 720p @ 30fps, 2 Mbps
   - 音频: 128 kbps, 48 kHz

2. **网络优化**
   - 使用 NACK 丢包重传
   - 启用 PLI 关键帧请求
   - 配置自适应码率

3. **服务器扩展**
   - 使用 SFU 集群支持大规模直播
   - 使用 CDN 分发录制回放

---

## 参考资源

- [WebRTC API 文档](https://developer.mozilla.org/zh-CN/docs/Web/API/WebRTC_API)
- [Pion WebRTC](https://github.com/pion/webrtc)
- [Ion SFU 文档](https://github.com/pion/ion-sfu)

---

**祝您开发顺利！如有问题，请提交 Issue。**
