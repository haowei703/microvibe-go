# SFU 快速启动指南

## 问题诊断

### 错误 1: URL 格式错误
```
error: "rpc call ping() on http://http//ion-sfu-1:7000"
```
**原因**: 代码重复添加了 `http://` 前缀
**状态**: ✅ 已修复

### 错误 2: 端口冲突 (macOS)
```
error: "status code: 403. could not decode body to rpc response: EOF"
```
**原因**: macOS 系统服务占用了端口 7000 和 7001
- 端口 7000: Apple AirTunes 服务
- 端口 7001: macOS ControlCenter 或 Docker

**解决方案**: 使用其他端口或禁用 SFU

## 本地开发配置

### 方案 1: 禁用 SFU (推荐用于本地开发)

如果你还没有部署 SFU 服务器，建议先禁用 SFU，系统会自动回退到 P2P 模式：

```yaml
# configs/config.yaml
sfu:
  enabled: false  # 禁用 SFU，使用 P2P 模式
```

### 方案 2: 更改 SFU 端口避免冲突

如果你想使用 SFU，但端口 7001 被占用：

```yaml
# configs/config.yaml
sfu:
  enabled: true
  server_url: "http://localhost:7777"  # 使用未被占用的端口
```

然后在 Docker 部署时映射端口：
```yaml
# docker-compose.yaml
ion-sfu:
  ports:
    - "7777:7000"  # 将容器内部 7000 映射到宿主机 7777
```

## 推荐的端口配置

| 端口 | 占用情况 (macOS) | 建议 |
|------|-----------------|------|
| 7000 | Apple AirTunes | ❌ 避免使用 |
| 7001 | ControlCenter/Docker | ❌ 避免使用 |
| 7777 | - | ✅ 推荐使用 |
| 8000 | - | ✅ 可用 |

## 验证配置

启动应用后检查日志：

**SFU 启用且连接成功**:
```
INFO  初始化 SFU 客户端服务  mode=standalone
INFO  SFU 客户端服务初始化成功  address=http://localhost:7777
```

**SFU 启用但连接失败** (端口冲突):
```
WARN  SFU 服务器连接失败，将在运行时重试  error="status code: 403"
```

**SFU 禁用** (P2P 模式):
```
INFO  信令服务使用 P2P 模式
```
