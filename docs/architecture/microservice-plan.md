# MicroVibe-Go 微服务改造方案

## 一、服务拆分设计

### 1. 用户服务 (User Service)
**端口**: 8001
**职责**:
- 用户注册/登录/认证
- 用户资料管理
- 关注/粉丝关系
- 用户统计

**数据库**:
- `users` 表
- `user_profiles` 表
- `follows` 表

**API**:
```
POST   /api/v1/users/register
POST   /api/v1/users/login
GET    /api/v1/users/:id
PUT    /api/v1/users/:id
POST   /api/v1/users/:id/follow
DELETE /api/v1/users/:id/follow
GET    /api/v1/users/:id/followers
GET    /api/v1/users/:id/following
```

---

### 2. 视频服务 (Video Service)
**端口**: 8002
**职责**:
- 视频上传/删除/更新
- 视频详情查询
- 视频点赞/收藏
- 视频评论

**数据库**:
- `videos` 表
- `likes` 表
- `favorites` 表
- `comments` 表
- `comment_likes` 表

**API**:
```
POST   /api/v1/videos
GET    /api/v1/videos/:id
PUT    /api/v1/videos/:id
DELETE /api/v1/videos/:id
POST   /api/v1/videos/:id/like
POST   /api/v1/videos/:id/favorite
GET    /api/v1/videos/:id/comments
POST   /api/v1/videos/:id/comments
```

---

### 3. 直播服务 (Live Service)
**端口**: 8003
**职责**:
- 直播间管理
- 直播礼物系统
- 直播粉丝团
- 直播商品
- WebSocket 信令

**数据库**:
- `live_streams` 表
- `live_gifts` 表
- `live_gift_records` 表
- `live_fan_clubs` 表
- `live_fan_club_members` 表
- `live_products` 表

**API**:
```
POST   /api/v1/lives
GET    /api/v1/lives/:id
POST   /api/v1/lives/:id/start
POST   /api/v1/lives/:id/end
POST   /api/v1/lives/:id/join
WS     /api/v1/lives/:id/ws
POST   /api/v1/lives/:id/gifts
GET    /api/v1/lives/:id/products
```

---

### 4. 推荐服务 (Recommend Service)
**端口**: 8004
**职责**:
- 视频推荐算法
- 召回层（协同过滤、内容召回、热门召回）
- 排序层（CTR 预估、完播率）
- 特征工程

**数据库**:
- Redis (用户特征、视频特征缓存)

**API**:
```
GET    /api/v1/recommend/feed          # 推荐流
GET    /api/v1/recommend/following     # 关注流
GET    /api/v1/recommend/hot           # 热门流
POST   /api/v1/recommend/refresh       # 刷新推荐
```

---

### 5. 搜索服务 (Search Service)
**端口**: 8005
**职责**:
- 综合搜索（视频/用户/话题）
- 搜索历史
- 热搜榜单
- 搜索建议

**数据库**:
- `search_history` 表
- `hot_searches` 表
- Elasticsearch (可选，用于全文搜索)

**API**:
```
GET    /api/v1/search                  # 综合搜索
GET    /api/v1/search/videos           # 视频搜索
GET    /api/v1/search/users            # 用户搜索
GET    /api/v1/search/hashtags         # 话题搜索
GET    /api/v1/search/hot              # 热搜榜
GET    /api/v1/search/suggestions      # 搜索建议
```

---

### 6. 消息服务 (Message Service)
**端口**: 8006
**职责**:
- 私信发送/接收
- 会话管理
- 系统通知
- 点赞/评论/关注通知

**数据库**:
- `messages` 表
- `conversations` 表
- `notifications` 表

**API**:
```
POST   /api/v1/messages                # 发送私信
GET    /api/v1/messages/:id            # 获取消息
GET    /api/v1/conversations           # 会话列表
GET    /api/v1/notifications           # 通知列表
PUT    /api/v1/notifications/:id/read  # 标记已读
```

---

### 7. 话题服务 (Hashtag Service)
**端口**: 8007
**职责**:
- 话题创建/查询
- 话题热门榜
- 话题视频关联

**数据库**:
- `hashtags` 表
- `video_hashtags` 表

**API**:
```
POST   /api/v1/hashtags                # 创建话题
GET    /api/v1/hashtags/:id            # 话题详情
GET    /api/v1/hashtags/hot            # 热门话题
GET    /api/v1/hashtags/:id/videos     # 话题视频列表
```

---

### 8. 媒体服务 (Media Service) - 新增
**端口**: 8008
**职责**:
- 文件上传（视频/图片）
- 视频转码（使用 FFmpeg）
- CDN 分发
- 缩略图生成

**存储**:
- OSS (阿里云/AWS S3/MinIO)

