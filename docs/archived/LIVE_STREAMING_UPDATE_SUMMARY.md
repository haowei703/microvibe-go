# 直播功能升级总结 - OBS 推流与流类型支持

## 更新日期
2025-11-03

## 更新概述

本次更新为 MicroVibe-Go 直播功能添加了**OBS 推流支持**和**灵活的流类型配置**,使系统能够支持:

1. ✅ **OBS Studio** 等专业推流工具 (RTMP 协议)
2. ✅ **纯音频直播** (电台、播客场景)
3. ✅ **纯视频直播** (风景监控、无声视频场景)
4. ✅ **音视频直播** (标准直播场景)
5. ✅ **多种编解码器** (H.264, H.265, VP8, VP9, AAC, Opus, MP3)
6. ✅ **灵活的码率、帧率、分辨率配置**

---

## 架构变更

### 之前的架构

```
┌─────────┐                    ┌──────────┐
│ 浏览器  │ ─── WebRTC ────→   │   SFU    │
│ 推流端  │                    │  服务器  │
└─────────┘                    └──────────┘
                                     │
                               ┌─────▼─────┐
                               │   观众端   │
                               └───────────┘
```

**限制:**
- 仅支持浏览器内 WebRTC 推流
- 无法使用专业推流软件 (OBS, vMix 等)
- 流类型固定为音视频

### 更新后的架构

```
┌──────────────┐              ┌──────────────┐
│  OBS Studio  │              │   浏览器     │
│  vMix, XSplit│  ── RTMP ──→ │  WebRTC 推流 │
└──────────────┘              └──────┬───────┘
        │                            │
        ▼                            ▼
┌────────────────────────────────────────────┐
│        流媒体服务器 (Nginx-RTMP/SRS)        │
│  - RTMP 推流接收                           │
│  - 转码处理 (可选)                         │
│  - HLS/FLV/RTMP 播放输出                   │
└────────────────┬───────────────────────────┘
                 │
        ┌────────┼────────┐
        ▼        ▼        ▼
    ┌─────┐  ┌─────┐  ┌────────┐
    │ HLS │  │ FLV │  │WebRTC  │
    │播放 │  │播放 │  │播放    │
    └─────┘  └─────┘  └────────┘
```

**优势:**
- 支持专业推流软件 (OBS, vMix, XSplit)
- 同时支持 RTMP 和 WebRTC 两种推流方式
- 多种播放协议 (HLS, FLV, RTMP, WebRTC)
- 灵活的流类型配置 (音频/视频/音视频)

---

## 数据模型变更

### 新增字段 (LiveStream 模型)

#### 1. 推流配置字段

| 字段 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `RtmpURL` | string | - | RTMP 播放地址 |
| `WebRTCURL` | string | - | WebRTC 播放地址 (格式: `webrtc://room/{room_id}`) |
| `PushProtocol` | string | `rtmp` | 推流协议: `rtmp`, `webrtc`, `srt` |

#### 2. 流类型配置字段 (新增)

| 字段 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `StreamType` | string | `video_audio` | 流类型: `video_only`, `audio_only`, `video_audio` |
| `HasVideo` | bool | `true` | 是否包含视频流 |
| `HasAudio` | bool | `true` | 是否包含音频流 |
| `VideoCodec` | string | `h264` | 视频编码: `h264`, `h265`, `vp8`, `vp9`, `av1` |
| `AudioCodec` | string | `aac` | 音频编码: `aac`, `opus`, `mp3` |
| `VideoBitrate` | int | `2500` | 视频码率 (kbps) |
| `AudioBitrate` | int | `128` | 音频码率 (kbps) |
| `FrameRate` | int | `30` | 帧率: `15`, `24`, `30`, `60` |
| `Resolution` | string | `720p` | 分辨率: `360p`, `480p`, `720p`, `1080p`, `2k`, `4k` |

### 数据库迁移

**文件:** `internal/database/migrations/20251103_add_livestream_obs_support.go`

**功能:**
- 自动添加新字段到 `live_streams` 表
- 创建索引: `push_protocol`, `stream_type`, `resolution`
- 为现有记录设置默认值
- 提供回滚函数 (如需要)

**运行迁移:**

```bash
make migrate
```

或手动运行:

```bash
go run cmd/migrate/main.go
```

---

## API 变更

### 1. 创建直播间 API

**接口:** `POST /api/v1/live/create`

**新增请求参数:**

