# SFU 混合推流部署指南

## 概述

本文档介绍如何部署支持 WebRTC + RTMP 双协议的混合推流系统。系统架构包含:

- **应用服务器** - MicroVibe Go 后端 (Gin + GORM)
- **SFU 服务器** - Pion Ion SFU (WebRTC 流分发)
- **RTMP 服务器** - Nginx-RTMP (RTMP 推流和 HLS/FLV 播放)
- **数据库** - PostgreSQL 16
- **缓存** - Redis 7

## 架构特性

### 推流方式

1. **WebRTC 推流** (浏览器)
   - 超低延迟 (< 1秒)
   - 无需额外软件
   - 适合移动端、临时直播

2. **RTMP 推流** (OBS/专业软件)
   - 稳定性好
   - 专业功能 (美颜、特效、混流)
   - 适合专业直播、长时间直播

### 播放方式

1. **WebRTC 播放** - 超低延迟 (< 500ms)
2. **HLS 播放** - 兼容性好 (10-30秒延迟)
3. **FLV 播放** - 低延迟 (2-5秒)
4. **RTMP 播放** - 传统播放器支持

## Docker Compose 部署

### 1. 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  # PostgreSQL 数据库
  postgres:
    image: postgres:16
    container_name: microvibe-postgres
    environment:
      POSTGRES_DB: microvibe
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - microvibe-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: microvibe-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - microvibe-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Pion Ion SFU 服务器
  ion-sfu:
    image: pion/ion-sfu:latest
    container_name: microvibe-sfu
    ports:
      - "7777:7000"      # JSON-RPC API (映射到 7777 避免与 macOS 服务冲突)
      - "5000-5100:5000-5100/udp"  # WebRTC UDP 端口范围
    volumes:
      - ./configs/sfu.toml:/configs/sfu.toml
    command: -c /configs/sfu.toml
    networks:
      - microvibe-network
    restart: unless-stopped

  # Nginx-RTMP 服务器
  nginx-rtmp:
    image: tiangolo/nginx-rtmp
    container_name: microvibe-rtmp
    ports:
      - "1935:1935"      # RTMP 推流端口
      - "8081:80"        # HLS/FLV HTTP 端口
    volumes:
      - ./configs/nginx-rtmp.conf:/etc/nginx/nginx.conf
      - ./data/hls:/tmp/hls
      - ./data/recordings:/tmp/recordings
    networks:
      - microvibe-network
    restart: unless-stopped

  # MicroVibe 应用服务
  app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: microvibe-app
    ports:
      - "8080:8080"
    environment:
      # 数据库配置
      DATABASE_HOST: postgres
      DATABASE_PORT: 5432
      DATABASE_USER: postgres
      DATABASE_PASSWORD: postgres
      DATABASE_DBNAME: microvibe
      DATABASE_SSLMODE: disable

      # Redis 配置
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: ""
      REDIS_DB: 0

      # 服务器配置
      SERVER_HOST: 0.0.0.0
      SERVER_PORT: 8080
      SERVER_MODE: release

      # JWT 配置
      JWT_SECRET: "your-production-secret-key"
      JWT_EXPIRE: 24

      # SFU 配置
      SFU_ENABLED: true
      SFU_SERVER_URL: "http://ion-sfu:7000"  # Docker 内部使用容器名和内部端口
      SFU_MODE: standalone

      # RTMP 流媒体配置
      STREAMING_RTMP_SERVER: "rtmp://nginx-rtmp:1935/live"
      STREAMING_HLS_SERVER: "http://localhost:8081/hls"
      STREAMING_FLV_SERVER: "http://localhost:8081/flv"
      STREAMING_RTMP_PLAY_SERVER: "rtmp://nginx-rtmp:1935/live"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      ion-sfu:
        condition: service_started
      nginx-rtmp:
        condition: service_started
    networks:
      - microvibe-network
    restart: unless-stopped

networks:
  microvibe-network:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
```

### 2. 创建 SFU 配置文件 (configs/sfu.toml)

```toml
[sfu]
# Pion Ion SFU 配置
ballast = 20971520  # 20MB

[router]
# WebRTC 端口范围
minport = 5000
maxport = 5100

[router.rtp]
# RTP 配置
port = 5004

[turn]
# TURN 服务器配置（可选，用于 NAT 穿透）
enabled = false
# realm = "microvibe.com"
# address = "0.0.0.0:3478"
# public_ip = "your-public-ip"
# port_range = [49152, 65535]

[webrtc]
# ICE 服务器配置
iceserver = [
    "stun:stun.l.google.com:19302",
    "stun:stun1.l.google.com:19302"
]

# SDP 语义化
sdpsemantics = "unified-plan"

# 端口分配策略
# mdns = true
# ice_lite = false
# candidates = ["udp", "tcp"]

[log]
level = "info"
```

### 3. 创建 Nginx-RTMP 配置文件 (configs/nginx-rtmp.conf)

```nginx
worker_processes auto;
rtmp_auto_push on;
events {}

