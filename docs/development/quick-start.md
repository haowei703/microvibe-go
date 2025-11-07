# 快速开始

## 前置要求

- Go 1.23+
- PostgreSQL 16
- Redis 7
- Docker & Docker Compose（可选）

## 本地开发

### 1. 克隆项目

```bash
git clone <repository-url>
cd microvibe-go
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置数据库和 Redis

确保 PostgreSQL 和 Redis 服务已启动。

编辑配置文件 `configs/config.yaml`：

```yaml
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
```

### 4. 执行数据库迁移

```bash
make migrate
```

这将自动创建所有表并填充初始数据。

### 5. 启动应用

```bash
make run
```

应用将在 `http://localhost:8080` 启动。

## 使用 Docker Compose

### 1. 启动所有服务

```bash
docker-compose up -d
```

这将启动：
- 应用服务（端口 8080）
- PostgreSQL（端口 5432）
- Redis（端口 6379）

### 2. 查看日志

```bash
docker-compose logs -f app
```

### 3. 停止服务

```bash
docker-compose down
```

## API 测试

### 健康检查

```bash
curl http://localhost:8080/health
```

### 用户注册

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test123",
    "password": "123456",
    "email": "test@example.com",
    "nickname": "测试用户"
  }'
```

### 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test123",
    "password": "123456"
  }'
```

### 获取推荐视频流

```bash
# 游客访问
curl http://localhost:8080/api/v1/videos/feed

# 登录用户访问（需要 Token）
curl http://localhost:8080/api/v1/videos/feed \
  -H "Authorization: Bearer <your-token>"
```

### 获取用户信息

```bash
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <your-token>"
```

### 关注用户

```bash
curl -X POST http://localhost:8080/api/v1/users/1/follow \
  -H "Authorization: Bearer <your-token>"
```

## 项目结构

```
microvibe-go/
├── cmd/
│   ├── server/          # 主程序入口
│   └── migrate/         # 数据库迁移工具
├── internal/
│   ├── algorithm/       # 推荐算法引擎
│   │   ├── recommend/   # 召回策略
│   │   ├── feature/     # 特征工程
│   │   ├── rank/        # 排序算法
│   │   └── filter/      # 过滤策略
│   ├── config/          # 配置管理
│   ├── database/        # 数据库连接
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件
│   ├── model/           # 数据模型
│   ├── router/          # 路由
│   └── service/         # 业务逻辑
├── pkg/
│   ├── response/        # 统一响应格式
│   └── utils/           # 工具函数
└── configs/             # 配置文件
```

## 开发指南

### 添加新的 API 接口

1. 在 `internal/model` 中定义数据模型
2. 在 `internal/service` 中实现业务逻辑
3. 在 `internal/handler` 中创建处理器
4. 在 `internal/router` 中注册路由

### 数据库操作

使用 GORM 进行数据库操作：

```go
// 创建
db.Create(&user)

// 查询
db.First(&user, id)
db.Where("username = ?", username).First(&user)

// 更新
db.Model(&user).Updates(map[string]interface{}{"nickname": "新昵称"})

// 删除（软删除）
db.Delete(&user, id)
```

### 推荐算法

推荐算法核心流程：

1. **召回**：从海量视频中快速召回候选集
   - 协同过滤召回
   - 内容召回
   - 热门召回
   - 关注召回

2. **特征工程**：提取用户和视频特征
   - 用户特征：观看历史、兴趣标签
   - 视频特征：分类、热度、质量分

3. **排序**：精准排序
   - CTR 预估
   - 完播率预估
   - 互动率预估

4. **过滤**：内容过滤
   - 去重
   - 质量过滤
   - 黑名单过滤

## 常见问题

### 1. 数据库连接失败

检查 PostgreSQL 是否启动，配置是否正确。

### 2. Redis 连接失败

检查 Redis 是否启动，配置是否正确。

### 3. 端口被占用

修改 `configs/config.yaml` 中的端口配置。

## 下一步

- 查看 [API 文档](./api.md)
- 了解 [推荐算法](./algorithm.md)
- 阅读 [部署指南](./deploy.md)