```json
{
  "title": "我的直播",
  "description": "描述",
  "cover": "封面URL",

  // ========== 新增字段 ==========
  "push_protocol": "rtmp",         // 推流协议 (可选,默认: rtmp)
  "stream_type": "video_audio",    // 流类型 (可选,默认: video_audio)
  "video_codec": "h264",           // 视频编码 (可选,默认: h264)
  "audio_codec": "aac",            // 音频编码 (可选,默认: aac)
  "video_bitrate": 2500,           // 视频码率 (可选,默认: 2500)
  "audio_bitrate": 128,            // 音频码率 (可选,默认: 128)
  "frame_rate": 30,                // 帧率 (可选,默认: 30)
  "resolution": "720p"             // 分辨率 (可选,默认: 720p)
}
```

**响应新增字段:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "title": "我的直播",

    // ========== 推流地址 (新增/更新) ==========
    "stream_url": "rtmp://localhost:1935/live/a1b2c3d4e5f6g7h8",  // RTMP 推流地址 (OBS 使用)
    "play_url": "http://localhost:8080/hls/a1b2c3d4e5f6g7h8",    // HLS 播放地址
    "flv_url": "http://localhost:8080/flv/a1b2c3d4e5f6g7h8",     // FLV 播放地址
    "rtmp_url": "rtmp://localhost:1935/live/a1b2c3d4e5f6g7h8",   // RTMP 播放地址 (新增)
    "webrtc_url": "webrtc://room/room_1234567890",               // WebRTC 播放地址 (新增)

    // ========== 流配置 (新增) ==========
    "push_protocol": "rtmp",
    "stream_type": "video_audio",
    "has_video": true,
    "has_audio": true,
    "video_codec": "h264",
    "audio_codec": "aac",
    "video_bitrate": 2500,
    "audio_bitrate": 128,
    "frame_rate": 30,
    "resolution": "720p",

    // ========== 原有字段 ==========
    "stream_key": "a1b2c3d4e5f6g7h8",
    "room_id": "room_1234567890",
    "status": "waiting",
    "owner_id": 1
  }
}
```

### 2. 获取直播间信息 API

**接口:** `GET /api/v1/live/:id`

**响应包含所有新字段** (同创建直播间响应)

### 3. 开始/结束直播 API

**无变更** (接口保持不变)

---

## 配置文件变更

### 新增配置 (`configs/config.yaml`)

```yaml
# 流媒体服务器配置（RTMP/HLS/FLV）
streaming:
  # RTMP 推流服务器（用于 OBS 等推流工具）
  rtmp_server: "rtmp://localhost:1935/live"  # RTMP 推流地址前缀
  rtmp_app: "live"                           # RTMP 应用名

  # 播放地址配置
  hls_server: "http://localhost:8080/hls"    # HLS 播放地址前缀
  flv_server: "http://localhost:8080/flv"    # FLV 播放地址前缀
  rtmp_play_server: "rtmp://localhost:1935/live"  # RTMP 播放地址前缀

  # 默认流配置
  default_stream_type: "video_audio"         # 默认流类型
  default_video_codec: "h264"                # 默认视频编解码器
  default_audio_codec: "aac"                 # 默认音频编解码器
  default_video_bitrate: 2500                # 默认视频码率 (kbps)
  default_audio_bitrate: 128                 # 默认音频码率 (kbps)
  default_frame_rate: 30                     # 默认帧率
  default_resolution: "720p"                 # 默认分辨率
```

### Config 结构体变更

**文件:** `internal/config/config.go`

**新增:**

```go
type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Redis     RedisConfig
    JWT       JWTConfig
    Upload    UploadConfig
    Streaming StreamingConfig  // ← 新增
    WebRTC    WebRTCConfig
    OAuth     OAuthConfig
}

// StreamingConfig 流媒体服务器配置
type StreamingConfig struct {
    RTMPServer           string `mapstructure:"rtmp_server"`
    RTMPApp              string `mapstructure:"rtmp_app"`
    HLSServer            string `mapstructure:"hls_server"`
    FLVServer            string `mapstructure:"flv_server"`
    RTMPPlayServer       string `mapstructure:"rtmp_play_server"`
    DefaultStreamType    string `mapstructure:"default_stream_type"`
    DefaultVideoCodec    string `mapstructure:"default_video_codec"`
    DefaultAudioCodec    string `mapstructure:"default_audio_codec"`
    DefaultVideoBitrate  int    `mapstructure:"default_video_bitrate"`
    DefaultAudioBitrate  int    `mapstructure:"default_audio_bitrate"`
    DefaultFrameRate     int    `mapstructure:"default_frame_rate"`
    DefaultResolution    string `mapstructure:"default_resolution"`
}
```

---

## Service 层变更

### 更新文件: `internal/service/live_stream_service.go`

#### 1. 新增依赖注入

```go
type liveStreamServiceImpl struct {
    liveRepo repository.LiveStreamRepository
    banRepo  repository.LiveBanRepository
    cfg      *config.Config  // ← 新增配置注入
}