rtmp {
    server {
        listen 1935;
        listen [::]:1935 ipv6only=on;

        application live {
            live on;
            record off;

            # HLS 配置
            hls on;
            hls_path /tmp/hls;
            hls_fragment 3;
            hls_playlist_length 60;

            # DASH 配置 (可选)
            # dash on;
            # dash_path /tmp/dash;

            # 录制配置 (可选)
            # record all;
            # record_path /tmp/recordings;
            # record_unique on;

            # 允许发布和播放
            allow publish all;
            allow play all;

            # 推流认证回调 (可选)
            # on_publish http://microvibe-app:8080/api/v1/live/auth/publish;
            # on_publish_done http://microvibe-app:8080/api/v1/live/auth/publish_done;
        }
    }
}

http {
    sendfile off;
    tcp_nopush on;
    directio 512;
    default_type application/octet-stream;

    server {
        listen 80;

        # HLS 播放
        location /hls {
            types {
                application/vnd.apple.mpegurl m3u8;
                video/mp2t ts;
            }
            root /tmp;
            add_header Cache-Control no-cache;
            add_header Access-Control-Allow-Origin *;
        }

        # FLV 播放
        location /flv {
            flv_live on;
            chunked_transfer_encoding on;
            add_header Access-Control-Allow-Origin *;
            add_header Cache-Control no-cache;

            types {
                application/x-mpegURL m3u8;
                video/mp2t ts;
            }

            root /tmp;
        }

        # RTMP 状态页面
        location /stat {
            rtmp_stat all;
            rtmp_stat_stylesheet stat.xsl;
            add_header Access-Control-Allow-Origin *;
        }

        location /stat.xsl {
            root /usr/local/nginx/html;
        }

        # 控制接口
        location /control {
            rtmp_control all;
            add_header Access-Control-Allow-Origin *;
        }
    }
}
```

### 4. Dockerfile

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server/main.go

# 运行阶段
FROM alpine:latest

WORKDIR /root/

# 安装 ca-certificates (HTTPS 请求需要)
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]
```

## 部署步骤

### 1. 准备配置文件

```bash
# 创建配置目录
mkdir -p configs data/hls data/recordings

# 创建 SFU 配置文件
cat > configs/sfu.toml << 'EOF'
# (见上面的 sfu.toml 配置)
EOF

# 创建 Nginx-RTMP 配置文件
cat > configs/nginx-rtmp.conf << 'EOF'
# (见上面的 nginx-rtmp.conf 配置)
EOF
```

### 2. 启动服务

```bash
# 构建并启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看应用日志
docker-compose logs -f app

# 查看 SFU 日志
docker-compose logs -f ion-sfu

# 查看 RTMP 日志
docker-compose logs -f nginx-rtmp
```

### 3. 初始化数据库

```bash
# 进入应用容器
docker-compose exec app sh

# 运行数据库迁移
./main migrate

# 或者从宿主机运行迁移工具
docker-compose exec app ./main migrate
```

### 4. 验证部署

#### 检查服务健康状态

```bash
# 检查应用健康
curl http://localhost:8080/health

# 检查 RTMP 状态
curl http://localhost:8081/stat

# 预期输出:
# {
#   "code": 0,
#   "message": "success",
#   "data": {
#     "database": "healthy",
#     "redis": "healthy"
#   }
# }
```

#### 测试 WebRTC 推流

1. 创建直播间:
```bash
curl -X POST http://localhost:8080/api/v1/live/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "测试直播",
    "push_protocol": "webrtc"
  }'
```

2. 使用浏览器连接 WebSocket:
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/live/ws?room_id=xxx&user_id=1&role=publisher');
// (见 HYBRID_STREAMING_ARCHITECTURE.md 中的完整示例)
```

#### 测试 RTMP 推流

1. 创建直播间:
```bash
curl -X POST http://localhost:8080/api/v1/live/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "测试直播",
    "push_protocol": "rtmp"
  }'
```

2. 使用 OBS 推流:
   - 服务器: `rtmp://localhost:1935/live`
   - 串流密钥: `{返回的 stream_key}`

3. 播放测试:
   - HLS: `http://localhost:8081/hls/{stream_key}.m3u8`
   - FLV: `http://localhost:8081/flv/{stream_key}.flv`
   - RTMP: `rtmp://localhost:1935/live/{stream_key}`

## 生产环境配置

### 1. 安全加固

#### 更新 JWT 密钥

```yaml
environment:
  JWT_SECRET: "使用强随机密钥"
```

#### 配置 HTTPS

使用 Nginx 反向代理:

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket 代理
    location /api/v1/live/ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

#### 启用 TURN 服务器

对于 NAT 穿透，配置 TURN 服务器:

```toml
# sfu.toml
[turn]
enabled = true
realm = "yourdomain.com"
address = "0.0.0.0:3478"
public_ip = "your-public-ip"
```

