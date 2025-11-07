# 事件系统使用文档

## 概述

本项目集成了一个基于生产者/消费者模式的事件监听系统。该系统采用事件驱动架构，支持同步和异步事件处理，实现业务逻辑解耦，提高系统可扩展性和可维护性。

## 核心特性

### 1. 生产者/消费者模式
- 基于 Channel 的事件队列
- 多 Worker 并发处理事件
- 支持异步非阻塞发布

### 2. 事件驱动架构
- 事件发布订阅模式（Pub/Sub）
- 松耦合的业务逻辑
- 易于扩展和维护

### 3. 灵活的事件处理
- 支持同步和异步处理
- 多个监听器订阅同一事件
- 独立的事件处理器

### 4. 类型安全
- 基于接口的事件定义
- 丰富的内置业务事件类型
- 支持自定义事件

### 5. 高可用设计
- 优雅启动和关闭
- 错误隔离（单个监听器失败不影响其他）
- Context 超时控制

## 核心组件

### Event 接口

所有事件必须实现 `Event` 接口：

```go
type Event interface {
    Name() string                      // 事件名称
    Timestamp() time.Time              // 事件发生时间
    Metadata() map[string]interface{}  // 事件元数据
}
```

### EventBus 接口

事件总线接口，负责事件的发布和订阅：

```go
type EventBus interface {
    Subscribe(eventName string, listener *EventListener) error
    Unsubscribe(eventName string, listenerID string) error
    Publish(ctx context.Context, event Event) error
    PublishAsync(ctx context.Context, event Event) error
    Start() error
    Stop() error
}
```

### EventListener

事件监听器：

```go
type EventListener struct {
    ID      string       // 监听器ID
    Handler EventHandler // 处理函数
    Async   bool         // 是否异步处理
}
```

### EventHandler

事件处理器函数类型：

```go
type EventHandler func(ctx context.Context, event Event) error
```

## 快速开始

### 1. 初始化事件总线

在应用启动时初始化全局事件总线：

```go
import (
    "microvibe-go/pkg/event"
)

func main() {
    // 启动全局事件总线
    if err := event.StartGlobalEventBus(); err != nil {
        logger.Fatal("启动事件总线失败", zap.Error(err))
    }

    // 应用退出时停止事件总线
    defer event.StopGlobalEventBus()

    // ... 其他初始化代码
}
```

### 2. 订阅事件

#### 方式一：使用全局函数

```go
// 创建监听器
listener := event.NewEventListener("my-listener", func(ctx context.Context, e event.Event) error {
    // 处理事件
    fmt.Printf("收到事件: %s\n", e.Name())
    return nil
}, false) // false = 同步处理

// 订阅事件
event.Subscribe("user.registered", listener)
```

#### 方式二：创建独立的事件总线

```go
// 创建事件总线（4个工作协程）
bus := event.NewEventBus(4)

// 启动
bus.Start()
defer bus.Stop()

// 订阅
listener := event.NewEventListener("listener-id", handler, false)
bus.Subscribe("event.name", listener)
```

### 3. 发布事件

#### 同步发布

```go
// 创建事件
evt := event.NewBaseEvent("app.started")

// 同步发布（阻塞直到所有监听器处理完成）
err := event.Publish(context.Background(), evt)
if err != nil {
    // 处理错误
}
```

#### 异步发布（推荐）

```go
// 创建事件
evt := event.NewUserRegisteredEvent(1, "alice", "alice@example.com")

// 异步发布（立即返回，后台处理）
err := event.PublishAsync(context.Background(), evt)
if err != nil {
    // 处理错误
}

// 继续执行其他操作...
```

## 内置业务事件

### 用户相关事件

#### 用户注册事件

```go
evt := event.NewUserRegisteredEvent(userID, username, email)
event.PublishAsync(ctx, evt)
```

#### 用户登录事件

```go
evt := event.NewUserLoggedInEvent(userID, ipAddress, userAgent)
event.PublishAsync(ctx, evt)
```