func NewLiveStreamService(
    liveRepo repository.LiveStreamRepository,
    banRepo repository.LiveBanRepository,
    cfg *config.Config,  // ← 新增参数
) LiveStreamService
```

#### 2. 更新请求结构体

```go
type CreateLiveStreamRequest struct {
    Title        string `json:"title" binding:"required"`
    Description  string `json:"description"`
    Cover        string `json:"cover"`

    // ========== 新增字段 ==========
    PushProtocol string `json:"push_protocol"`
    StreamType   string `json:"stream_type"`
    VideoCodec   string `json:"video_codec"`
    AudioCodec   string `json:"audio_codec"`
    VideoBitrate int    `json:"video_bitrate"`
    AudioBitrate int    `json:"audio_bitrate"`
    FrameRate    int    `json:"frame_rate"`
    Resolution   string `json:"resolution"`
}
```

#### 3. CreateLiveStream 方法增强

**新增功能:**
1. 应用默认配置 (如果请求中未指定)
2. 根据 `stream_type` 自动设置 `has_video` 和 `has_audio`
3. 生成多种播放地址 (HLS, FLV, RTMP, WebRTC)
4. 生成 OBS 推流地址

**核心逻辑:**

```go
func (s *liveStreamServiceImpl) CreateLiveStream(ctx context.Context, userID uint, req *CreateLiveStreamRequest) (*model.LiveStream, error) {
    // 1. 应用默认配置
    streamType := getOrDefault(req.StreamType, s.cfg.Streaming.DefaultStreamType, "video_audio")
    videoCodec := getOrDefault(req.VideoCodec, s.cfg.Streaming.DefaultVideoCodec, "h264")
    // ...

    // 2. 根据流类型设置标志
    hasVideo := streamType == "video_only" || streamType == "video_audio"
    hasAudio := streamType == "audio_only" || streamType == "video_audio"

    // 3. 生成流媒体 URL
    streamURL := generateStreamURL(s.cfg.Streaming.RTMPServer, streamKey)
    playURL := generatePlayURL(s.cfg.Streaming.HLSServer, streamKey)
    flvURL := generatePlayURL(s.cfg.Streaming.FLVServer, streamKey)
    rtmpURL := generatePlayURL(s.cfg.Streaming.RTMPPlayServer, streamKey)
    webrtcURL := generateWebRTCURL(roomID)

    // 4. 创建直播间
    liveStream := &model.LiveStream{
        // ...填充所有字段
    }
}
```

#### 4. 新增辅助函数

```go
// 生成 RTMP 推流地址 (用于 OBS)
func generateStreamURL(rtmpServer, streamKey string) string

// 生成播放地址
func generatePlayURL(server, streamKey string) string

// 生成 WebRTC 播放地址
func generateWebRTCURL(roomID string) string

// 获取配置值或默认值
func getOrDefault(value, configDefault, fallback string) string
func getOrDefaultInt(value, configDefault, fallback int) int
```

---

## Router 层变更

### 更新文件: `internal/router/router.go`

**变更:**

```go
// 初始化 Service 层
liveService := service.NewLiveStreamService(liveRepo, banRepo, cfg)  // ← 新增 cfg 参数
```

---

## 使用场景示例

### 场景 1: 标准音视频直播 (默认)

**创建直播间:**

```json
{
  "title": "我的直播"
}
```

**自动配置:**
- 流类型: `video_audio`
- 视频编码: `h264`
- 音频编码: `aac`
- 视频码率: `2500 Kbps`
- 音频码率: `128 Kbps`
- 帧率: `30 FPS`
- 分辨率: `720p`

**OBS 配置:**
- 服务器: `rtmp://localhost:1935/live`
- 串流密钥: `{返回的 stream_key}`
- 视频编码器: `x264` 或 `NVENC H.264`
- 视频码率: `2500 Kbps`
- 音频码率: `128 Kbps`

### 场景 2: 纯音频直播 (电台/播客)

**创建直播间:**

```json
{
  "title": "深夜电台",
  "stream_type": "audio_only",
  "audio_codec": "aac",
  "audio_bitrate": 192
}
```

