# 前端消息系统集成指南

## 概述

MicroVibe-Go 消息系统采用 **REST API + WebSocket** 混合架构：
- **REST API**：持久化操作（发送、拉取历史、标记已读）
- **WebSocket**：实时推送（收到新消息通知）

---

## 1. API 端点

### 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: Bearer Token（JWT）
- **请求头**: `Authorization: Bearer <token>`

### 1.1 发送消息

```http
POST /messages/send
Content-Type: application/json
Authorization: Bearer <token>

{
  "receiver_id": 2,
  "type": 1,
  "content": "你好，在吗？",
  "media_url": "",
  "video_id": null
}
```

**请求参数**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| receiver_id | uint | ✅ | 接收者用户ID |
| type | int8 | ✅ | 消息类型：1-文本，2-图片，3-视频，4-语音 |
| content | string | ⚠️ | 消息内容（文本消息必填，最大5000字符） |
| media_url | string | ❌ | 媒体文件URL（图片/视频/语音） |
| video_id | uint | ❌ | 分享的视频ID |

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 123,
    "sender_id": 1,
    "receiver_id": 2,
    "type": 1,
    "content": "你好，在吗？",
    "media_url": "",
    "video_id": null,
    "is_read": false,
    "read_at": null,
    "created_at": "2026-04-17T18:00:00+08:00",
    "updated_at": "2026-04-17T18:00:00+08:00"
  }
}
```

---

### 1.2 获取会话列表

```http
GET /messages/conversations?page=1&page_size=20
Authorization: Bearer <token>
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "user1_id": 1,
        "user2_id": 2,
        "last_message_id": 123,
        "last_content": "你好，在吗？",
        "unread_count1": 0,
        "unread_count2": 1,
        "created_at": "2026-04-17T18:00:00+08:00",
        "updated_at": "2026-04-17T18:00:00+08:00",
        "user1": {
          "id": 1,
          "username": "alice",
          "nickname": "爱丽丝",
          "avatar": "http://localhost:8080/uploads/avatars/1.jpg"
        },
        "user2": {
          "id": 2,
          "username": "bob",
          "nickname": "鲍勃",
          "avatar": "http://localhost:8080/uploads/avatars/2.jpg"
        },
        "last_message": {
          "id": 123,
          "content": "你好，在吗？",
          "created_at": "2026-04-17T18:00:00+08:00"
        }
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 1.3 获取会话消息（聊天记录）

```http
GET /messages/conversations/{user_id}?page=1&page_size=20
Authorization: Bearer <token>
```

**路径参数**：
- `user_id`: 对方用户ID

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 123,
        "sender_id": 1,
        "receiver_id": 2,
        "type": 1,
        "content": "你好，在吗？",
        "media_url": "",
        "video_id": null,
        "is_read": true,
        "read_at": "2026-04-17T18:05:00+08:00",
        "created_at": "2026-04-17T18:00:00+08:00",
        "sender": {
          "id": 1,
          "username": "alice",
          "nickname": "爱丽丝",
          "avatar": "http://localhost:8080/uploads/avatars/1.jpg"
        },
        "receiver": {
          "id": 2,
          "username": "bob",
          "nickname": "鲍勃",
          "avatar": "http://localhost:8080/uploads/avatars/2.jpg"
        }
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 1.4 标记会话已读

```http
POST /messages/conversations/{user_id}/read
Authorization: Bearer <token>
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 1.5 标记单条消息已读

```http
POST /messages/{id}/read
Authorization: Bearer <token>
```

---

### 1.6 删除消息

```http
DELETE /messages/{id}
Authorization: Bearer <token>
```

---

### 1.7 获取未读消息数

```http
GET /messages/unread/count
Authorization: Bearer <token>
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "count": 5
  }
}
```

---

## 2. WebSocket 实时推送

### 2.1 连接 WebSocket

```javascript
const token = localStorage.getItem('token');
const ws = new WebSocket(`ws://localhost:8080/api/v1/messages/ws?token=${token}`);