#### 用户更新事件

```go
updatedFields := []string{"age", "avatar"}
evt := event.NewUserUpdatedEvent(userID, updatedFields)
event.PublishAsync(ctx, evt)
```

#### 用户删除事件

```go
evt := event.NewUserDeletedEvent(userID, username)
event.PublishAsync(ctx, evt)
```

### 视频相关事件

#### 视频上传事件

```go
evt := event.NewVideoUploadedEvent(videoID, userID, title, duration)
event.PublishAsync(ctx, evt)
```

#### 视频发布事件

```go
evt := event.NewVideoPublishedEvent(videoID, userID, title, categoryID)
event.PublishAsync(ctx, evt)
```

#### 视频观看事件

```go
evt := event.NewVideoViewedEvent(videoID, userID, watchDuration, ipAddress)
event.PublishAsync(ctx, evt)
```

#### 视频删除事件

```go
evt := event.NewVideoDeletedEvent(videoID, userID)
event.PublishAsync(ctx, evt)
```

### 互动相关事件

#### 视频点赞事件

```go
evt := event.NewVideoLikedEvent(videoID, userID)
event.PublishAsync(ctx, evt)
```

#### 视频评论事件

```go
evt := event.NewVideoCommentedEvent(videoID, userID, commentID, content)
event.PublishAsync(ctx, evt)
```

#### 视频分享事件

```go
evt := event.NewVideoSharedEvent(videoID, userID, "WeChat")
event.PublishAsync(ctx, evt)
```

#### 用户关注事件

```go
evt := event.NewUserFollowedEvent(followerID, followingID)
event.PublishAsync(ctx, evt)
```

#### 用户取消关注事件

```go
evt := event.NewUserUnfollowedEvent(followerID, followingID)
event.PublishAsync(ctx, evt)
```

### 系统事件

#### 系统错误事件

```go
context := map[string]interface{}{
    "module": "user-service",
    "action": "register",
}
evt := event.NewSystemErrorEvent(errorMsg, errorStack, context)
event.PublishAsync(ctx, evt)
```

#### 系统警告事件

```go
context := map[string]interface{}{
    "threshold": 90,
    "current": 95,
}
evt := event.NewSystemWarningEvent("CPU 使用率过高", context)
event.PublishAsync(ctx, evt)
```

## 实战示例

### 示例 1: 用户注册流程

```go
// 在 Repository 层发布事件
func (r *userRepositoryImpl) Create(ctx context.Context, user *model.User) error {
    // 保存用户到数据库
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        return err
    }

    // 发布用户注册事件
    evt := event.NewUserRegisteredEvent(user.ID, user.Username, user.Email)
    event.PublishAsync(ctx, evt)

    return nil
}

// 在应用启动时订阅事件
func setupUserEventListeners() {
    // 监听器1: 发送欢迎邮件
    event.Subscribe(event.EventUserRegistered, event.NewEventListener(
        "send-welcome-email",
        func(ctx context.Context, e event.Event) error {
            userEvent := e.(*event.UserRegisteredEvent)

            // 发送邮件逻辑
            emailService.SendWelcomeEmail(userEvent.Email, userEvent.Username)

            logger.Info("欢迎邮件已发送",
                zap.Uint("user_id", userEvent.UserID),
                zap.String("email", userEvent.Email))

            return nil
        },
        true, // 异步处理
    ))

    // 监听器2: 初始化用户配置
    event.Subscribe(event.EventUserRegistered, event.NewEventListener(
        "init-user-config",
        func(ctx context.Context, e event.Event) error {
            userEvent := e.(*event.UserRegisteredEvent)

            // 创建默认配置
            configService.CreateDefaultConfig(userEvent.UserID)

            return nil
        },
        false, // 同步处理
    ))

    // 监听器3: 发送推送通知
    event.Subscribe(event.EventUserRegistered, event.NewEventListener(
        "push-notification",
        func(ctx context.Context, e event.Event) error {
            userEvent := e.(*event.UserRegisteredEvent)

            // 发送推送通知
            pushService.SendNotification(userEvent.UserID, "欢迎加入！")

            return nil
        },
        true, // 异步处理
    ))

    // 监听器4: 更新统计数据
    event.Subscribe(event.EventUserRegistered, event.NewEventListener(
        "update-statistics",
        func(ctx context.Context, e event.Event) error {
            // 增加用户注册统计
            statsService.IncrementUserCount()
            return nil
        },
        true, // 异步处理
    ))
}
```

