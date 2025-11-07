package event_test

import (
	"context"
	"microvibe-go/pkg/event"
	"microvibe-go/pkg/logger"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func init() {
	// 初始化 logger
	logger.InitLogger("error")
}

// ========================================
// 基础事件测试
// ========================================

func TestBaseEvent(t *testing.T) {
	e := event.NewBaseEvent("test.event")

	if e.Name() != "test.event" {
		t.Errorf("期望事件名称为 test.event, 得到 %s", e.Name())
	}

	if e.Timestamp().IsZero() {
		t.Error("事件时间戳不应该为零")
	}

	if e.Metadata() == nil {
		t.Error("元数据不应该为 nil")
	}

	// 测试元数据
	e.Metadata()["key"] = "value"
	if e.Metadata()["key"] != "value" {
		t.Error("元数据设置失败")
	}
}

// ========================================
// EventBus 订阅测试
// ========================================

func TestEventBus_Subscribe(t *testing.T) {
	bus := event.NewEventBus(2)

	// 创建监听器
	var called int32
	listener := event.NewEventListener("test-listener", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	}, false)

	// 订阅事件
	err := bus.Subscribe("test.event", listener)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	// 重复订阅应该失败
	err = bus.Subscribe("test.event", listener)
	if err == nil {
		t.Error("重复订阅应该返回错误")
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := event.NewEventBus(2)

	listener := event.NewEventListener("test-listener", func(ctx context.Context, e event.Event) error {
		return nil
	}, false)

	// 订阅
	bus.Subscribe("test.event", listener)

	// 取消订阅
	err := bus.Unsubscribe("test.event", "test-listener")
	if err != nil {
		t.Fatalf("取消订阅失败: %v", err)
	}

	// 再次取消订阅应该失败
	err = bus.Unsubscribe("test.event", "test-listener")
	if err == nil {
		t.Error("取消不存在的订阅应该返回错误")
	}
}

// ========================================
// 同步事件发布测试
// ========================================

func TestEventBus_PublishSync(t *testing.T) {
	bus := event.NewEventBus(2)

	var callCount int32
	listener1 := event.NewEventListener("listener1", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}, false)

	listener2 := event.NewEventListener("listener2", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}, false)

	// 订阅
	bus.Subscribe("test.event", listener1)
	bus.Subscribe("test.event", listener2)

	// 发布同步事件
	testEvent := event.NewBaseEvent("test.event")
	err := bus.Publish(context.Background(), testEvent)
	if err != nil {
		t.Fatalf("发布事件失败: %v", err)
	}

	// 验证两个监听器都被调用
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("期望调用2次监听器, 实际调用 %d 次", callCount)
	}
}

// ========================================
// 异步事件发布测试
// ========================================

func TestEventBus_PublishAsync(t *testing.T) {
	bus := event.NewEventBus(2)

	// 启动事件总线
	err := bus.Start()
	if err != nil {
		t.Fatalf("启动事件总线失败: %v", err)
	}
	defer bus.Stop()

	var callCount int32
	var wg sync.WaitGroup
	wg.Add(1)

	listener := event.NewEventListener("async-listener", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&callCount, 1)
		wg.Done()
		return nil
	}, false)

	// 订阅
	bus.Subscribe("test.async", listener)

	// 发布异步事件
	testEvent := event.NewBaseEvent("test.async")
	err = bus.PublishAsync(context.Background(), testEvent)
	if err != nil {
		t.Fatalf("发布异步事件失败: %v", err)
	}

	// 等待处理完成
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("期望调用1次监听器, 实际调用 %d 次", callCount)
	}
}

// ========================================
// 并发测试
// ========================================

func TestEventBus_Concurrent(t *testing.T) {
	bus := event.NewEventBus(4)

	err := bus.Start()
	if err != nil {
		t.Fatalf("启动事件总线失败: %v", err)
	}
	defer bus.Stop()

	var callCount int32
	var publishWg sync.WaitGroup
	var processWg sync.WaitGroup

	// 创建多个监听器
	listenerCount := 5
	eventCount := 10

	processWg.Add(listenerCount * eventCount)

	for i := 0; i < listenerCount; i++ {
		listenerID := "listener-" + string(rune('A'+i))
		listener := event.NewEventListener(listenerID, func(ctx context.Context, e event.Event) error {
			atomic.AddInt32(&callCount, 1)
			processWg.Done()
			return nil
		}, false)
		bus.Subscribe("concurrent.test", listener)
	}

	// 并发发布多个事件
	publishWg.Add(eventCount)

	for i := 0; i < eventCount; i++ {
		go func() {
			defer publishWg.Done()
			testEvent := event.NewBaseEvent("concurrent.test")
			bus.PublishAsync(context.Background(), testEvent)
		}()
	}

	// 等待发布完成
	publishWg.Wait()

	// 等待处理完成（带超时）
	done := make(chan struct{})
	go func() {
		processWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 处理完成
	case <-time.After(5 * time.Second):
		t.Fatal("事件处理超时")
	}

	// 验证：eventCount个事件 × listenerCount个监听器 = 期望调用次数
	expectedCalls := int32(eventCount * listenerCount)
	actualCalls := atomic.LoadInt32(&callCount)

	if actualCalls != expectedCalls {
		t.Errorf("期望调用 %d 次监听器, 实际调用 %d 次", expectedCalls, actualCalls)
	}
}

