# MicroVibe-Go

基于 AI 推荐算法的多端短视频平台后端系统

## 项目简介

MicroVibe-Go 是一个完整的短视频平台后端解决方案，对标抖音的核心功能。项目采用 Go 语言开发，集成了自研的推荐算法引擎，支持视频上传、推荐、社交互动、直播等完整功能。

## 核心特性

### 基础架构
- **三层架构**：Model-Repository-Service-Handler 清晰分层
- **Gin 框架**：高性能的 Web 框架
- **GORM**：强大的 ORM 框架，支持 PostgreSQL
- **Redis**：用于缓存、会话管理和实时数据
- **Zap 日志**：Uber 开源的高性能结构化日志框架
- **Docker**：完整的容器化支持
- **JWT**：安全的用户认证机制

### 核心功能
- **用户系统**：注册、登录、个人主页、用户关系
- **视频功能**：上传、播放、点赞、收藏、分享
- **推荐算法**：基于用户行为的智能推荐引擎
- **社交互动**：评论、弹幕、关注、粉丝
- **消息系统**：私信、系统通知、互动提醒
- **直播功能**：直播推流、观看、互动
- **搜索功能**：视频搜索、用户搜索、热搜榜
- **话题标签**：话题创建、参与、热门话题

## 项目架构

```
microvibe-go/
├── cmd/
│   ├── server/              # 应用入口
│   └── migrate/             # 数据库迁移工具
├── internal/
│   ├── model/               # 数据模型（Model 层）
│   ├── repository/          # 数据访问层（Repository 层）⭐
│   ├── service/             # 业务逻辑层（Service 层）⭐
│   ├── handler/             # HTTP 处理器（Handler 层）⭐
│   ├── middleware/          # 中间件（认证、日志等）
│   ├── router/              # 路由定义
│   ├── config/              # 配置管理
│   ├── database/            # 数据库连接和迁移
│   └── algorithm/           # 推荐算法引擎
│       ├── recommend/       # 推荐算法核心
│       ├── feature/         # 特征工程
│       ├── rank/            # 排序算法
│       └── filter/          # 过滤策略
├── pkg/
│   ├── logger/              # 日志工具（Zap）⭐
│   ├── cache/               # 缓存框架（内存/Redis/多级）⭐
│   ├── event/               # 事件系统（EventBus）
│   ├── errors/              # 错误处理（统一错误定义）
│   ├── response/            # 统一响应格式
│   └── utils/               # 工具函数（JWT、密码加密等）
├── configs/                 # 配置文件
│   ├── config.yaml          # 应用配置
│   ├── nginx.conf           # Nginx 配置
│   ├── Caddyfile            # Caddy 配置
│   └── sfu.toml             # SFU 配置
├── docs/                    # 文档
│   ├── architecture/        # 架构文档
│   │   ├── system-architecture.md         # 系统架构
│   │   ├── recommendation-algorithm.md    # 推荐算法详解
│   │   ├── streaming-architecture.md      # 直播架构
│   │   ├── event-driven-architecture.md   # 事件驱动架构
│   │   ├── microservice-plan.md           # 微服务规划
│   │   └── kratos-migration-guide.md      # Kratos 迁移指南
│   ├── development/         # 开发文档
│   │   ├── quick-start.md               # 快速开始
│   │   ├── cache-framework.md           # 缓存框架使用指南 ⭐
│   │   ├── error-handling.md            # 错误处理指南
│   │   ├── pre-commit-guide.md          # Git 提交规范
│   │   ├── live-streaming-guide.md      # 直播功能开发指南
│   │   ├── obs-streaming.md             # OBS 推流配置
│   │   └── sfu-quickstart.md            # SFU 快速开始
│   ├── integration/         # 集成文档
│   │   ├── authentik-sso.md             # Authentik SSO 集成
│   │   ├── ion-sfu.md                   # Ion SFU 集成
│   │   └── ion-sfu-deployment.md        # Ion SFU 部署
│   ├── archived/            # 历史文档（已归档）
│   └── README.md            # 文档索引
├── examples/                # 示例代码
│   ├── cache/               # 缓存示例
│   ├── event/               # 事件示例
│   ├── webrtc-broadcaster.html    # WebRTC 推流示例
│   ├── webrtc-viewer.html         # WebRTC 观看示例
│   └── device-test.html           # 设备测试页面
├── scripts/                 # 脚本工具
│   ├── setup-oauth.sh       # OAuth 配置脚本
│   └── start-authentik.sh   # Authentik 启动脚本
├── .git-hooks/              # Git Hooks
│   └── commit-msg           # 提交信息检查
├── .pre-commit-config.yaml  # Pre-commit 配置
├── openapi.json             # OpenAPI 文档
├── Dockerfile               # Docker 镜像
├── docker-compose.yml       # 容器编排
├── docker-compose.authentik.yml  # Authentik 容器编排
├── Makefile                 # 便捷命令
├── CLAUDE.md                # Claude Code 开发指南 ⭐
└── PROGRESS.md              # 项目进度跟踪 ⭐
```