### 示例 2: 视频点赞流程

```go
// Service 层
func (s *videoServiceImpl) LikeVideo(ctx context.Context, videoID, userID uint) error {
    // 1. 记录点赞
    if err := s.likeRepo.Create(ctx, &model.Like{
        VideoID: videoID,
        UserID:  userID,
    }); err != nil {
        return err
    }

    // 2. 更新视频点赞数
    if err := s.videoRepo.IncrementLikes(ctx, videoID); err != nil {
        return err
    }

    // 3. 发布点赞事件
    evt := event.NewVideoLikedEvent(videoID, userID)
    event.PublishAsync(ctx, evt)

    return nil
}

// 订阅点赞事件
func setupVideoEventListeners() {
    // 监听器1: 发送通知给视频作者
    event.Subscribe(event.EventVideoLiked, event.NewEventListener(
        "notify-author",
        func(ctx context.Context, e event.Event) error {
            likeEvent := e.(*event.VideoLikedEvent)

            // 查询视频作者
            video, _ := videoRepo.FindByID(ctx, likeEvent.VideoID)

            // 发送通知
            notificationService.SendLikeNotification(video.UserID, likeEvent.UserID)

            return nil
        },
        true,
    ))

    // 监听器2: 更新推荐算法数据
    event.Subscribe(event.EventVideoLiked, event.NewEventListener(
        "update-recommendation",
        func(ctx context.Context, e event.Event) error {
            likeEvent := e.(*event.VideoLikedEvent)

            // 更新推荐模型
            recommendService.UpdateUserPreference(likeEvent.UserID, likeEvent.VideoID)

            return nil
        },
        true,
    ))

    // 监听器3: 更新热度排名
    event.Subscribe(event.EventVideoLiked, event.NewEventListener(
        "update-hot-rank",
        func(ctx context.Context, e event.Event) error {
            likeEvent := e.(*event.VideoLikedEvent)

            // 更新热度分数
            rankService.UpdateHotScore(likeEvent.VideoID, 1)

            return nil
        },
        true,
    ))
}
```

### 示例 3: 视频上传流程

```go
// Handler 层
func (h *VideoHandler) Upload(c *gin.Context) {
    // 1. 上传文件...
    // 2. 保存到数据库...

    // 3. 发布视频上传事件
    evt := event.NewVideoUploadedEvent(video.ID, video.UserID, video.Title, video.Duration)
    event.PublishAsync(c.Request.Context(), evt)

    response.Success(c, video)
}

// 订阅视频上传事件
func setupVideoUploadListeners() {
    // 监听器1: 生成视频缩略图
    event.Subscribe(event.EventVideoUploaded, event.NewEventListener(
        "generate-thumbnail",
        func(ctx context.Context, e event.Event) error {
            videoEvent := e.(*event.VideoUploadedEvent)

            // 生成缩略图（耗时操作）
            thumbnailService.Generate(videoEvent.VideoID)

            return nil
        },
        true, // 异步处理
    ))

    // 监听器2: 视频转码
    event.Subscribe(event.EventVideoUploaded, event.NewEventListener(
        "transcode-video",
        func(ctx context.Context, e event.Event) error {
            videoEvent := e.(*event.VideoUploadedEvent)

            // 添加到转码队列
            transcodeService.AddToQueue(videoEvent.VideoID)

            return nil
        },
        true,
    ))

    // 监听器3: 内容审核
    event.Subscribe(event.EventVideoUploaded, event.NewEventListener(
        "content-review",
        func(ctx context.Context, e event.Event) error {
            videoEvent := e.(*event.VideoUploadedEvent)

            // 提交审核
            reviewService.SubmitReview(videoEvent.VideoID)

            return nil
        },
        true,
    ))
}
```