**API**:
```
POST   /api/v1/media/upload            # 上传文件
GET    /api/v1/media/:id               # 获取文件信息
POST   /api/v1/media/:id/transcode     # 转码任务
```

---

## 二、基础设施服务

### 1. API Gateway (网关)
**推荐**: Kong / APISIX / Traefik

**功能**:
- 统一入口
- 路由转发
- JWT 认证
- 限流熔断
- 日志收集

**配置示例 (Kong)**:
```yaml
services:
  - name: user-service
    url: http://user-service:8001
    routes:
      - name: user-routes
        paths:
          - /api/v1/users
    plugins:
      - name: jwt
      - name: rate-limiting
        config:
          minute: 100
```

---

### 2. Service Registry (服务注册中心)
**推荐**: Consul / Nacos / Etcd

**功能**:
- 服务注册/发现
- 健康检查
- 配置中心

**Consul 示例**:
```go
// 服务注册
registration := &consul.AgentServiceRegistration{
    ID:      "user-service-1",
    Name:    "user-service",
    Port:    8001,
    Address: "192.168.1.10",
    Check: &consul.AgentServiceCheck{
        HTTP:     "http://192.168.1.10:8001/health",
        Interval: "10s",
        Timeout:  "5s",
    },
}
client.Agent().ServiceRegister(registration)
```

---

### 3. Message Queue (消息队列)
**推荐**: Kafka / NATS / RabbitMQ

**使用场景**:
- 异步事件处理（点赞、评论、关注）
- 视频转码任务队列
- 消息推送队列
- 数据同步

**事件示例**:
```go
// 视频点赞事件
type VideoLikedEvent struct {
    VideoID   uint      `json:"video_id"`
    UserID    uint      `json:"user_id"`
    Timestamp time.Time `json:"timestamp"`
}

// 发布到 Kafka
producer.Publish("video.liked", event)

// 订阅并处理
consumer.Subscribe("video.liked", func(msg *Message) {
    // 更新点赞数
    // 发送通知
    // 更新推荐算法
})
```

---

### 4. Distributed Tracing (分布式链路追踪)
**推荐**: Jaeger / Zipkin / SkyWalking

**功能**:
- 追踪跨服务调用
- 性能分析
- 错误定位

**集成示例 (Jaeger)**:
```go
import "github.com/opentracing/opentracing-go"

// 创建 Span
span := opentracing.StartSpan("user.get")
defer span.Finish()

// 传递到下游服务
ctx := opentracing.ContextWithSpan(context.Background(), span)
videoService.GetVideo(ctx, videoID)
```

---

### 5. Monitoring & Logging (监控和日志)
**推荐**:
- Metrics: Prometheus + Grafana
- Logging: ELK (Elasticsearch + Logstash + Kibana) / Loki

**Prometheus 指标示例**:
```go
import "github.com/prometheus/client_golang/prometheus"

var (
    httpRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"service", "method", "status"},
    )
)

// 记录请求
httpRequests.WithLabelValues("user-service", "GET", "200").Inc()
```

---

## 三、改造步骤

### 阶段 1: 准备工作（1-2 周）

1. **选择微服务框架**
   ```bash
   # 安装 Kratos CLI
   go install github.com/go-kratos/kratos/cmd/kratos/v2@latest

   # 创建服务模板
   kratos new user-service
   kratos new video-service
   kratos new live-service
   ```

2. **搭建基础设施**
   - 部署 Consul (服务注册)
   - 部署 Kafka (消息队列)
   - 部署 Jaeger (链路追踪)
   - 部署 Prometheus + Grafana (监控)

3. **定义服务接口（Proto）**
   ```protobuf
   // user.proto
   syntax = "proto3";
   package api.user.v1;

   service UserService {
       rpc Register(RegisterRequest) returns (RegisterReply);
       rpc Login(LoginRequest) returns (LoginReply);
       rpc GetUser(GetUserRequest) returns (GetUserReply);
       rpc UpdateUser(UpdateUserRequest) returns (UpdateUserReply);
   }

   message RegisterRequest {
       string username = 1;
       string password = 2;
       string email = 3;
   }

   message RegisterReply {
       uint32 user_id = 1;
       string token = 2;
   }
   ```

---

### 阶段 2: 拆分用户服务（1-2 周）

1. **创建 User Service**
   ```bash
   # 项目结构
   user-service/
   ├── api/               # Proto 定义
   ├── cmd/               # 启动入口
   ├── internal/
   │   ├── biz/          # 业务逻辑
   │   ├── data/         # 数据访问
   │   ├── service/      # gRPC/HTTP 服务实现
   │   └── conf/         # 配置
   └── go.mod
   ```

