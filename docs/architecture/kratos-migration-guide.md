# Kratos 微服务架构改造指南

## 项目概述

将 MicroVibe-Go 单体应用改造为基于 Kratos 的微服务架构，支持 Kubernetes 部署。

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│              Ingress (Nginx/Traefik)                         │
│              域名路由、TLS、负载均衡                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              API Gateway (APISIX)                            │
│          统一入口、鉴权、限流、熔断、路由转发                   │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ User Service  │    │ Video Service │    │ Live Service  │
│   (Pod x3)    │    │   (Pod x5)    │    │   (Pod x3)    │
│   gRPC:9000   │    │   gRPC:9000   │    │   gRPC:9000   │
│   HTTP:8000   │    │   HTTP:8000   │    │   HTTP:8000   │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
                 ┌─────────────────────────┐
                 │  Service Mesh (Istio)   │
                 │  服务发现、流量管理       │
                 └─────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ PostgreSQL    │    │  Redis Cluster│    │  Kafka Cluster│
│  StatefulSet  │    │  StatefulSet  │    │  StatefulSet  │
└───────────────┘    └───────────────┘    └───────────────┘
```

## 改造计划

### Phase 1: 基础设施搭建（Week 1-2）

#### 1.1 安装 Kratos CLI

```bash
# 安装 Kratos CLI 工具
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest

# 验证安装
kratos --version
# Output: kratos version v2.x.x
```

#### 1.2 创建微服务项目结构

```bash
# 创建微服务根目录
mkdir -p microvibe-kratos
cd microvibe-kratos

# 创建各个微服务
kratos new user-service
kratos new video-service
kratos new live-service
kratos new recommend-service
kratos new search-service
kratos new message-service
kratos new hashtag-service
kratos new media-service

# 项目结构
microvibe-kratos/
├── user-service/          # 用户服务
├── video-service/         # 视频服务
├── live-service/          # 直播服务
├── recommend-service/     # 推荐服务
├── search-service/        # 搜索服务
├── message-service/       # 消息服务
├── hashtag-service/       # 话题服务
├── media-service/         # 媒体服务
├── api/                   # 公共 Proto 定义
├── deploy/                # K8s 部署文件
│   ├── base/             # 基础配置
│   └── overlays/         # 环境配置
│       ├── dev/
│       ├── staging/
│       └── prod/
└── scripts/              # 脚本工具
```

#### 1.3 搭建 K8s 开发环境

**方案 A: 本地 Minikube**
```bash
# 安装 Minikube
brew install minikube

# 启动集群
minikube start --cpus=4 --memory=8192 --driver=docker

# 启用 Ingress
minikube addons enable ingress

# 启用 Metrics Server
minikube addons enable metrics-server
```

**方案 B: 云服务 (推荐生产)**
```bash
# 阿里云 ACK / 腾讯云 TKE / AWS EKS
# 通过控制台创建托管 K8s 集群
```

#### 1.4 部署基础设施

```bash
# 部署 PostgreSQL Operator
kubectl apply -f https://operatorhub.io/install/postgres-operator.yaml

# 部署 Redis Operator
kubectl apply -f https://operatorhub.io/install/redis-operator.yaml

# 部署 Kafka (Strimzi Operator)
kubectl create namespace kafka
kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# 部署 APISIX
helm repo add apisix https://charts.apiseven.com
helm install apisix apisix/apisix --namespace apisix --create-namespace

# 部署 Jaeger (链路追踪)
kubectl create namespace observability
kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/latest/download/jaeger-operator.yaml -n observability