## 最佳实践

### 1. 事件命名规范

使用小写字母和点号分隔：

```go
"user.registered"     // 用户注册
"user.logged_in"      // 用户登录
"video.uploaded"      // 视频上传
"video.liked"         // 视频点赞
"system.error"        // 系统错误
```

### 2. 监听器 ID 命名

使用小写字母和连字符，描述功能：

```go
"send-welcome-email"     // 发送欢迎邮件
"update-statistics"      // 更新统计
"generate-thumbnail"     // 生成缩略图
```

### 3. 异步 vs 同步

**异步处理（推荐）**：
- ✅ 非关键业务逻辑
- ✅ 耗时操作（发送邮件、生成缩略图）
- ✅ 不影响主流程的操作

**同步处理**：
- ⚠️ 关键业务逻辑
- ⚠️ 需要立即完成的操作
- ⚠️ 有依赖关系的操作

### 4. 错误处理

```go
listener := event.NewEventListener("my-listener", func(ctx context.Context, e event.Event) error {
    // 使用 defer 确保资源清理
    defer func() {
        if r := recover(); r != nil {
            logger.Error("监听器 panic", zap.Any("recover", r))
        }
    }()

    // 处理事件
    if err := processEvent(e); err != nil {
        // 记录错误日志
        logger.Error("事件处理失败",
            zap.String("event", e.Name()),
            zap.Error(err))
        return err
    }

    return nil
}, true)
```

### 5. Context 传递

```go
// 发布事件时传递 Context
ctx := context.WithValue(parentCtx, "request_id", requestID)
event.PublishAsync(ctx, evt)

// 监听器中使用 Context
listener := event.NewEventListener("my-listener", func(ctx context.Context, e event.Event) error {
    // 获取 request_id
    requestID := ctx.Value("request_id")

    // 使用带超时的 Context
    timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return processWithTimeout(timeoutCtx, e)
}, true)
```

### 6. 事件元数据

```go
// 创建事件时添加元数据
evt := event.NewUserRegisteredEvent(userID, username, email)
evt.Metadata()["ip_address"] = "192.168.1.1"
evt.Metadata()["user_agent"] = "Mozilla/5.0..."
evt.Metadata()["source"] = "mobile_app"

// 监听器中读取元数据
listener := event.NewEventListener("listener", func(ctx context.Context, e event.Event) error {
    if source, ok := e.Metadata()["source"].(string); ok {
        logger.Info("事件来源", zap.String("source", source))
    }
    return nil
}, true)
```

## 性能优化

### 1. 调整 Worker 数量

根据系统负载调整工作协程数量：

```go
// 低负载：2-4 个 Worker
bus := event.NewEventBus(2)

// 中等负载：4-8 个 Worker
bus := event.NewEventBus(4)

// 高负载：8-16 个 Worker
bus := event.NewEventBus(16)
```

### 2. 事件通道缓冲

默认缓冲 1000 个事件，可根据需求调整源码：

```go
eventChan: make(chan eventTask, 1000), // 默认
eventChan: make(chan eventTask, 10000), // 高吞吐量
```

### 3. 批量发布

对于大量事件，考虑批量发布：

```go
// 发布多个事件
events := []event.Event{
    event.NewVideoLikedEvent(1, 100),
    event.NewVideoLikedEvent(2, 100),
    event.NewVideoLikedEvent(3, 100),
}

for _, evt := range events {
    event.PublishAsync(ctx, evt)
}
```

## 监控和调试

### 1. 添加日志

```go
listener := event.NewEventListener("my-listener", func(ctx context.Context, e event.Event) error {
    start := time.Now()

    // 处理事件
    err := processEvent(e)

    // 记录处理时间
    duration := time.Since(start)
    logger.Info("事件处理完成",
        zap.String("event", e.Name()),
        zap.Duration("duration", duration),
        zap.Error(err))

    return err
}, true)
```