**配置结果:**
- `has_video`: `false`
- `has_audio`: `true`
- 不生成视频流

**OBS 配置:**
- 不添加任何视频源
- 仅添加音频输入 (麦克风、桌面音频)

### 场景 3: 高清游戏直播 (1080p @ 60fps)

**创建直播间:**

```json
{
  "title": "《原神》实况",
  "stream_type": "video_audio",
  "video_codec": "h264",
  "audio_codec": "aac",
  "video_bitrate": 6000,
  "audio_bitrate": 192,
  "frame_rate": 60,
  "resolution": "1080p"
}
```

**OBS 配置:**
- 输出分辨率: `1920x1080`
- 帧率: `60 FPS`
- 视频码率: `6000 Kbps`
- 编码器: `NVENC H.264` (硬件加速)
- 音频码率: `192 Kbps`

### 场景 4: 纯视频直播 (风景监控)

**创建直播间:**

```json
{
  "title": "城市风景 24h",
  "stream_type": "video_only",
  "video_codec": "h264",
  "video_bitrate": 3000,
  "frame_rate": 25,
  "resolution": "720p"
}
```

**配置结果:**
- `has_video`: `true`
- `has_audio`: `false`
- 不生成音频流

---

## 部署指南

### 1. 更新数据库

```bash
# 运行数据库迁移
make migrate
```

或手动运行:

```bash
go run cmd/migrate/main.go
```

### 2. 更新配置文件

编辑 `configs/config.yaml`,添加 `streaming` 配置块 (参考上文)

### 3. 部署流媒体服务器

#### 选项 A: Nginx-RTMP (推荐)

**安装:**

```bash
sudo apt update
sudo apt install nginx libnginx-mod-rtmp
```