**四层架构说明**：
- **Model 层** (`internal/model/`): 数据模型定义（GORM）
- **Repository 层** (`internal/repository/`): 数据访问，封装数据库 CRUD 操作
- **Service 层** (`internal/service/`): 业务逻辑，调用 Repository
- **Handler 层** (`internal/handler/`): HTTP 请求处理，调用 Service

**核心功能包** (`pkg/`):
- **cache**: 高性能泛型缓存框架（内存/Redis/多级缓存）
- **event**: 事件驱动系统（发布订阅模式）
- **errors**: 统一错误处理（业务错误、数据库错误）
- **logger**: Zap 结构化日志
- **response**: 统一 HTTP 响应格式

详细架构说明请参考：
- [系统架构](./docs/architecture/system-architecture.md)
- [推荐算法](./docs/architecture/recommendation-algorithm.md)
- [直播架构](./docs/architecture/streaming-architecture.md)
- [事件驱动架构](./docs/architecture/event-driven-architecture.md)

## 数据模型

### 核心表结构
- **users**：用户信息表
- **videos**：视频信息表
- **comments**：评论表
- **likes**：点赞记录表
- **favorites**：收藏记录表
- **follows**：关注关系表
- **messages**：消息表
- **live_streams**：直播间表
- **hashtags**：话题标签表
- **user_behaviors**：用户行为表（推荐算法使用）

## 推荐算法架构

### 算法流程
1. **召回层**：从海量视频中快速召回候选集
   - 协同过滤召回
   - 内容召回（标签、分类）
   - 热门召回
   - 关注召回

2. **特征工程**：提取用户和视频特征
   - 用户特征：观看历史、互动行为、兴趣标签
   - 视频特征：分类、标签、热度、质量分
   - 交叉特征：用户-视频匹配度

3. **排序层**：精准排序推荐结果
   - 基于点击率预估（CTR）
   - 完播率预估
   - 互动率预估
   - 多目标融合排序

4. **过滤层**：内容过滤和去重
   - 已观看视频过滤
   - 相似视频去重
   - 低质量内容过滤

### 算法特点
- **实时性**：基于 Redis 的实时特征更新
- **个性化**：深度学习用户兴趣
- **多样性**：避免信息茧房，保证内容多样性
- **冷启动**：新用户和新视频的冷启动策略

## 技术栈

### 后端技术
- **Go 1.23+**：主要开发语言
- **Gin**：Web 框架
- **GORM**：ORM 框架
- **Zap**：高性能结构化日志框架（Uber 开源）
- **JWT-Go**：JWT 认证
- **Viper**：配置管理
- **bcrypt**：密码加密

### 数据存储
- **PostgreSQL 16**：主数据库
- **Redis 7**：缓存和会话
- **MinIO/OSS**：对象存储（视频文件）

### 中间件
- **Nginx**：反向代理和负载均衡
- **FFmpeg**：视频处理
- **消息队列**：异步任务处理

## 环境要求