2. **迁移代码**
   - 复制 `internal/model/user.go`
   - 复制 `internal/repository/user_repository.go` → `data/`
   - 复制 `internal/service/user_service.go` → `biz/`
   - 重构 Handler → `service/user.go`

3. **集成 Consul**
   ```go
   // 服务注册
   import "github.com/go-kratos/kratos/contrib/registry/consul/v2"

   consulClient, _ := consul.NewClient(consul.DefaultConfig())
   registry := consul.New(consulClient)

   app := kratos.New(
       kratos.Name("user-service"),
       kratos.Registrar(registry),
   )
   ```

4. **测试**
   - 单元测试
   - 集成测试
   - gRPC 调用测试

---

### 阶段 3: 拆分其他服务（3-4 周）

按优先级逐步拆分：
1. Video Service
2. Live Service
3. Recommend Service
4. Search Service
5. Message Service
6. Hashtag Service
7. Media Service

**每个服务重复阶段 2 的步骤**

---

### 阶段 4: 部署 API Gateway（1 周）

1. **安装 Kong**
   ```bash
   docker run -d --name kong \
       -e "KONG_DATABASE=postgres" \
       -e "KONG_PG_HOST=postgres" \
       -p 8000:8000 \
       -p 8001:8001 \
       kong:latest
   ```

2. **配置路由**
   ```bash
   # 用户服务路由
   curl -X POST http://localhost:8001/services/ \
       --data name=user-service \
       --data url=http://user-service:8001

   curl -X POST http://localhost:8001/services/user-service/routes \
       --data 'paths[]=/api/v1/users'
   ```

3. **配置 JWT 认证**
   ```bash
   curl -X POST http://localhost:8001/services/user-service/plugins \
       --data name=jwt
   ```

---

### 阶段 5: 服务间通信改造（2-3 周）

1. **使用 gRPC 调用**
   ```go
   // Video Service 调用 User Service
   import pb "user-service/api/user/v1"

   conn, _ := grpc.Dial("user-service:9000")
   client := pb.NewUserServiceClient(conn)

   user, err := client.GetUser(ctx, &pb.GetUserRequest{
       UserId: 123,
   })
   ```

2. **使用消息队列解耦**
   ```go
   // Video Service 发布点赞事件
   event := &VideoLikedEvent{
       VideoID: 456,
       UserID:  123,
   }
   kafka.Publish("video.liked", event)

   // Message Service 订阅并发送通知
   kafka.Subscribe("video.liked", func(event *VideoLikedEvent) {
       notification := createNotification(event)
       notificationRepo.Create(notification)
   })
   ```

---

### 阶段 6: 监控和日志（1-2 周）

1. **集成 Prometheus**
   ```go
   // 暴露 metrics 端点
   http.Handle("/metrics", promhttp.Handler())
   ```

2. **集成 Jaeger**
   ```go
   import "github.com/go-kratos/kratos/v2/middleware/tracing"

   app := kratos.New(
       kratos.Server(
           http.NewServer(
               http.Middleware(
                   tracing.Server(),
               ),
           ),
       ),
   )
   ```

3. **统一日志格式**
   ```go
   import "github.com/go-kratos/kratos/v2/log"

   logger := log.With(log.DefaultLogger,
       "service", "user-service",
       "version", "v1.0.0",
   )
   ```

---

## 四、数据库拆分策略

### 1. 按服务拆分数据库

```sql
-- User Service Database
CREATE DATABASE microvibe_user;
-- Tables: users, user_profiles, follows

-- Video Service Database
CREATE DATABASE microvibe_video;
-- Tables: videos, likes, favorites, comments, comment_likes

-- Live Service Database
CREATE DATABASE microvibe_live;
-- Tables: live_streams, live_gifts, live_gift_records, ...

-- Message Service Database
CREATE DATABASE microvibe_message;
-- Tables: messages, conversations, notifications

-- Search Service Database
CREATE DATABASE microvibe_search;
-- Tables: search_history, hot_searches

-- Hashtag Service Database
CREATE DATABASE microvibe_hashtag;
-- Tables: hashtags, video_hashtags
```

### 2. 处理跨库 JOIN

**问题**: 无法跨服务 JOIN 查询

**解决方案**:

**方案 1: 服务间 RPC 调用**
```go
// Video Service 需要获取视频 + 用户信息
videos := videoRepo.FindByUserID(userID)

// 调用 User Service 获取用户信息
userIDs := extractUserIDs(videos)
users := userServiceClient.GetUsersByIDs(ctx, userIDs)

// 组装数据
videoWithUsers := mergeVideoAndUser(videos, users)
```