### 2. 统计事件数量

```go
var eventCount int64

listener := event.NewEventListener("counter", func(ctx context.Context, e event.Event) error {
    atomic.AddInt64(&eventCount, 1)

    // 每处理 1000 个事件输出一次
    if atomic.LoadInt64(&eventCount)%1000 == 0 {
        logger.Info("事件处理统计", zap.Int64("count", eventCount))
    }

    return nil
}, true)
```

## 注意事项

### 1. 避免循环依赖

❌ 错误：
```go
// 监听器 A 发布事件 B
event.Subscribe("event.a", event.NewEventListener("a", func(ctx context.Context, e event.Event) error {
    event.PublishAsync(ctx, event.NewBaseEvent("event.b"))
    return nil
}, true))

// 监听器 B 发布事件 A（循环依赖！）
event.Subscribe("event.b", event.NewEventListener("b", func(ctx context.Context, e event.Event) error {
    event.PublishAsync(ctx, event.NewBaseEvent("event.a"))
    return nil
}, true))
```

### 2. 监听器不应阻塞

❌ 错误：
```go
listener := event.NewEventListener("blocker", func(ctx context.Context, e event.Event) error {
    time.Sleep(10 * time.Minute) // 长时间阻塞！
    return nil
}, false)
```

✅ 正确：
```go
listener := event.NewEventListener("non-blocker", func(ctx context.Context, e event.Event) error {
    // 使用 goroutine 处理耗时操作
    go func() {
        time.Sleep(10 * time.Minute)
    }()
    return nil
}, true)
```

### 3. 事件总线生命周期

```go
// ✅ 正确：全局事件总线自动管理
event.StartGlobalEventBus()
defer event.StopGlobalEventBus()

// ❌ 错误：忘记启动
bus := event.NewEventBus(4)
// 忘记调用 bus.Start()
bus.PublishAsync(ctx, evt) // 会返回错误！
```

### 4. 并发安全

事件总线是并发安全的，可以在多个 goroutine 中安全使用：

```go
// ✅ 安全：并发发布
for i := 0; i < 100; i++ {
    go func(id int) {
        evt := event.NewVideoLikedEvent(uint(id), uint(id))
        event.PublishAsync(ctx, evt)
    }(i)
}
```

## 测试

### 单元测试示例

```go
func TestUserService_Register(t *testing.T) {
    // 创建测试用的事件总线
    bus := event.NewEventBus(2)
    bus.Start()
    defer bus.Stop()

    var receivedEvent *event.UserRegisteredEvent
    var wg sync.WaitGroup
    wg.Add(1)

    // 订阅测试监听器
    listener := event.NewEventListener("test-listener", func(ctx context.Context, e event.Event) error {
        receivedEvent = e.(*event.UserRegisteredEvent)
        wg.Done()
        return nil
    }, false)

    bus.Subscribe(event.EventUserRegistered, listener)

    // 执行测试
    userService.Register(ctx, "test", "test@example.com")

    // 等待事件处理
    wg.Wait()

    // 验证
    if receivedEvent == nil {
        t.Fatal("未收到事件")
    }
    if receivedEvent.Username != "test" {
        t.Errorf("期望用户名 test, 得到 %s", receivedEvent.Username)
    }
}
```

## 参考文档

- **示例代码**: `examples/event/event_example.go`
- **单元测试**: `test/event_test.go`
- **事件定义**: `pkg/event/events.go`

## 总结

事件系统提供了：
- ✅ **解耦业务逻辑**：发布者和订阅者互不依赖
- ✅ **异步处理**：提高系统响应速度
- ✅ **可扩展性**：轻松添加新的事件监听器
- ✅ **高性能**：基于 Channel 的并发处理
- ✅ **类型安全**：丰富的内置事件类型
- ✅ **易于测试**：独立的事件总线实例

通过事件驱动架构，实现了松耦合、高内聚的系统设计！