// 或者在请求头中传递 token（推荐）
const ws = new WebSocket('ws://localhost:8080/api/v1/messages/ws');
// 注意：实际实现中需要在 HTTP 升级请求的 Authorization 头中传递 token
```

**重要**：WebSocket 连接需要先通过 JWT 认证，token 通过 HTTP 请求头传递（在升级握手时）。

### 2.2 接收实时消息

```javascript
ws.onopen = () => {
  console.log('WebSocket 已连接');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.type === 'message') {
    // 收到新消息
    const message = data.payload;
    console.log('收到新消息:', message);
    
    // 更新 UI：插入消息到聊天列表
    appendMessageToUI(message);
    
    // 更新未读数
    updateUnreadCount();
    
    // 如果当前在该会话页面，自动标记已读
    if (currentChatUserId === message.sender_id) {
      markConversationAsRead(message.sender_id);
    }
  }
};

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('WebSocket 已断开');
  // 重连逻辑
  setTimeout(() => reconnectWebSocket(), 3000);
};
```

### 2.3 推送消息格式

```json
{
  "type": "message",
  "payload": {
    "id": 124,
    "sender_id": 2,
    "receiver_id": 1,
    "type": 1,
    "content": "我在的，有什么事吗？",
    "media_url": "",
    "video_id": null,
    "is_read": false,
    "read_at": null,
    "created_at": "2026-04-17T18:01:00+08:00",
    "sender": {
      "id": 2,
      "username": "bob",
      "nickname": "鲍勃",
      "avatar": "http://localhost:8080/uploads/avatars/2.jpg"
    }
  }
}
```

---

## 3. 完整集成流程

### 3.1 进入聊天页面

```javascript
async function enterChatPage(targetUserId) {
  // 1. 拉取历史消息（首屏数据）
  const response = await fetch(
    `http://localhost:8080/api/v1/messages/conversations/${targetUserId}?page=1&page_size=20`,
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  );
  const data = await response.json();
  
  // 2. 渲染消息列表
  renderMessages(data.data.list);
  
  // 3. 标记会话已读
  await markConversationAsRead(targetUserId);
  
  // 4. 建立 WebSocket 连接（如果未连接）
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    connectWebSocket();
  }
}
```

### 3.2 发送消息

```javascript
async function sendMessage(receiverId, content, type = 1) {
  // 1. 调用 REST API 发送
  const response = await fetch('http://localhost:8080/api/v1/messages/send', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      receiver_id: receiverId,
      type: type,
      content: content
    })
  });
  
  const result = await response.json();
  
  if (result.code === 0) {
    // 2. 立即显示在 UI（乐观更新）
    appendMessageToUI(result.data);
    
    // 注意：对方会通过 WebSocket 收到推送，不需要轮询
  } else {
    console.error('发送失败:', result.message);
  }
}
```

### 3.3 接收消息

```javascript
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.type === 'message') {
    const message = data.payload;
    
    // 判断是否是当前会话
    if (currentChatUserId === message.sender_id) {
      // 当前会话：直接插入消息
      appendMessageToUI(message);
      
      // 自动标记已读
      markMessageAsRead(message.id);
    } else {
      // 其他会话：更新会话列表的未读数和最后一条消息
      updateConversationList(message);
      
      // 显示通知
      showNotification(`${message.sender.nickname}: ${message.content}`);
    }
  }
};
```

### 3.4 App 切后台/重新进入

```javascript
// App 进入前台
document.addEventListener('visibilitychange', async () => {
  if (!document.hidden) {
    // 1. 重连 WebSocket（如果断开）
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      connectWebSocket();
    }
    
    // 2. 拉取最新消息（补齐断开期间的消息）
    if (currentChatUserId) {
      await fetchLatestMessages(currentChatUserId);
    }
    
    // 3. 更新会话列表
    await fetchConversationList();
  }
});
```

---

## 4. 消息类型处理

### 4.1 文本消息 (type=1)

```javascript
{
  "receiver_id": 2,
  "type": 1,
  "content": "你好，在吗？"
}
```

### 4.2 图片消息 (type=2)

```javascript
// 1. 先上传图片到服务器
const formData = new FormData();
formData.append('file', imageFile);

const uploadRes = await fetch('http://localhost:8080/api/v1/upload/image', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: formData
});
const uploadData = await uploadRes.json();

// 2. 发送图片消息
{
  "receiver_id": 2,
  "type": 2,
  "content": "[图片]",
  "media_url": uploadData.data.url
}
```

### 4.3 视频消息 (type=3)

```javascript
{
  "receiver_id": 2,
  "type": 3,
  "content": "[视频]",
  "media_url": "http://localhost:8080/uploads/videos/xxx.mp4"
}
```

### 4.4 语音消息 (type=4)

```javascript
{
  "receiver_id": 2,
  "type": 4,
  "content": "[语音]",
  "media_url": "http://localhost:8080/uploads/audio/xxx.mp3"
}
```

---

## 5. 错误处理

### 5.1 常见错误码

| code | message | 说明 |
|------|---------|------|
| 0 | success | 成功 |
| 1 | 参数错误 | 请求参数不合法 |
| 3001 | 未登录 | Token 无效或过期 |
| 3003 | 接收者不存在 | receiver_id 无效 |
| 3004 | 不能给自己发送消息 | sender_id == receiver_id |
| 3005 | 接收者账号异常 | 对方账号被封禁 |

### 5.2 WebSocket 重连策略

```javascript
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

function connectWebSocket() {
  const ws = new WebSocket(`ws://localhost:8080/api/v1/messages/ws`);
  
  ws.onopen = () => {
    console.log('WebSocket 已连接');
    reconnectAttempts = 0; // 重置重连次数
  };
  
  ws.onclose = () => {
    console.log('WebSocket 已断开');
    
    if (reconnectAttempts < maxReconnectAttempts) {
      reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
      console.log(`${delay}ms 后重连...`);
      setTimeout(() => connectWebSocket(), delay);
    } else {
      console.error('WebSocket 重连失败，请刷新页面');
    }
  };
  
  return ws;
}
```

---

## 6. 性能优化建议

### 6.1 消息列表虚拟滚动

对于长聊天记录，使用虚拟滚动（如 `react-window`）只渲染可见区域的消息。

### 6.2 图片懒加载

```javascript
<img 
  src={message.media_url} 
  loading="lazy" 
  alt="图片消息"