**方案 2: 数据冗余**
```go
// Video 表中冗余部分用户信息
type Video struct {
    ID          uint
    UserID      uint
    Username    string  // 冗余
    UserAvatar  string  // 冗余
    Title       string
    // ...
}

// 用户更新时通过消息队列同步
kafka.Subscribe("user.updated", func(event *UserUpdatedEvent) {
    videoRepo.UpdateUserInfo(event.UserID, event.Username, event.Avatar)
})
```

**方案 3: CQRS (读写分离)**
```go
// 写入: 分别写入各服务数据库
// 读取: 从统一的读库查询（通过 CDC 同步）

// Debezium CDC 同步到 Elasticsearch
videos_with_user_info (ES Index)
{
    "video_id": 1,
    "user_id": 123,
    "username": "john",
    "title": "Video Title",
    // ...
}
```

---

## 五、部署架构（Kubernetes）

```yaml
# user-service-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
    spec:
      containers:
      - name: user-service
        image: microvibe/user-service:v1.0.0
        ports:
        - containerPort: 8001
          name: http
        - containerPort: 9001
          name: grpc
        env:
        - name: DB_HOST
          value: postgres-user
        - name: REDIS_HOST
          value: redis
        - name: CONSUL_ADDR
          value: consul:8500
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8001
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: user-service
spec:
  selector:
    app: user-service
  ports:
  - name: http
    port: 8001
    targetPort: 8001
  - name: grpc
    port: 9001
    targetPort: 9001
```

---

## 六、成本估算

### 1. 开发成本
- 框架学习: 1 周
- 基础设施搭建: 1-2 周
- 服务拆分: 6-8 周
- 测试调优: 2-3 周
- **总计: 10-14 周（2.5-3.5 个月）**

### 2. 人力成本
- 后端开发: 2-3 人
- DevOps: 1 人
- 测试: 1 人

### 3. 基础设施成本 (云服务)
- API Gateway: ¥500/月
- Service Registry (Consul): ¥300/月
- Message Queue (Kafka): ¥800/月
- Database (RDS): ¥2000/月
- Redis: ¥500/月
- Monitoring: ¥300/月
- **总计: ¥4400/月**

---

## 七、技术栈推荐

```yaml
# 微服务框架
framework: Kratos / Go-Micro

# 服务间通信
rpc: gRPC
serialization: Protobuf

# 服务注册发现
registry: Consul / Nacos

# API 网关
gateway: Kong / APISIX

# 消息队列
mq: Kafka / NATS

# 配置中心
config: Consul KV / Nacos Config

# 链路追踪
tracing: Jaeger

# 监控告警
monitoring: Prometheus + Grafana

# 日志收集
logging: Loki / ELK

# 容器编排
orchestration: Kubernetes

# CI/CD
cicd: GitLab CI / GitHub Actions

# 数据库
database: PostgreSQL (分库)
cache: Redis (分布式)

# 对象存储
storage: MinIO / Aliyun OSS
```

---

## 八、最佳实践

### 1. 渐进式迁移
- 先拆分独立性强的服务（如消息、搜索）
- 保留单体应用，逐步迁移流量
- 使用 Strangler Fig 模式

### 2. 服务治理
- 统一错误码
- 统一日志格式
- 统一监控指标
- 统一配置管理

### 3. 数据一致性
- 使用分布式事务（Saga / TCC）
- 最终一致性（消息队列 + 重试）
- 事件溯源（Event Sourcing）

### 4. 服务安全
- 服务间 mTLS 认证
- API Gateway JWT 验证
- 限流熔断
- 黑白名单

### 5. 测试策略
- 单元测试（各服务独立）
- 集成测试（服务间调用）
- 端到端测试（完整流程）
- 性能测试（压力测试）

---

## 九、是否需要微服务化？

### ✅ 需要微服务的场景
- 团队规模 > 20 人
- 业务复杂度高，模块耦合严重
- 不同模块技术栈不同
- 需要独立部署和扩展
- 高可用要求（99.99%）

### ❌ 不建议微服务的场景
- 团队 < 5 人
- 业务简单，流量不大（< 1000 QPS）
- 快速迭代阶段
- 缺乏 DevOps 能力

### 🤔 您的项目建议
根据当前代码规模和架构：
1. **如果是学习/练习项目**: 建议先保持单体，学习微服务概念
2. **如果是商业项目且团队 < 10 人**: 建议采用**模块化单体**（Modular Monolith）
3. **如果是大型团队**: 可以采用微服务，但建议分阶段改造

---

## 十、快速开始（示例）

我可以帮您创建一个基于 **Kratos** 的 User Service 示例，演示完整的微服务改造过程。是否需要我继续？

选项:
- A. 创建 Kratos User Service 完整示例
- B. 创建 Go-Micro 示例
- C. 创建基于 gRPC 的轻量级方案
- D. 先看看模块化单体架构（更适合当前阶段）
