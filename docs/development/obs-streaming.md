# OBS 推流配置指南

本文档详细说明如何使用 **OBS Studio** 或其他推流工具进行直播推流。

## 目录

- [系统架构](#系统架构)
- [快速开始](#快速开始)
- [OBS Studio 配置](#obs-studio-配置)
- [流类型配置](#流类型配置)
- [API 使用说明](#api-使用说明)
- [故障排除](#故障排除)

---

## 系统架构

MicroVibe-Go 支持两种直播推流方式:

### 1. **RTMP 推流** (推荐用于 OBS)
- 推流协议: RTMP (Real-Time Messaging Protocol)
- 适用场景: OBS Studio, vMix, XSplit 等专业推流软件
- 播放格式: HLS, FLV, RTMP
- 优点: 稳定性好、延迟低、兼容性强

### 2. **WebRTC 推流** (推荐用于浏览器)
- 推流协议: WebRTC
- 适用场景: 浏览器内直播、低延迟互动
- 播放格式: WebRTC
- 优点: 超低延迟、P2P 传输

---

## 快速开始

### 第 1 步: 创建直播间

调用 API 创建直播间，获取推流地址和密钥:

```bash
POST /api/v1/live/create
Authorization: Bearer YOUR_JWT_TOKEN
Content-Type: application/json

{
  "title": "我的第一场直播",
  "description": "测试OBS推流",
  "cover": "https://example.com/cover.jpg",
  "push_protocol": "rtmp",
  "stream_type": "video_audio",
  "video_codec": "h264",
  "audio_codec": "aac",
  "video_bitrate": 2500,
  "audio_bitrate": 128,
  "frame_rate": 30,
  "resolution": "720p"
}
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "title": "我的第一场直播",
    "stream_key": "a1b2c3d4e5f6g7h8",
    "room_id": "room_1234567890",
    "stream_url": "rtmp://localhost:1935/live/a1b2c3d4e5f6g7h8",
    "play_url": "http://localhost:8080/hls/a1b2c3d4e5f6g7h8",
    "flv_url": "http://localhost:8080/flv/a1b2c3d4e5f6g7h8",
    "rtmp_url": "rtmp://localhost:1935/live/a1b2c3d4e5f6g7h8",
    "webrtc_url": "webrtc://room/room_1234567890",
    "push_protocol": "rtmp",
    "stream_type": "video_audio",
    "video_codec": "h264",
    "audio_codec": "aac",
    "video_bitrate": 2500,
    "audio_bitrate": 128,
    "frame_rate": 30,
    "resolution": "720p",
    "status": "waiting"
  }
}
```

**重要字段说明:**
- `stream_url`: **推流地址** (配置到 OBS 的服务器和串流密钥)
- `stream_key`: **推流密钥** (仅主播可见，请妥善保管)
- `play_url`: HLS 播放地址 (观众端播放)
- `flv_url`: FLV 播放地址
- `rtmp_url`: RTMP 播放地址

### 第 2 步: 配置 OBS Studio

将 `stream_url` 拆分为:
- **服务器**: `rtmp://localhost:1935/live`
- **串流密钥**: `a1b2c3d4e5f6g7h8`

详见 [OBS Studio 配置](#obs-studio-配置)

### 第 3 步: 开始推流

1. 在 OBS 中点击 **"开始串流"**
2. 调用 API 开始直播:

```bash
POST /api/v1/live/start
Authorization: Bearer YOUR_JWT_TOKEN
Content-Type: application/json

{
  "stream_key": "a1b2c3d4e5f6g7h8"
}
```

3. 观众可以通过 `play_url` 或 `flv_url` 观看直播

### 第 4 步: 结束直播

```bash
POST /api/v1/live/end
Authorization: Bearer YOUR_JWT_TOKEN
Content-Type: application/json

{
  "stream_key": "a1b2c3d4e5f6g7h8"
}
```

---

## OBS Studio 配置

### 1. 下载并安装 OBS Studio

官网: [https://obsproject.com/](https://obsproject.com/)

支持平台: Windows, macOS, Linux

### 2. 配置推流设置

#### 方法 A: 自定义串流服务器

1. 打开 OBS Studio
2. 点击 **设置 (Settings)** → **串流 (Stream)**
3. 配置如下:

| 配置项 | 值 |
|--------|------|
| **服务** | `自定义...` |
| **服务器** | `rtmp://localhost:1935/live` (从 `stream_url` 中提取) |
| **串流密钥** | `a1b2c3d4e5f6g7h8` (API 返回的 `stream_key`) |

4. 点击 **确定 (OK)**

#### 方法 B: 通过完整 URL 配置

某些 OBS 版本支持直接粘贴完整 RTMP URL:

```
rtmp://localhost:1935/live/a1b2c3d4e5f6g7h8
```

### 3. 配置输出设置

点击 **设置 (Settings)** → **输出 (Output)** → **串流 (Streaming)**

#### 推荐配置 (720p @ 30fps):

| 配置项 | 值 | 说明 |
|--------|------|------|
| **编码器** | `x264` 或 `NVIDIA NVENC H.264` | 软件编码 或 硬件加速 |
| **比特率控制** | `CBR` (恒定比特率) | 稳定的码率 |
| **视频比特率** | `2500 Kbps` | 对应 API 的 `video_bitrate` |
| **音频比特率** | `128 Kbps` | 对应 API 的 `audio_bitrate` |
| **关键帧间隔** | `2` 秒 | 建议值 |

#### 高清配置 (1080p @ 60fps):

| 配置项 | 值 |
|--------|------|
| **视频比特率** | `5000 - 8000 Kbps` |
| **音频比特率** | `192 Kbps` |
| **关键帧间隔** | `2` 秒 |
| **编码器预设** | `veryfast` (x264) 或 `Quality` (NVENC) |

### 4. 配置视频设置

点击 **设置 (Settings)** → **视频 (Video)**

| 分辨率 | 基础分辨率 | 输出分辨率 | 帧率 |
|--------|-----------|-----------|------|
| **360p** | 1920x1080 | 640x360 | 30 FPS |
| **480p** | 1920x1080 | 854x480 | 30 FPS |
| **720p** | 1920x1080 | 1280x720 | 30 FPS |
| **1080p** | 1920x1080 | 1920x1080 | 30/60 FPS |

**常用帧率 (FPS) 值:**
- `30`: 默认，适合一般场景
- `60`: 高刷新率，适合游戏直播

### 5. 配置音频设置

点击 **设置 (Settings)** → **音频 (Audio)**

| 配置项 | 值 |
|--------|------|
| **采样率** | `44.1 kHz` 或 `48 kHz` |
| **声道** | `立体声 (Stereo)` |

---

## 流类型配置

MicroVibe-Go 支持灵活的流类型配置，满足不同直播场景需求。

### 1. 流类型 (Stream Type)

| 类型 | 说明 | 适用场景 |
|------|------|---------|
| `video_audio` | **音视频** (默认) | 标准直播 |
| `video_only` | **纯视频** (无音频) | 风景/监控直播 |
| `audio_only` | **纯音频** (无视频) | 电台/播客直播 |

**API 配置示例:**

```json
{
  "stream_type": "audio_only",
  "audio_codec": "aac",
  "audio_bitrate": 128
}
```

**OBS 配置:**
- **纯音频直播**: 在 OBS 中不添加任何视频源，仅添加音频输入
- **纯视频直播**: 在 OBS 音频设置中禁用麦克风和桌面音频

### 2. 视频编解码器 (Video Codec)

| 编解码器 | 说明 | 兼容性 | 推荐场景 |
|---------|------|--------|---------|
| `h264` | **H.264 / AVC** (默认) | ⭐⭐⭐⭐⭐ 最佳 | 通用场景 |
| `h265` | **H.265 / HEVC** | ⭐⭐⭐ 中等 | 高清直播 (需解码器支持) |
| `vp8` | **VP8** | ⭐⭐⭐⭐ 良好 | WebRTC 场景 |
| `vp9` | **VP9** | ⭐⭐⭐ 中等 | 高压缩率场景 |

**OBS 编码器对照表:**

| API Codec | OBS 编码器名称 |
|-----------|---------------|
| `h264` | `x264`, `NVIDIA NVENC H.264`, `QuickSync H.264` |
| `h265` | `NVIDIA NVENC H.265 (HEVC)`, `QuickSync H.265` |

### 3. 音频编解码器 (Audio Codec)

| 编解码器 | 说明 | 兼容性 | 推荐场景 |
|---------|------|--------|---------|
| `aac` | **AAC** (默认) | ⭐⭐⭐⭐⭐ 最佳 | 通用场景 |
| `opus` | **Opus** | ⭐⭐⭐⭐ 良好 | 低延迟/高质量音频 |
| `mp3` | **MP3** | ⭐⭐⭐ 中等 | 兼容性场景 |

**OBS 音频编码器设置:**
- 默认音频编码器通常为 `AAC`
- 在 **输出 → 音频** 中可以调整音频比特率

### 4. 码率配置 (Bitrate)

#### 视频码率 (Video Bitrate)

| 分辨率 | 30 FPS | 60 FPS |
|--------|--------|--------|
| **360p** | 600-1000 Kbps | 1000-1500 Kbps |
| **480p** | 1000-2000 Kbps | 2000-3000 Kbps |
| **720p** | 2500-4000 Kbps | 4500-6000 Kbps |
| **1080p** | 4500-6000 Kbps | 7000-10000 Kbps |
| **2K** | 8000-12000 Kbps | 12000-16000 Kbps |
| **4K** | 15000-25000 Kbps | 25000-40000 Kbps |

#### 音频码率 (Audio Bitrate)

| 质量 | 码率 | 说明 |
|------|------|------|
| **低** | 64 Kbps | 语音通话 |
| **中** | 128 Kbps | 标准音质 (默认) |
| **高** | 192-256 Kbps | 音乐直播 |
| **极高** | 320 Kbps | 无损级音质 |

### 5. 帧率 (Frame Rate)

| 帧率 | 说明 | 适用场景 |
|------|------|---------|
| **15 FPS** | 低帧率 | 省带宽场景 |
| **24 FPS** | 电影级 | 电影风格直播 |
| **30 FPS** | 标准 (默认) | 一般直播 |
| **60 FPS** | 高帧率 | 游戏/运动直播 |

### 6. 分辨率 (Resolution)

| 分辨率 | 像素 | 说明 |
|--------|------|------|
| **360p** | 640x360 | 低清 |
| **480p** | 854x480 | 标清 |
| **720p** | 1280x720 | 高清 (默认) |
| **1080p** | 1920x1080 | 全高清 |
| **2K** | 2560x1440 | 2K 超清 |
| **4K** | 3840x2160 | 4K 超高清 |

---

## API 使用说明

### 1. 创建直播间 (支持流类型配置)

```bash
POST /api/v1/live/create
Authorization: Bearer YOUR_JWT_TOKEN
```

**完整请求示例:**

```json
{
  "title": "专业游戏直播",
  "description": "《原神》实况",
  "cover": "https://example.com/cover.jpg",

  "push_protocol": "rtmp",

  "stream_type": "video_audio",
  "video_codec": "h264",
  "audio_codec": "aac",
  "video_bitrate": 5000,
  "audio_bitrate": 192,
  "frame_rate": 60,
  "resolution": "1080p"
}
```

**最简请求 (使用默认配置):**

```json
{
  "title": "我的直播"
}
```

将自动应用默认配置:
- `push_protocol`: `rtmp`
- `stream_type`: `video_audio`
- `video_codec`: `h264`
- `audio_codec`: `aac`
- `video_bitrate`: `2500` Kbps
- `audio_bitrate`: `128` Kbps
- `frame_rate`: `30` FPS
- `resolution`: `720p`

### 2. 纯音频直播示例 (电台/播客)

```json
{
  "title": "深夜电台",
  "description": "分享音乐和故事",

  "push_protocol": "rtmp",
  "stream_type": "audio_only",
  "audio_codec": "aac",
  "audio_bitrate": 192
}
```

**OBS 配置:**
- 不添加任何视频源 (摄像头、屏幕捕获等)
- 仅添加音频输入 (麦克风、桌面音频等)

### 3. 纯视频直播示例 (风景/监控)

```json
{
  "title": "城市风景 24 小时直播",

  "push_protocol": "rtmp",
  "stream_type": "video_only",
  "video_codec": "h264",
  "video_bitrate": 3000,
  "frame_rate": 25,
  "resolution": "720p"
}
```

**OBS 配置:**
- 在音频设置中禁用所有音频输入
- 或静音麦克风和桌面音频轨道

### 4. 开始/结束直播

**开始直播:**

```bash
POST /api/v1/live/start
Authorization: Bearer YOUR_JWT_TOKEN

{
  "stream_key": "YOUR_STREAM_KEY"
}
```

**结束直播:**

```bash
POST /api/v1/live/end
Authorization: Bearer YOUR_JWT_TOKEN

{
  "stream_key": "YOUR_STREAM_KEY"
}
```

### 5. 获取直播间信息

```bash
GET /api/v1/live/:id
```

**响应包含:**
- 推流地址 (`stream_url`)
- 播放地址 (`play_url`, `flv_url`, `rtmp_url`, `webrtc_url`)
- 流配置 (`stream_type`, `video_codec`, `audio_codec` 等)
- 统计数据 (`view_count`, `like_count`, `online_count` 等)

---

## 故障排除

### 1. OBS 连接失败

**错误提示:** "Failed to connect to server"

**解决方案:**
1. 检查 RTMP 服务器地址是否正确
2. 确认流媒体服务器 (如 Nginx-RTMP) 已启动
3. 检查防火墙是否允许 1935 端口
4. 验证 `stream_key` 是否正确

### 2. 推流成功但无画面

**可能原因:**
- 纯音频直播 (`stream_type: "audio_only"`) 不会有视频画面
- OBS 未添加视频源

**解决方案:**
- 检查直播间的 `stream_type` 配置
- 在 OBS 中添加视频源 (摄像头、屏幕捕获等)

### 3. 推流成功但无声音

**可能原因:**
- 纯视频直播 (`stream_type: "video_only"`) 不会有音频
- OBS 音频设备未启用

**解决方案:**
- 检查直播间的 `stream_type` 配置
- 在 OBS 中启用音频输入设备

### 4. 延迟过高

**解决方案:**
1. 降低视频码率和分辨率
2. 使用硬件编码器 (NVENC, QuickSync)
3. 调整 OBS 编码器预设为 `ultrafast` 或 `superfast`
4. 使用 WebRTC 推流 (超低延迟)

### 5. 画面卡顿

**解决方案:**
1. 降低视频码率
2. 降低分辨率或帧率
3. 检查网络带宽 (建议上行速度 > 视频码率 * 1.5)
4. 使用 CBR (恒定比特率) 而非 VBR

### 6. 查看服务器日志

**检查流媒体服务器状态:**

```bash
# 查看 Nginx-RTMP 日志
tail -f /var/log/nginx/error.log

# 检查端口是否监听
netstat -tulnp | grep 1935
```

---

## 推荐流媒体服务器

### 选项 1: Nginx-RTMP (开源)

**安装 (Ubuntu/Debian):**

```bash
sudo apt update
sudo apt install nginx libnginx-mod-rtmp
```

**配置 (`/etc/nginx/nginx.conf`):**

```nginx
rtmp {
    server {
        listen 1935;
        chunk_size 4096;

        application live {
            live on;
            record off;

            # HLS 输出
            hls on;
            hls_path /tmp/hls;
            hls_fragment 3;
            hls_playlist_length 60;

            # FLV 输出
            exec ffmpeg -i rtmp://localhost/live/$name
                -c:v copy -c:a copy -f flv rtmp://localhost/flv/$name;
        }

        application flv {
            live on;
            record off;
        }
    }
}

http {
    server {
        listen 8080;

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
        }
    }
}
```

**启动服务:**

```bash
sudo systemctl restart nginx
```

### 选项 2: SRS (Simple RTMP Server)

**安装:**

```bash
docker run -d -p 1935:1935 -p 8080:8080 ossrs/srs:5
```

**配置文件:** 参考 [SRS 官方文档](https://github.com/ossrs/srs)

---

## 总结

通过本指南,你已经学会:

1. ✅ 创建支持 OBS 推流的直播间
2. ✅ 配置 OBS Studio 进行 RTMP 推流
3. ✅ 灵活配置流类型 (纯音频、纯视频、音视频)
4. ✅ 优化编码器、码率、分辨率等参数
5. ✅ 故障排除和调试

**下一步:**
- 部署流媒体服务器 (Nginx-RTMP 或 SRS)
- 集成 CDN 加速 (阿里云、腾讯云)
- 配置录制和回放功能
- 添加美颜和滤镜效果

有任何问题,请参考 [项目文档](../README.md) 或提交 Issue。

---

**相关文档:**
- [API 文档](../openapi.json)
- [数据库迁移](../internal/database/migrations/20251103_add_livestream_obs_support.go)
- [配置文件](../configs/config.yaml)