# 部署 Prometheus + Grafana (监控)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack -n observability
```

---

### Phase 2: User Service 改造（Week 2-3）

#### 2.1 创建 User Service 项目

```bash
cd microvibe-kratos
kratos new user-service
cd user-service
```

**项目结构**:
```
user-service/
├── api/                    # API Proto 定义
│   └── user/
│       └── v1/
│           ├── user.proto         # gRPC 接口定义
│           ├── user.pb.go         # 生成的代码
│           ├── user_grpc.pb.go
│           └── user_http.pb.go
├── cmd/
│   └── server/
│       ├── main.go               # 启动入口
│       ├── wire.go               # 依赖注入
│       └── wire_gen.go           # 生成的依赖注入代码
├── internal/
│   ├── biz/                      # 业务逻辑层
│   │   ├── user.go              # 用户业务逻辑
│   │   └── biz.go               # Biz Provider
│   ├── data/                     # 数据访问层
│   │   ├── user.go              # 用户数据访问
│   │   ├── data.go              # Data Provider
│   │   └── ent/                 # Ent ORM (可选)
│   ├── service/                  # 服务实现层
│   │   ├── user.go              # 实现 gRPC/HTTP 接口
│   │   └── service.go           # Service Provider
│   ├── conf/                     # 配置定义
│   │   ├── conf.proto           # 配置 Proto
│   │   └── conf.pb.go
│   └── server/                   # 服务器配置
│       ├── grpc.go              # gRPC Server
│       ├── http.go              # HTTP Server
│       └── server.go            # Server Provider
├── configs/
│   └── config.yaml              # 配置文件
├── deploy/
│   └── k8s/
│       ├── deployment.yaml      # K8s Deployment
│       ├── service.yaml         # K8s Service
│       ├── configmap.yaml       # ConfigMap
│       └── hpa.yaml             # 自动伸缩
├── Dockerfile                    # 多阶段构建
├── Makefile                      # 常用命令
└── go.mod
```

#### 2.2 定义 Proto 接口

**api/user/v1/user.proto**:
```protobuf
syntax = "proto3";

package api.user.v1;

option go_package = "user-service/api/user/v1;v1";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";

// UserService 用户服务
service UserService {
  // 用户注册
  rpc Register (RegisterRequest) returns (RegisterReply) {
    option (google.api.http) = {
      post: "/api/v1/users/register"
      body: "*"
    };
  }

  // 用户登录
  rpc Login (LoginRequest) returns (LoginReply) {
    option (google.api.http) = {
      post: "/api/v1/users/login"
      body: "*"
    };
  }

  // 获取用户信息
  rpc GetUser (GetUserRequest) returns (GetUserReply) {
    option (google.api.http) = {
      get: "/api/v1/users/{user_id}"
    };
  }

  // 更新用户信息
  rpc UpdateUser (UpdateUserRequest) returns (UpdateUserReply) {
    option (google.api.http) = {
      put: "/api/v1/users/{user_id}"
      body: "*"
    };
  }

  // 关注用户
  rpc FollowUser (FollowUserRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/api/v1/users/{target_id}/follow"
    };
  }

  // 取消关注
  rpc UnfollowUser (UnfollowUserRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/users/{target_id}/follow"
    };
  }

  // 获取关注列表
  rpc GetFollowing (GetFollowingRequest) returns (GetFollowingReply) {
    option (google.api.http) = {
      get: "/api/v1/users/{user_id}/following"
    };
  }

  // 获取粉丝列表
  rpc GetFollowers (GetFollowersRequest) returns (GetFollowersReply) {
    option (google.api.http) = {
      get: "/api/v1/users/{user_id}/followers"
    };
  }
}

// 注册请求
message RegisterRequest {
  string username = 1;
  string password = 2;
  string email = 3;
  string phone = 4;
  string nickname = 5;
}

message RegisterReply {
  uint32 user_id = 1;
  string token = 2;
  UserInfo user = 3;
}

// 登录请求
message LoginRequest {
  string username = 1;  // 用户名或邮箱
  string password = 2;
}

message LoginReply {
  uint32 user_id = 1;
  string token = 2;
  UserInfo user = 3;
}

// 获取用户信息
message GetUserRequest {
  uint32 user_id = 1;
}

message GetUserReply {
  UserInfo user = 1;
}

// 更新用户信息
message UpdateUserRequest {
  uint32 user_id = 1;
  string nickname = 2;
  string avatar = 3;
  string bio = 4;
  int32 gender = 5;
  string birthday = 6;
}

message UpdateUserReply {
  UserInfo user = 1;
}

// 关注用户
message FollowUserRequest {
  uint32 user_id = 1;      // 当前用户 ID (从 JWT 获取)
  uint32 target_id = 2;    // 目标用户 ID
}

// 取消关注
message UnfollowUserRequest {
  uint32 user_id = 1;
  uint32 target_id = 2;
}

// 获取关注列表
message GetFollowingRequest {
  uint32 user_id = 1;
  int32 page = 2;
  int32 page_size = 3;
}

