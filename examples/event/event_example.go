package main

import (
	"context"
	"fmt"
	"microvibe-go/pkg/event"
	"microvibe-go/pkg/logger"
	"time"
)

func main() {
	// 初始化 logger
	logger.InitLogger("info")

	fmt.Println("=== 事件系统使用示例 ===")

	example1BasicEvent()
	example2AsyncEvent()
	example3MultipleListeners()
	example4BusinessEvents()
	example5ErrorHandling()

	fmt.Println("\n✅ 所有示例完成！")
}

// 示例 1: 基础事件使用
func example1BasicEvent() {
	fmt.Println("--- 示例 1: 基础事件使用 ---")

	// 创建事件总线
	bus := event.NewEventBus(2)

	// 创建监听器
	listener := event.NewEventListener("print-listener", func(ctx context.Context, e event.Event) error {
		fmt.Printf("📢 收到事件: %s, 时间: %s\n", e.Name(), e.Timestamp().Format("15:04:05"))
		return nil
	}, false)

	// 订阅事件
	bus.Subscribe("app.started", listener)

	// 发布同步事件
	startEvent := event.NewBaseEvent("app.started")
	bus.Publish(context.Background(), startEvent)

	fmt.Println()
}

// 示例 2: 异步事件处理
func example2AsyncEvent() {
	fmt.Println("--- 示例 2: 异步事件处理 ---")

	// 创建并启动事件总线
	bus := event.NewEventBus(4)
	bus.Start()
	defer bus.Stop()

	// 创建异步监听器
	listener := event.NewEventListener("async-listener", func(ctx context.Context, e event.Event) error {
		// 模拟耗时操作
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("✅ 异步处理完成: %s\n", e.Name())
		return nil
	}, true) // async = true

	// 订阅
	bus.Subscribe("task.process", listener)

	// 发布异步事件
	fmt.Println("📤 发布异步事件...")
	bus.PublishAsync(context.Background(), event.NewBaseEvent("task.process"))

	// 继续执行其他操作
	fmt.Println("👉 继续执行主流程...")

	// 等待异步处理完成
	time.Sleep(200 * time.Millisecond)
	fmt.Println()
}

// 示例 3: 多个监听器
func example3MultipleListeners() {
	fmt.Println("--- 示例 3: 多个监听器订阅同一事件 ---")

	bus := event.NewEventBus(2)
	bus.Start()
	defer bus.Stop()

	// 创建多个监听器
	listener1 := event.NewEventListener("email-notifier", func(ctx context.Context, e event.Event) error {
		fmt.Println("📧 发送邮件通知")
		return nil
	}, false)

	listener2 := event.NewEventListener("sms-notifier", func(ctx context.Context, e event.Event) error {
		fmt.Println("📱 发送短信通知")
		return nil
	}, false)

	listener3 := event.NewEventListener("logger", func(ctx context.Context, e event.Event) error {
		fmt.Println("📝 记录日志")
		return nil
	}, false)

	// 订阅
	bus.Subscribe("user.registered", listener1)
	bus.Subscribe("user.registered", listener2)
	bus.Subscribe("user.registered", listener3)

	// 发布事件（所有监听器都会被调用）
	bus.PublishAsync(context.Background(), event.NewBaseEvent("user.registered"))

	time.Sleep(100 * time.Millisecond)
	fmt.Println()
}

// 示例 4: 业务事件
func example4BusinessEvents() {
	fmt.Println("--- 示例 4: 业务事件 ---")

	bus := event.NewEventBus(4)
	bus.Start()
	defer bus.Stop()

	// 用户注册事件监听器
	bus.Subscribe(event.EventUserRegistered, event.NewEventListener(
		"welcome-email",
		func(ctx context.Context, e event.Event) error {
			if userEvent, ok := e.(*event.UserRegisteredEvent); ok {
				fmt.Printf("👋 发送欢迎邮件给: %s (%s)\n",
					userEvent.Username, userEvent.Email)
			}
			return nil
		},
		false,
	))

	// 视频上传事件监听器
	bus.Subscribe(event.EventVideoUploaded, event.NewEventListener(
		"video-processor",
		func(ctx context.Context, e event.Event) error {
			if videoEvent, ok := e.(*event.VideoUploadedEvent); ok {
				fmt.Printf("🎬 处理视频: %s (时长: %d 秒)\n",
					videoEvent.Title, videoEvent.Duration)
			}
			return nil
		},
		false,
	))

	// 视频点赞事件监听器
	bus.Subscribe(event.EventVideoLiked, event.NewEventListener(
		"like-counter",
		func(ctx context.Context, e event.Event) error {
			if likeEvent, ok := e.(*event.VideoLikedEvent); ok {
				fmt.Printf("👍 用户 %d 点赞了视频 %d\n",
					likeEvent.UserID, likeEvent.VideoID)
			}
			return nil
		},
		false,
	))

	// 发布业务事件
	ctx := context.Background()

	// 用户注册
	bus.PublishAsync(ctx, event.NewUserRegisteredEvent(1, "alice", "alice@example.com"))
	time.Sleep(50 * time.Millisecond)

	// 视频上传
	bus.PublishAsync(ctx, event.NewVideoUploadedEvent(100, 1, "我的第一个视频", 120))
	time.Sleep(50 * time.Millisecond)

	// 视频点赞
	bus.PublishAsync(ctx, event.NewVideoLikedEvent(100, 2))
	time.Sleep(50 * time.Millisecond)

	fmt.Println()
}