/>
```

### 6.3 消息本地缓存

使用 IndexedDB 或 LocalStorage 缓存历史消息，减少网络请求。

### 6.4 防抖发送

```javascript
const debouncedSend = debounce(sendMessage, 300);
```

---

## 7. 安全注意事项

1. **XSS 防护**：渲染用户消息时，必须转义 HTML 标签
   ```javascript
   const escapeHtml = (text) => {
     return text.replace(/[&<>"']/g, (m) => ({
       '&': '&amp;',
       '<': '&lt;',
       '>': '&gt;',
       '"': '&quot;',
       "'": '&#39;'
     })[m]);
   };
   ```

2. **Token 安全**：不要在 URL 中传递 token，使用 HTTP 请求头

3. **内容过滤**：敏感词过滤应在后端完成

---

## 8. 示例代码（React）

```jsx
import { useEffect, useState, useRef } from 'react';

function ChatPage({ targetUserId }) {
  const [messages, setMessages] = useState([]);
  const [inputText, setInputText] = useState('');
  const wsRef = useRef(null);
  const token = localStorage.getItem('token');

  // 初始化
  useEffect(() => {
    // 1. 拉取历史消息
    fetchMessages();
    
    // 2. 建立 WebSocket
    connectWebSocket();
    
    // 3. 标记已读
    markAsRead();
    
    return () => {
      // 清理 WebSocket
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [targetUserId]);

  const fetchMessages = async () => {
    const res = await fetch(
      `http://localhost:8080/api/v1/messages/conversations/${targetUserId}?page=1&page_size=50`,
      { headers: { 'Authorization': `Bearer ${token}` } }
    );
    const data = await res.json();
    setMessages(data.data.list.reverse()); // 时间升序
  };

  const connectWebSocket = () => {
    const ws = new WebSocket('ws://localhost:8080/api/v1/messages/ws');
    
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'message' && data.payload.sender_id === targetUserId) {
        setMessages(prev => [...prev, data.payload]);
        markAsRead();
      }
    };
    
    wsRef.current = ws;
  };

  const sendMessage = async () => {
    if (!inputText.trim()) return;
    
    const res = await fetch('http://localhost:8080/api/v1/messages/send', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        receiver_id: targetUserId,
        type: 1,
        content: inputText
      })
    });
    
    const data = await res.json();
    if (data.code === 0) {
      setMessages(prev => [...prev, data.data]);
      setInputText('');
    }
  };

  const markAsRead = async () => {
    await fetch(
      `http://localhost:8080/api/v1/messages/conversations/${targetUserId}/read`,
      {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      }
    );
  };

  return (
    <div>
      <div className="messages">
        {messages.map(msg => (
          <div key={msg.id} className={msg.sender_id === currentUserId ? 'sent' : 'received'}>
            {msg.content}
          </div>
        ))}
      </div>
      <input 
        value={inputText} 
        onChange={(e) => setInputText(e.target.value)}
        onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
      />
      <button onClick={sendMessage}>发送</button>
    </div>
  );
}
```

---

## 9. 测试建议

### 9.1 功能测试

- [ ] 发送文本消息
- [ ] 发送图片/视频/语音消息
- [ ] 接收实时推送
- [ ] 标记已读
- [ ] 删除消息
- [ ] 会话列表更新
- [ ] 未读数统计

### 9.2 边界测试

- [ ] 网络断开重连
- [ ] App 切后台再回来
- [ ] 多端同时登录
- [ ] 发送空消息（应拦截）
- [ ] 发送超长消息（5000字符限制）
- [ ] 给不存在的用户发消息

### 9.3 性能测试

- [ ] 1000+ 条消息的渲染性能
- [ ] WebSocket 长时间保持连接
- [ ] 频繁发送消息（防抖）

---

## 10. 常见问题

**Q: WebSocket 连接后立即断开？**  
A: 检查 token 是否有效，确保在 HTTP 升级请求的 Authorization 头中传递。

**Q: 发送消息后对方收不到推送？**  
A: 确认对方已建立 WebSocket 连接，检查服务端日志是否有推送失败记录。

**Q: 消息重复显示？**  
A: 发送消息后不要同时监听 WebSocket 推送，发送者的消息应该在 REST API 响应后立即显示。

**Q: 如何实现消息撤回？**  
A: 当前版本不支持，需要后端添加 `DELETE /messages/:id/recall` 接口。

**Q: 如何实现群聊？**  
A: 当前版本仅支持单聊，群聊需要重新设计数据模型（Group、GroupMember 表）。

---

## 11. 相关文档

- [OpenAPI 规范](../openapi.json) - 完整的 API 定义
- [消息模型](../internal/model/message.go) - 数据结构定义
- [消息服务](../internal/service/message_service.go) - 业务逻辑实现

---

**最后更新**: 2026-04-17  
**版本**: v1.0.0