message GetFollowingReply {
  repeated UserInfo users = 1;
  int64 total = 2;
}

// 获取粉丝列表
message GetFollowersRequest {
  uint32 user_id = 1;
  int32 page = 2;
  int32 page_size = 3;
}

message GetFollowersReply {
  repeated UserInfo users = 1;
  int64 total = 2;
}

// 用户信息
message UserInfo {
  uint32 id = 1;
  string username = 2;
  string nickname = 3;
  string email = 4;
  string phone = 5;
  string avatar = 6;
  string bio = 7;
  int32 gender = 8;
  string birthday = 9;
  int64 follower_count = 10;
  int64 following_count = 11;
  int64 video_count = 12;
  int64 like_count = 13;
  string created_at = 14;
  string updated_at = 15;
}
```

#### 2.3 生成代码

```bash
# 安装 protoc 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest

# 生成代码
make api

# 或手动执行
protoc --proto_path=. \
       --proto_path=./third_party \
       --go_out=paths=source_relative:. \
       --go-grpc_out=paths=source_relative:. \
       --go-http_out=paths=source_relative:. \
       api/user/v1/*.proto
```

#### 2.4 实现业务逻辑

**internal/biz/user.go**:
```go
package biz

import (
    "context"
    "errors"
    "time"

    "github.com/go-kratos/kratos/v2/log"
    "golang.org/x/crypto/bcrypt"
)

// User 用户实体
type User struct {
    ID             uint32
    Username       string
    Password       string
    Email          string
    Phone          string
    Nickname       string
    Avatar         string
    Bio            string
    Gender         int32
    Birthday       *time.Time
    FollowerCount  int64
    FollowingCount int64
    VideoCount     int64
    LikeCount      int64
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// UserRepo 用户仓储接口
type UserRepo interface {
    CreateUser(ctx context.Context, user *User) (*User, error)
    GetUser(ctx context.Context, id uint32) (*User, error)
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    UpdateUser(ctx context.Context, user *User) (*User, error)
    DeleteUser(ctx context.Context, id uint32) error

    // 关注相关
    Follow(ctx context.Context, userID, targetID uint32) error
    Unfollow(ctx context.Context, userID, targetID uint32) error
    IsFollowing(ctx context.Context, userID, targetID uint32) (bool, error)
    GetFollowing(ctx context.Context, userID uint32, page, pageSize int32) ([]*User, int64, error)
    GetFollowers(ctx context.Context, userID uint32, page, pageSize int32) ([]*User, int64, error)
}

// UserUsecase 用户用例
type UserUsecase struct {
    repo UserRepo
    log  *log.Helper
}

// NewUserUsecase 创建用户用例
func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
    return &UserUsecase{
        repo: repo,
        log:  log.NewHelper(logger),
    }
}

// Register 用户注册
func (uc *UserUsecase) Register(ctx context.Context, username, password, email, phone, nickname string) (*User, string, error) {
    uc.log.WithContext(ctx).Infof("Register user: %s", username)

    // 检查用户是否已存在
    existUser, _ := uc.repo.GetUserByUsername(ctx, username)
    if existUser != nil {
        return nil, "", errors.New("username already exists")
    }

    if email != "" {
        existUser, _ = uc.repo.GetUserByEmail(ctx, email)
        if existUser != nil {
            return nil, "", errors.New("email already exists")
        }
    }

    // 密码加密
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, "", err
    }

    // 创建用户
    user := &User{
        Username: username,
        Password: string(hashedPassword),
        Email:    email,
        Phone:    phone,
        Nickname: nickname,
    }

    if user.Nickname == "" {
        user.Nickname = username
    }

    user, err = uc.repo.CreateUser(ctx, user)
    if err != nil {
        return nil, "", err
    }

    // 生成 JWT Token (这里简化处理，实际应该注入 JWT Service)
    token := "mock-jwt-token"

    return user, token, nil
}

// Login 用户登录
func (uc *UserUsecase) Login(ctx context.Context, username, password string) (*User, string, error) {
    uc.log.WithContext(ctx).Infof("Login user: %s", username)

    // 查找用户（支持用户名或邮箱登录）
    user, err := uc.repo.GetUserByUsername(ctx, username)
    if err != nil {
        // 尝试使用邮箱查找
        user, err = uc.repo.GetUserByEmail(ctx, username)
        if err != nil {
            return nil, "", errors.New("invalid username or password")
        }
    }

    // 验证密码
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        return nil, "", errors.New("invalid username or password")
    }

    // 生成 JWT Token
    token := "mock-jwt-token"

    return user, token, nil
}

// GetUser 获取用户信息
func (uc *UserUsecase) GetUser(ctx context.Context, id uint32) (*User, error) {
    return uc.repo.GetUser(ctx, id)
}

// UpdateUser 更新用户信息
func (uc *UserUsecase) UpdateUser(ctx context.Context, user *User) (*User, error) {
    return uc.repo.UpdateUser(ctx, user)
}

// FollowUser 关注用户
func (uc *UserUsecase) FollowUser(ctx context.Context, userID, targetID uint32) error {
    if userID == targetID {
        return errors.New("cannot follow yourself")
    }

    // 检查目标用户是否存在
    _, err := uc.repo.GetUser(ctx, targetID)
    if err != nil {
        return errors.New("target user not found")
    }

    // 检查是否已关注
    isFollowing, err := uc.repo.IsFollowing(ctx, userID, targetID)
    if err != nil {
        return err
    }
    if isFollowing {
        return errors.New("already following")
    }

    return uc.repo.Follow(ctx, userID, targetID)
}

// UnfollowUser 取消关注
func (uc *UserUsecase) UnfollowUser(ctx context.Context, userID, targetID uint32) error {
    return uc.repo.Unfollow(ctx, userID, targetID)
}

// GetFollowing 获取关注列表
func (uc *UserUsecase) GetFollowing(ctx context.Context, userID uint32, page, pageSize int32) ([]*User, int64, error) {
    return uc.repo.GetFollowing(ctx, userID, page, pageSize)
}

// GetFollowers 获取粉丝列表
func (uc *UserUsecase) GetFollowers(ctx context.Context, userID uint32, page, pageSize int32) ([]*User, int64, error) {
    return uc.repo.GetFollowers(ctx, userID, page, pageSize)
}
```

#### 2.5 实现数据访问层

**internal/data/user.go**:
```go
package data

import (
    "context"
    "user-service/internal/biz"

    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"
)

// User GORM 模型
type User struct {
    gorm.Model
    Username       string `gorm:"uniqueIndex;size:50;not null"`
    Password       string `gorm:"size:255;not null"`
    Email          string `gorm:"uniqueIndex;size:100"`
    Phone          string `gorm:"uniqueIndex;size:20"`
    Nickname       string `gorm:"size:50"`
    Avatar         string `gorm:"size:255"`
    Bio            string `gorm:"type:text"`
    Gender         int32
    Birthday       *time.Time
    FollowerCount  int64 `gorm:"default:0"`
    FollowingCount int64 `gorm:"default:0"`
    VideoCount     int64 `gorm:"default:0"`
    LikeCount      int64 `gorm:"default:0"`
}

// Follow 关注关系模型
type Follow struct {
    gorm.Model
    UserID   uint32 `gorm:"index;not null"`
    TargetID uint32 `gorm:"index;not null"`
}

type userRepo struct {
    data *Data
    log  *log.Helper
}

// NewUserRepo 创建用户仓储
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
    return &userRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

func (r *userRepo) CreateUser(ctx context.Context, user *biz.User) (*biz.User, error) {
    dbUser := &User{
        Username: user.Username,
        Password: user.Password,
        Email:    user.Email,
        Phone:    user.Phone,
        Nickname: user.Nickname,
        Avatar:   user.Avatar,
        Bio:      user.Bio,
        Gender:   user.Gender,
        Birthday: user.Birthday,
    }

    if err := r.data.db.WithContext(ctx).Create(dbUser).Error; err != nil {
        return nil, err
    }

    return r.toEntity(dbUser), nil
}

func (r *userRepo) GetUser(ctx context.Context, id uint32) (*biz.User, error) {
    var user User
    if err := r.data.db.WithContext(ctx).First(&user, id).Error; err != nil {
        return nil, err
    }
    return r.toEntity(&user), nil
}

// ... 其他方法实现

func (r *userRepo) toEntity(user *User) *biz.User {
    return &biz.User{
        ID:             uint32(user.ID),
        Username:       user.Username,
        Password:       user.Password,
        Email:          user.Email,
        Phone:          user.Phone,
        Nickname:       user.Nickname,
        Avatar:         user.Avatar,
        Bio:            user.Bio,
        Gender:         user.Gender,
        Birthday:       user.Birthday,
        FollowerCount:  user.FollowerCount,
        FollowingCount: user.FollowingCount,
        VideoCount:     user.VideoCount,
        LikeCount:      user.LikeCount,
        CreatedAt:      user.CreatedAt,
        UpdatedAt:      user.UpdatedAt,
    }
}
```

---

### Phase 3: K8s 部署配置（Week 3-4）

#### 3.1 Dockerfile (多阶段构建)

**user-service/Dockerfile**:
```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct

# 缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 编译
COPY . .
RUN go build -ldflags "-s -w" -o /app/user-service ./cmd/server

# 运行阶段
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/user-service ./
COPY configs ./configs

EXPOSE 8000 9000

CMD ["./user-service", "-conf", "configs"]
```

#### 3.2 K8s ConfigMap

**deploy/k8s/configmap.yaml**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: user-service-config
  namespace: microvibe
data:
  config.yaml: |
    server:
      http:
        addr: 0.0.0.0:8000
        timeout: 30s
      grpc:
        addr: 0.0.0.0:9000
        timeout: 30s

    data:
      database:
        driver: postgres
        source: postgres://user:password@postgres-user:5432/microvibe_user?sslmode=disable
      redis:
        addr: redis-cluster:6379
        read_timeout: 0.2s
        write_timeout: 0.2s

    trace:
      endpoint: http://jaeger-collector:14268/api/traces
```

#### 3.3 K8s Deployment

**deploy/k8s/deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: microvibe
  labels:
    app: user-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
        version: v1
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8000"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: user-service
        image: registry.example.com/microvibe/user-service:v1.0.0
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 8000
          protocol: TCP
        - name: grpc
          containerPort: 9000
          protocol: TCP
        env:
        - name: SERVICE_NAME
          value: "user-service"
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        volumeMounts:
        - name: config
          mountPath: /app/configs
          readOnly: true
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
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
      volumes:
      - name: config
        configMap:
          name: user-service-config
---
apiVersion: v1
kind: Service
metadata:
  name: user-service
  namespace: microvibe
  labels:
    app: user-service
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 8000
    targetPort: 8000
    protocol: TCP
  - name: grpc
    port: 9000
    targetPort: 9000
    protocol: TCP
  selector:
    app: user-service
```

#### 3.4 HPA (自动伸缩)

**deploy/k8s/hpa.yaml**:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: user-service-hpa
  namespace: microvibe
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
      - type: Pods
        value: 4
        periodSeconds: 15
      selectPolicy: Max
```

---

### Phase 4: 完整部署脚本

#### 4.1 Makefile

**user-service/Makefile**:
```makefile
.PHONY: init api build docker push deploy

# 初始化项目
init:
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go mod download

# 生成 API 代码
api:
	kratos proto client api/user/v1/*.proto

# 编译
build:
	go build -ldflags "-s -w" -o ./bin/user-service ./cmd/server

# 构建 Docker 镜像
docker:
	docker build -t registry.example.com/microvibe/user-service:v1.0.0 .

# 推送镜像
push:
	docker push registry.example.com/microvibe/user-service:v1.0.0

# 部署到 K8s
deploy:
	kubectl apply -f deploy/k8s/

# 查看日志
logs:
	kubectl logs -f -l app=user-service -n microvibe --tail=100

# 进入 Pod
exec:
	kubectl exec -it $(shell kubectl get pod -l app=user-service -n microvibe -o jsonpath='{.items[0].metadata.name}') -n microvibe -- /bin/sh
```

---

## 下一步行动

我已经为您准备好了完整的 Kratos 微服务改造方案。现在我将：

1. ✅ 创建第一个微服务（User Service）的完整代码
2. ✅ 包含 K8s 部署配置
3. ✅ 提供构建和部署脚本

**请问是否开始创建 User Service 的完整项目？**

我将创建：
- [ ] 完整的项目结构
- [ ] Proto 定义和生成代码
- [ ] Biz/Data/Service 层实现
- [ ] K8s 部署文件
- [ ] Dockerfile 和 Makefile

回复 "开始" 我就立即创建第一个微服务示例！