// 示例 5: 错误处理
func example5ErrorHandling() {
	fmt.Println("--- 示例 5: 错误处理 ---")

	bus := event.NewEventBus(2)

	// 会返回错误的监听器
	errorListener := event.NewEventListener("error-listener", func(ctx context.Context, e event.Event) error {
		fmt.Println("❌ 监听器处理失败")
		return fmt.Errorf("处理失败: 模拟错误")
	}, false)

	// 正常的监听器
	normalListener := event.NewEventListener("normal-listener", func(ctx context.Context, e event.Event) error {
		fmt.Println("✅ 监听器处理成功")
		return nil
	}, false)

	// 订阅
	bus.Subscribe("test.error", errorListener)
	bus.Subscribe("test.error", normalListener)

	// 发布事件（即使有监听器失败，其他监听器仍会执行）
	err := bus.Publish(context.Background(), event.NewBaseEvent("test.error"))
	if err != nil {
		fmt.Printf("⚠️  部分监听器处理失败: %v\n", err)
	}

	fmt.Println()
}

// ========================================
// 完整示例：用户注册流程
// ========================================

func exampleUserRegistrationFlow() {
	fmt.Println("=== 完整示例：用户注册流程 ===")

	// 创建并启动事件总线
	bus := event.NewEventBus(4)
	bus.Start()
	defer bus.Stop()

	// 订阅用户注册事件 - 发送欢迎邮件
	bus.Subscribe(event.EventUserRegistered, event.NewEventListener(
		"send-welcome-email",
		func(ctx context.Context, e event.Event) error {
			userEvent := e.(*event.UserRegisteredEvent)
			fmt.Printf("📧 发送欢迎邮件给: %s\n", userEvent.Email)
			time.Sleep(100 * time.Millisecond) // 模拟发送邮件
			return nil
		},
		true, // 异步处理
	))

	// 订阅用户注册事件 - 初始化用户配置
	bus.Subscribe(event.EventUserRegistered, event.NewEventListener(
		"init-user-config",
		func(ctx context.Context, e event.Event) error {
			userEvent := e.(*event.UserRegisteredEvent)
			fmt.Printf("⚙️  初始化用户配置: %s\n", userEvent.Username)
			return nil
		},
		false, // 同步处理
	))

	// 订阅用户注册事件 - 推送通知
	bus.Subscribe(event.EventUserRegistered, event.NewEventListener(
		"push-notification",
		func(ctx context.Context, e event.Event) error {
			userEvent := e.(*event.UserRegisteredEvent)
			fmt.Printf("📱 发送推送通知: 欢迎 %s\n", userEvent.Username)
			return nil
		},
		true, // 异步处理
	))

	// 订阅用户注册事件 - 统计分析
	bus.Subscribe(event.EventUserRegistered, event.NewEventListener(
		"user-analytics",
		func(ctx context.Context, e event.Event) error {
			fmt.Println("📊 更新用户统计数据")
			return nil
		},
		true, // 异步处理
	))

	// 模拟用户注册
	fmt.Println("👤 用户注册中...")
	userEvent := event.NewUserRegisteredEvent(1, "bob", "bob@example.com")

	// 发布注册事件
	bus.PublishAsync(context.Background(), userEvent)

	// 继续主流程
	fmt.Println("✅ 注册完成，继续其他操作...")

	// 等待所有事件处理完成
	time.Sleep(300 * time.Millisecond)

	fmt.Println("\n所有后续处理已完成！")
}