- Go 1.23+
- Docker & Docker Compose
- PostgreSQL 16
- Redis 7
- MinIO（可选，用于本地对象存储）

## 快速开始

### 使用 Docker Compose（推荐）

1. 启动所有服务：
```bash
docker-compose up -d
```

2. 查看服务状态：
```bash
docker-compose ps
```

3. 查看日志：
```bash
docker-compose logs -f app
```

4. 停止服务：
```bash
docker-compose down
```

### 本地开发

1. 安装依赖：
```bash
go mod download
```

2. 配置数据库和 Redis（确保服务已启动）

3. 运行应用：
```bash
make run
# 或
go run cmd/server/main.go
```

4. 安装 Pre-commit Hooks（推荐）：
```bash
# macOS
brew install pre-commit

# 安装 hooks
make pre-commit-install
```

这将自动配置 Git 提交规范检查，包括：
- Go 代码格式化（gofmt、go imports）
- 代码质量检查（go vet、go build）
- Conventional Commits 提交信息规范

详细使用说明请参考：[Pre-commit 使用指南](docs/development/pre-commit-guide.md)

## API 文档

### 用户模块
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `GET /api/v1/users/:id` - 获取用户信息
- `PUT /api/v1/users/:id` - 更新用户信息

### 视频模块
- `POST /api/v1/videos` - 上传视频
- `GET /api/v1/videos/:id` - 获取视频详情
- `GET /api/v1/videos/feed` - 获取推荐视频流
- `DELETE /api/v1/videos/:id` - 删除视频

### 社交模块
- `POST /api/v1/videos/:id/like` - 点赞视频
- `POST /api/v1/videos/:id/comment` - 评论视频
- `POST /api/v1/users/:id/follow` - 关注用户
- `GET /api/v1/users/:id/followers` - 获取粉丝列表

### 直播模块
- `POST /api/v1/live/start` - 开始直播
- `POST /api/v1/live/:id/stop` - 结束直播
- `GET /api/v1/live/list` - 获取直播列表

详细的 API 文档请参考 [API Documentation](./docs/api.md)

## 配置说明

配置文件位于 `configs/config.yaml`，也可以通过环境变量覆盖：

```yaml
server:
  host: "0.0.0.0"
  port: "8080"
  mode: "debug"  # debug 或 release

database:
  host: "localhost"
  port: "5432"
  user: "postgres"
  password: "postgres"
  dbname: "microvibe"
  sslmode: "disable"

redis:
  host: "localhost"
  port: "6379"
  password: ""
  db: 0

jwt:
  secret: "your-secret-key"
  expire: 86400  # 24小时

upload:
  max_size: 104857600  # 100MB
  allowed_types: ["video/mp4", "video/avi"]
```

## 开发指南

### 编译项目
```bash
make build
```

### 运行测试
```bash
make test
```

### 构建 Docker 镜像
```bash
make docker-build
```

### 代码规范
- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 所有公共函数必须有中文注释
- 提交前运行 `go vet` 和 `golint`

## 性能优化

1. **数据库优化**
   - 合理使用索引
   - 查询优化和慢查询监控
   - 读写分离（主从复制）

2. **缓存策略**
   - Redis 缓存热门数据
   - 视频信息缓存
   - 用户会话缓存

3. **CDN 加速**
   - 视频文件 CDN 分发
   - 静态资源加速

## 部署方案

### 单机部署
适合小规模应用，使用 Docker Compose 一键部署。

### 集群部署
- 应用层：多实例部署 + 负载均衡
- 数据库：主从复制 + 读写分离
- Redis：主从复制 + 哨兵模式
- 对象存储：分布式存储集群

## 监控和日志

- **日志**：结构化日志，支持多级别
- **监控**：Prometheus + Grafana
- **追踪**：分布式链路追踪
- **告警**：关键指标告警

## 安全性

- JWT Token 认证
- 密码加密存储（bcrypt）
- SQL 注入防护（GORM 参数化查询）
- XSS 防护
- CSRF 防护
- 限流和防刷

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 联系方式

如有问题，请提交 Issue 或联系开发团队。
>> EOF