### 2. 性能优化

#### 数据库连接池

```yaml
environment:
  DB_MAX_OPEN_CONNS: 100
  DB_MAX_IDLE_CONNS: 10
  DB_CONN_MAX_LIFETIME: 3600
```

#### Redis 持久化

```yaml
redis:
  command: redis-server --appendonly yes
  volumes:
    - redis_data:/data
```

#### SFU 集群部署

```yaml
services:
  ion-sfu-1:
    image: pion/ion-sfu:latest
    ports:
      - "7001:7000"
      - "5000-5050:5000-5050/udp"

  ion-sfu-2:
    image: pion/ion-sfu:latest
    ports:
      - "7002:7000"
      - "5051-5100:5051-5100/udp"

  app:
    environment:
      SFU_MODE: cluster
      SFU_CLUSTER_NODES: "ion-sfu-1:7000,ion-sfu-2:7000"
      SFU_LOAD_BALANCE_METHOD: roundrobin
```

### 3. 监控和日志

#### 配置日志收集

```yaml
services:
  app:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### Prometheus 监控 (可选)

```yaml
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
```

## 故障排查

### 端口冲突问题

**症状**: SFU 启动失败或返回 403 Forbidden

**原因**: macOS 上端口 7000、7001 可能被系统服务占用：
- 端口 7000: macOS AirTunes 服务
- 端口 7001: macOS ControlCenter 或 Docker

**解决方案**:

1. **检查端口占用情况**:
```bash
lsof -i :7000 -i :7001
```

2. **修改端口映射** (推荐):
在 `docker-compose.yml` 中修改端口映射：
```yaml
ion-sfu:
  ports:
    - "7777:7000"  # 将容器内部 7000 映射到宿主机 7777
```

3. **更新配置文件**:
```yaml
# configs/config.yaml
sfu:
  server_url: "http://localhost:7777"  # 使用映射后的端口
```

4. **验证端口可用**:
```bash
curl -X POST http://localhost:7777 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"ping","params":{},"id":1}'
```

### SFU 连接失败

**症状**: WebRTC 无法建立连接

**检查步骤**:
1. 验证 SFU 服务是否运行: `docker-compose ps ion-sfu`
2. 检查 SFU 日志: `docker-compose logs ion-sfu`
3. 确认 UDP 端口范围开放: `5000-5100/udp`
4. 测试 SFU 健康检查:
```bash
curl -X POST http://localhost:7777 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"ping","params":{},"id":1}'
```

### RTMP 推流失败

**症状**: OBS 无法连接到 RTMP 服务器

**检查步骤**:
1. 验证 Nginx-RTMP 运行: `docker-compose ps nginx-rtmp`
2. 检查 RTMP 状态: `curl http://localhost:8081/stat`
3. 确认端口 1935 开放: `netstat -tlnp | grep 1935`
4. 查看 Nginx 日志: `docker-compose logs nginx-rtmp`

### 数据库连接失败

**症状**: 应用无法连接到 PostgreSQL

**检查步骤**:
1. 验证 PostgreSQL 运行: `docker-compose ps postgres`
2. 测试连接:
```bash
docker-compose exec postgres psql -U postgres -d microvibe -c "SELECT 1;"
```
3. 检查网络连接: `docker-compose exec app ping postgres`

### Redis 连接失败

**症状**: 缓存功能不可用

**检查步骤**:
1. 验证 Redis 运行: `docker-compose ps redis`
2. 测试连接:
```bash
docker-compose exec redis redis-cli ping
```

## 扩容和高可用

### 水平扩展应用服务

```yaml
services:
  app:
    deploy:
      replicas: 3

  nginx-lb:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx-lb.conf:/etc/nginx/nginx.conf
    depends_on:
      - app
```

### SFU 集群负载均衡

应用服务自动使用轮询算法在多个 SFU 节点间分配会话。

### 数据库主从复制

```yaml
services:
  postgres-master:
    # 主数据库配置

  postgres-replica:
    # 从数据库配置
    environment:
      POSTGRES_PRIMARY_HOST: postgres-master
```

## 备份策略

### 数据库备份

```bash
# 每日备份脚本
docker-compose exec postgres pg_dump -U postgres microvibe > backup_$(date +%Y%m%d).sql

# 恢复备份
docker-compose exec -T postgres psql -U postgres microvibe < backup_20250103.sql
```

### 录制文件备份

```bash
# 备份录制文件到云存储
aws s3 sync ./data/recordings s3://your-bucket/recordings/
```

## 参考资料

- [Pion Ion SFU 文档](https://github.com/pion/ion-sfu)
- [Nginx-RTMP 模块文档](https://github.com/arut/nginx-rtmp-module)
- [WebRTC 标准](https://www.w3.org/TR/webrtc/)
- [混合推流架构设计](./HYBRID_STREAMING_ARCHITECTURE.md)
