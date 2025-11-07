# OpenAPI 更新清单

## 需要添加的新 API 接口

根据 `internal/router/router.go` 的路由定义，以下接口需要添加到 `openapi.json`:

### 1. OAuth 认证接口

```
GET  /api/v1/oauth/login      - OAuth 登录跳转
GET  /api/v1/oauth/callback   - OAuth 回调处理
```

### 2. 搜索相关接口

```
GET  /api/v1/search                    - 综合搜索
GET  /api/v1/search/videos             - 搜索视频
GET  /api/v1/search/users              - 搜索用户
GET  /api/v1/search/hashtags           - 搜索话题
GET  /api/v1/search/hot                - 热搜榜
GET  /api/v1/search/suggestions        - 搜索建议
GET  /api/v1/search/history           - 搜索历史（需登录）
DELETE /api/v1/search/history         - 清空搜索历史（需登录）
```

### 3. 消息相关接口

```
GET    /api/v1/messages/conversations                   - 会话列表（需登录）
GET    /api/v1/messages/conversations/:user_id          - 会话消息（需登录）
POST   /api/v1/messages/send                            - 发送消息（需登录）
POST   /api/v1/messages/conversations/:user_id/read     - 标记会话已读（需登录）
POST   /api/v1/messages/:id/read                        - 标记消息已读（需登录）
DELETE /api/v1/messages/:id                             - 删除消息（需登录）
GET    /api/v1/messages/unread/count                    - 未读消息数（需登录）
```

### 4. 通知相关接口

```
GET  /api/v1/notifications                    - 通知列表（需登录）
POST /api/v1/notifications/:id/read           - 标记通知已读（需登录）
POST /api/v1/notifications/read-all           - 标记所有通知已读（需登录）
GET  /api/v1/notifications/unread/count       - 未读通知数（需登录）
```

### 5. 话题标签相关接口

```
GET  /api/v1/hashtags/hot             - 热门话题
GET  /api/v1/hashtags/:id             - 话题详情
GET  /api/v1/hashtags/:id/videos      - 话题视频列表
POST /api/v1/hashtags                 - 创建话题（需登录）
```

## 已存在但可能需要更新的接口

### 视频接口
- ✅ `/api/v1/videos/feed` - 已存在
- ✅ `/api/v1/videos/hot` - 已存在
- ✅ `/api/v1/videos/follow` - 已存在

### 用户接口
- ✅ `/api/v1/users/:id` - 已存在
- ✅ `/api/v1/users/me` - 已存在
- ✅ `/api/v1/users/:id/follow` - 已存在
- ✅ `/api/v1/users/:id/videos` - 已存在

### 评论接口
- ✅ `/api/v1/comments` - 已存在
- ✅ `/api/v1/comments/:id` - 已存在
- ✅ `/api/v1/comments/video/:video_id` - 已存在
- ✅ `/api/v1/comments/:id/like` - 已存在

### 直播接口
- ✅ `/api/v1/live/list` - 已存在
- ✅ `/api/v1/live/:id` - 已存在
- ✅ `/api/v1/live/room/:room_id` - 已存在
- ✅ `/api/v1/live/create` - 已存在
- ✅ `/api/v1/live/start` - 已存在
- ✅ `/api/v1/live/end` - 已存在
- ✅ `/api/v1/live/my` - 已存在
- ✅ `/api/v1/live/join/:room_id` - 已存在
- ✅ `/api/v1/live/leave/:room_id` - 已存在
- ✅ `/api/v1/live/ws` - 已存在

## 需要添加的 Schema 定义

### 搜索相关

```json
{
  "SearchRequest": {
    "type": "object",
    "properties": {
      "keyword": {
        "type": "string",
        "description": "搜索关键词"
      },
      "type": {
        "type": "string",
        "enum": ["all", "video", "user", "hashtag"],
        "description": "搜索类型"
      }
    }
  },
  "Hashtag": {
    "type": "object",
    "properties": {
      "id": {"type": "integer"},
      "name": {"type": "string"},
      "description": {"type": "string"},
      "cover": {"type": "string"},
      "video_count": {"type": "integer"},
      "view_count": {"type": "integer"},
      "created_at": {"type": "string", "format": "date-time"}
    }
  }
}
```

### 消息相关

```json
{
  "Message": {
    "type": "object",
    "properties": {
      "id": {"type": "integer"},
      "from_user_id": {"type": "integer"},
      "to_user_id": {"type": "integer"},
      "content": {"type": "string"},
      "content_type": {"type": "integer"},
      "is_read": {"type": "boolean"},
      "created_at": {"type": "string", "format": "date-time"}
    }
  },
  "Conversation": {
    "type": "object",
    "properties": {
      "user_id": {"type": "integer"},
      "username": {"type": "string"},
      "avatar": {"type": "string"},
      "last_message": {"$ref": "#/components/schemas/Message"},
      "unread_count": {"type": "integer"},
      "updated_at": {"type": "string", "format": "date-time"}
    }
  }
}
```

### 通知相关

```json
{
  "Notification": {
    "type": "object",
    "properties": {
      "id": {"type": "integer"},
      "user_id": {"type": "integer"},
      "type": {"type": "string"},
      "title": {"type": "string"},
      "content": {"type": "string"},
      "data": {"type": "object"},
      "is_read": {"type": "boolean"},
      "created_at": {"type": "string", "format": "date-time"}
    }
  }
}
```

## 更新建议

由于添加的接口较多，建议：

1. **手动编辑 openapi.json** 添加上述接口定义
2. **或使用工具自动生成** - 使用 swag 等工具从代码注释生成
3. **或使用在线编辑器** - https://editor.swagger.io/ 导入编辑

## 验证工具

更新后使用以下工具验证：
```bash
# 安装 openapi-generator-cli
npm install -g @open api/openapi-generator-cli

# 验证 JSON 格式
npx openapi-generator-cli validate -i openapi.json
```