// ========================================
// 业务事件测试
// ========================================

func TestUserRegisteredEvent(t *testing.T) {
	e := event.NewUserRegisteredEvent(1, "alice", "alice@example.com")

	if e.Name() != event.EventUserRegistered {
		t.Errorf("期望事件名称为 %s, 得到 %s", event.EventUserRegistered, e.Name())
	}

	if e.UserID != 1 {
		t.Errorf("期望用户ID为 1, 得到 %d", e.UserID)
	}

	if e.Username != "alice" {
		t.Errorf("期望用户名为 alice, 得到 %s", e.Username)
	}

	if e.Email != "alice@example.com" {
		t.Errorf("期望邮箱为 alice@example.com, 得到 %s", e.Email)
	}
}

func TestVideoUploadedEvent(t *testing.T) {
	e := event.NewVideoUploadedEvent(100, 1, "测试视频", 120)

	if e.Name() != event.EventVideoUploaded {
		t.Errorf("期望事件名称为 %s, 得到 %s", event.EventVideoUploaded, e.Name())
	}

	if e.VideoID != 100 {
		t.Errorf("期望视频ID为 100, 得到 %d", e.VideoID)
	}

	if e.Duration != 120 {
		t.Errorf("期望视频时长为 120, 得到 %d", e.Duration)
	}
}

// ========================================
// 事件处理器错误处理测试
// ========================================

func TestEventBus_ErrorHandling(t *testing.T) {
	bus := event.NewEventBus(2)

	listener := event.NewEventListener("error-listener", func(ctx context.Context, e event.Event) error {
		return context.DeadlineExceeded // 模拟错误
	}, false)

	bus.Subscribe("error.event", listener)

	// 发布事件（即使处理器出错，也不应该 panic）
	testEvent := event.NewBaseEvent("error.event")
	err := bus.Publish(context.Background(), testEvent)

	if err == nil {
		t.Error("期望返回错误")
	}
}

// ========================================
// 全局事件总线测试
// ========================================

func TestGlobalEventBus(t *testing.T) {
	// 获取全局事件总线
	bus1 := event.GetGlobalEventBus()
	bus2 := event.GetGlobalEventBus()

	// 验证是同一个实例（单例模式）
	if bus1 != bus2 {
		t.Error("全局事件总线应该是单例")
	}

	// 启动
	err := event.StartGlobalEventBus()
	if err != nil {
		t.Fatalf("启动全局事件总线失败: %v", err)
	}

	var called int32
	var wg sync.WaitGroup
	wg.Add(1)

	listener := event.NewEventListener("global-listener", func(ctx context.Context, e event.Event) error {
		atomic.AddInt32(&called, 1)
		wg.Done()
		return nil
	}, false)

	// 使用全局订阅
	err = event.Subscribe("global.test", listener)
	if err != nil {
		t.Fatalf("全局订阅失败: %v", err)
	}

	// 使用全局发布
	testEvent := event.NewBaseEvent("global.test")
	err = event.PublishAsync(context.Background(), testEvent)
	if err != nil {
		t.Fatalf("全局发布失败: %v", err)
	}

	wg.Wait()

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("期望调用1次, 实际调用 %d 次", called)
	}

	// 停止
	err = event.StopGlobalEventBus()
	if err != nil {
		t.Fatalf("停止全局事件总线失败: %v", err)
	}
}

// ========================================
// 性能基准测试
// ========================================

func BenchmarkEventBus_PublishSync(b *testing.B) {
	bus := event.NewEventBus(4)

	listener := event.NewEventListener("bench-listener", func(ctx context.Context, e event.Event) error {
		return nil
	}, false)

	bus.Subscribe("bench.event", listener)

	testEvent := event.NewBaseEvent("bench.event")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(ctx, testEvent)
	}
}

func BenchmarkEventBus_PublishAsync(b *testing.B) {
	bus := event.NewEventBus(4)
	bus.Start()
	defer bus.Stop()

	listener := event.NewEventListener("bench-listener", func(ctx context.Context, e event.Event) error {
		return nil
	}, false)

	bus.Subscribe("bench.async", listener)

	testEvent := event.NewBaseEvent("bench.async")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.PublishAsync(ctx, testEvent)
	}
}

func BenchmarkEventCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.NewUserRegisteredEvent(1, "test", "test@example.com")
	}
}