**配置:** 参考 [docs/OBS_STREAMING_GUIDE.md](docs/OBS_STREAMING_GUIDE.md#推荐流媒体服务器)

#### 选项 B: SRS (Simple RTMP Server)

```bash
docker run -d -p 1935:1935 -p 8080:8080 ossrs/srs:5
```

### 4. 重启应用

```bash
# Docker 环境
make docker-down
make docker-up

# 或直接运行
make build
./main
```

### 5. 测试推流

1. 创建直播间 (调用 API)
2. 获取 `stream_url` 和 `stream_key`
3. 在 OBS 中配置推流地址
4. 点击 "开始串流"
5. 调用开始直播 API
6. 使用播放器访问 `play_url` 观看

---

## 文件变更清单

### 新增文件

| 文件路径 | 说明 |
|---------|------|
| `internal/database/migrations/20251103_add_livestream_obs_support.go` | 数据库迁移脚本 |
| `docs/OBS_STREAMING_GUIDE.md` | OBS 推流配置指南 (完整版,5000+ 字) |
| `LIVE_STREAMING_UPDATE_SUMMARY.md` | 本文档 |

### 修改文件

| 文件路径 | 变更内容 |
|---------|---------|
| `internal/model/live.go` | 添加 11 个新字段 (推流配置 + 流类型配置) |
| `internal/config/config.go` | 添加 `StreamingConfig` 结构体和默认值 |
| `configs/config.yaml` | 添加 `streaming` 配置块 |
| `internal/service/live_stream_service.go` | 更新 `CreateLiveStreamRequest` 和 `CreateLiveStream` 方法,添加辅助函数 |
| `internal/router/router.go` | 更新 `NewLiveStreamService` 调用,传入 `cfg` 参数 |
| `internal/database/migrate.go` | 添加自定义迁移调用 |

---

## 测试指南

### 1. 单元测试

```bash
# 测试 Service 层
go test ./internal/service -v -run TestCreateLiveStream

# 测试迁移
go test ./internal/database/migrations -v
```

### 2. API 集成测试

**创建标准直播间:**

```bash
curl -X POST http://localhost:8080/api/v1/live/create \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试直播"
  }'
```

**创建纯音频直播间:**

```bash
curl -X POST http://localhost:8080/api/v1/live/create \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "电台直播",
    "stream_type": "audio_only",
    "audio_bitrate": 192
  }'
```

**获取直播间信息:**

```bash
curl http://localhost:8080/api/v1/live/1
```

### 3. OBS 推流测试

1. 创建直播间,获取 `stream_url`
2. 在 OBS 中配置:
   - 服务器: `rtmp://localhost:1935/live`
   - 串流密钥: `{stream_key}`
3. 点击 "开始串流"
4. 使用 VLC 或浏览器播放 `play_url`

### 4. 验证流类型

**验证纯音频流:**
- 创建 `stream_type: "audio_only"` 直播间
- 检查 `has_video` 应为 `false`
- 检查 `has_audio` 应为 `true`
- 播放流应该只有音频,无视频画面

**验证纯视频流:**
- 创建 `stream_type: "video_only"` 直播间
- 检查 `has_video` 应为 `true`
- 检查 `has_audio` 应为 `false`
- 播放流应该只有视频画面,无音频

---

## 性能优化建议

### 1. 转码优化

如果需要支持多种清晰度,建议使用 **FFmpeg** 进行实时转码:

```bash
ffmpeg -i rtmp://localhost/live/streamkey \
  -c:v libx264 -preset veryfast -b:v 2500k -s 1280x720 \
  -c:a aac -b:a 128k -f flv rtmp://localhost/live/streamkey_720p \
  -c:v libx264 -preset veryfast -b:v 1000k -s 854x480 \
  -c:a aac -b:a 96k -f flv rtmp://localhost/live/streamkey_480p
```

### 2. CDN 加速

生产环境建议使用 CDN 加速播放:

- 阿里云直播: https://www.aliyun.com/product/live
- 腾讯云直播: https://cloud.tencent.com/product/lvb
- 七牛云直播: https://www.qiniu.com/products/pili

**配置示例 (阿里云):**

```yaml
streaming:
  hls_server: "https://your-domain.alicdn.com/hls"
  flv_server: "https://your-domain.alicdn.com/flv"
```

### 3. 录制与回放

在 Nginx-RTMP 中启用录制:

```nginx
application live {
    live on;
    record all;
    record_path /var/videos;
    record_suffix -%Y%m%d-%H%M%S.flv;
}
```

---

## 向后兼容性

本次更新**完全向后兼容**:

1. ✅ 现有 API 接口保持不变
2. ✅ 现有数据库记录会自动设置默认值
3. ✅ WebRTC 推流功能继续可用
4. ✅ 未指定新参数时,使用默认配置

**迁移路径:**
- 现有直播间自动升级为 `stream_type: "video_audio"`
- 现有 API 调用无需修改即可正常工作
- 新功能通过可选参数逐步启用

---

## 已知问题与限制

### 1. 流媒体服务器依赖

- 需要额外部署 Nginx-RTMP 或 SRS 服务器
- 建议使用 Docker 容器部署以简化配置

### 2. 编解码器兼容性

- `h265` (H.265/HEVC) 需要播放器支持
- `vp9` 在某些浏览器中可能不兼容
- 建议生产环境优先使用 `h264` 和 `aac`

### 3. 播放延迟

| 协议 | 延迟 |
|------|------|
| RTMP | 2-5 秒 |
| HLS | 10-30 秒 |
| FLV | 2-5 秒 |
| WebRTC | < 1 秒 |

**建议:**
- 低延迟场景使用 WebRTC
- 通用场景使用 FLV 或 RTMP
- 移动端场景使用 HLS

---

## 下一步计划

### 短期 (1-2 周)

- [ ] 添加录制与回放功能
- [ ] 集成 CDN 加速
- [ ] 添加流状态监控 (在线/离线检测)
- [ ] 优化推流密钥管理 (重置密钥功能)

### 中期 (1 个月)

- [ ] 支持多清晰度自动切换 (自适应码率)
- [ ] 添加美颜和滤镜 API
- [ ] 支持 SRT 推流协议
- [ ] 直播数据统计和分析

### 长期 (3 个月)

- [ ] 集群化部署 (多服务器负载均衡)
- [ ] AI 内容审核 (鉴黄、鉴暴、鉴政)
- [ ] 直播切片和精彩片段提取
- [ ] 实时转码和水印

---

## 相关文档

- **OBS 推流完整指南**: [docs/OBS_STREAMING_GUIDE.md](docs/OBS_STREAMING_GUIDE.md)
- **API 文档**: [openapi.json](openapi.json)
- **数据库迁移**: [internal/database/migrations/20251103_add_livestream_obs_support.go](internal/database/migrations/20251103_add_livestream_obs_support.go)
- **配置文件**: [configs/config.yaml](configs/config.yaml)
- **LiveStream 模型**: [internal/model/live.go](internal/model/live.go)

---

## 贡献者

- **开发**: Claude Code
- **需求提出**: User
- **测试**: Pending
- **文档**: Claude Code

---

## 许可证

本项目遵循 MIT 许可证。

---

**更新日期**: 2025-11-03
**版本**: v1.1.0
**状态**: ✅ 已完成
