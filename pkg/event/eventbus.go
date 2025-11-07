package event

import (
	"context"
	"fmt"
	"sync"
	"time"

	"microvibe-go/pkg/logger"

	"go.uber.org/zap"
)

// EventBus 事件总线接口
type EventBus interface {
	// Subscribe 订阅事件
	Subscribe(eventName string, listener *EventListener) error

	// Unsubscribe 取消订阅
	Unsubscribe(eventName string, listenerID string) error

	// Publish 发布事件（同步）
	Publish(ctx context.Context, event Event) error

	// PublishAsync 发布事件（异步）
	PublishAsync(ctx context.Context, event Event) error

	// Start 启动事件总线
	Start() error

	// Stop 停止事件总线
	Stop() error
}

// eventBusImpl 事件总线实现
type eventBusImpl struct {
	listeners map[string][]*EventListener // 事件名 -> 监听器列表
	mu        sync.RWMutex                // 读写锁

	eventChan chan eventTask // 异步事件通道
	workerNum int            // 工作协程数量

	ctx    context.Context    // 上下文
	cancel context.CancelFunc // 取消函数
	wg     sync.WaitGroup     // 等待组

	started bool // 是否已启动
}

// eventTask 事件任务
type eventTask struct {
	ctx   context.Context
	event Event
}

// NewEventBus 创建事件总线
func NewEventBus(workerNum int) EventBus {
	if workerNum <= 0 {
		workerNum = 4 // 默认4个工作协程
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &eventBusImpl{
		listeners: make(map[string][]*EventListener),
		eventChan: make(chan eventTask, 1000), // 缓冲1000个事件
		workerNum: workerNum,
		ctx:       ctx,
		cancel:    cancel,
		started:   false,
	}
}

// Subscribe 订阅事件
func (bus *eventBusImpl) Subscribe(eventName string, listener *EventListener) error {
	if listener == nil {
		return fmt.Errorf("listener cannot be nil")
	}
	if listener.ID == "" {
		return fmt.Errorf("listener ID cannot be empty")
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	// 检查是否已存在相同ID的监听器
	for _, l := range bus.listeners[eventName] {
		if l.ID == listener.ID {
			return fmt.Errorf("listener with ID %s already exists for event %s", listener.ID, eventName)
		}
	}

	bus.listeners[eventName] = append(bus.listeners[eventName], listener)

	logger.Debug("事件监听器已订阅",
		zap.String("event", eventName),
		zap.String("listener_id", listener.ID),
		zap.Bool("async", listener.Async))

	return nil
}

// Unsubscribe 取消订阅
func (bus *eventBusImpl) Unsubscribe(eventName string, listenerID string) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	listeners, exists := bus.listeners[eventName]
	if !exists {
		return fmt.Errorf("no listeners found for event %s", eventName)
	}

	// 查找并移除监听器
	for i, listener := range listeners {
		if listener.ID == listenerID {
			// 删除监听器
			bus.listeners[eventName] = append(listeners[:i], listeners[i+1:]...)

			logger.Debug("事件监听器已取消订阅",
				zap.String("event", eventName),
				zap.String("listener_id", listenerID))

			return nil
		}
	}

	return fmt.Errorf("listener %s not found for event %s", listenerID, eventName)
}

// Publish 发布事件（同步）
func (bus *eventBusImpl) Publish(ctx context.Context, event Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	bus.mu.RLock()
	listeners, exists := bus.listeners[event.Name()]
	bus.mu.RUnlock()

	if !exists || len(listeners) == 0 {
		logger.Debug("没有监听器订阅此事件",
			zap.String("event", event.Name()))
		return nil
	}

	logger.Debug("发布同步事件",
		zap.String("event", event.Name()),
		zap.Int("listeners", len(listeners)))

	// 同步执行所有监听器
	var errors []error
	for _, listener := range listeners {
		if err := bus.executeListener(ctx, listener, event); err != nil {
			errors = append(errors, err)
			logger.Error("事件处理失败",
				zap.String("event", event.Name()),
				zap.String("listener_id", listener.ID),
				zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some listeners failed: %v", errors)
	}

	return nil
}

// PublishAsync 发布事件（异步）
func (bus *eventBusImpl) PublishAsync(ctx context.Context, event Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if !bus.started {
		return fmt.Errorf("event bus not started")
	}

	// 将事件放入通道
	select {
	case bus.eventChan <- eventTask{ctx: ctx, event: event}:
		logger.Debug("发布异步事件",
			zap.String("event", event.Name()))
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("event channel full, timeout after 5s")
	}
}

// executeListener 执行监听器
func (bus *eventBusImpl) executeListener(ctx context.Context, listener *EventListener, event Event) error {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 执行处理器
	if err := listener.Handler(timeoutCtx, event); err != nil {
		return fmt.Errorf("listener %s error: %w", listener.ID, err)
	}

	return nil
}

// Start 启动事件总线
func (bus *eventBusImpl) Start() error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.started {
		return fmt.Errorf("event bus already started")
	}

	// 启动工作协程
	for i := 0; i < bus.workerNum; i++ {
		bus.wg.Add(1)
		go bus.worker(i)
	}

	bus.started = true

	logger.Info("事件总线已启动",
		zap.Int("workers", bus.workerNum))

	return nil
}

// Stop 停止事件总线
func (bus *eventBusImpl) Stop() error {
	bus.mu.Lock()
	if !bus.started {
		bus.mu.Unlock()
		return fmt.Errorf("event bus not started")
	}
	bus.mu.Unlock()

	// 取消上下文
	bus.cancel()

	// 关闭事件通道
	close(bus.eventChan)

	// 等待所有工作协程退出
	bus.wg.Wait()

	bus.mu.Lock()
	bus.started = false
	bus.mu.Unlock()

	logger.Info("事件总线已停止")

	return nil
}

// worker 工作协程（消费者）
func (bus *eventBusImpl) worker(id int) {
	defer bus.wg.Done()

	logger.Debug("事件处理工作协程启动", zap.Int("worker_id", id))

	for {
		select {
		case <-bus.ctx.Done():
			logger.Debug("事件处理工作协程退出", zap.Int("worker_id", id))
			return

		case task, ok := <-bus.eventChan:
			if !ok {
				logger.Debug("事件通道已关闭，工作协程退出", zap.Int("worker_id", id))
				return
			}

			// 处理事件
			bus.handleEventTask(task)
		}
	}
}

// handleEventTask 处理事件任务
func (bus *eventBusImpl) handleEventTask(task eventTask) {
	bus.mu.RLock()
	listeners, exists := bus.listeners[task.event.Name()]
	bus.mu.RUnlock()

	if !exists || len(listeners) == 0 {
		logger.Debug("没有监听器订阅此事件",
			zap.String("event", task.event.Name()))
		return
	}

	logger.Debug("处理异步事件",
		zap.String("event", task.event.Name()),
		zap.Int("listeners", len(listeners)))

	// 执行所有监听器
	for _, listener := range listeners {
		// 为每个监听器创建独立的 goroutine（如果配置为异步）
		if listener.Async {
			go func(l *EventListener) {
				if err := bus.executeListener(task.ctx, l, task.event); err != nil {
					logger.Error("异步事件处理失败",
						zap.String("event", task.event.Name()),
						zap.String("listener_id", l.ID),
						zap.Error(err))
				}
			}(listener)
		} else {
			// 同步执行
			if err := bus.executeListener(task.ctx, listener, task.event); err != nil {
				logger.Error("事件处理失败",
					zap.String("event", task.event.Name()),
					zap.String("listener_id", listener.ID),
					zap.Error(err))
			}
		}
	}
}

// ========================================
// 全局事件总线实例
// ========================================

var (
	globalEventBus     EventBus
	globalEventBusOnce sync.Once
)

// GetGlobalEventBus 获取全局事件总线实例（单例模式）
func GetGlobalEventBus() EventBus {
	globalEventBusOnce.Do(func() {
		globalEventBus = NewEventBus(4) // 默认4个工作协程
	})
	return globalEventBus
}

// StartGlobalEventBus 启动全局事件总线
func StartGlobalEventBus() error {
	return GetGlobalEventBus().Start()
}

// StopGlobalEventBus 停止全局事件总线
func StopGlobalEventBus() error {
	return GetGlobalEventBus().Stop()
}

// Subscribe 全局订阅事件
func Subscribe(eventName string, listener *EventListener) error {
	return GetGlobalEventBus().Subscribe(eventName, listener)
}

// Unsubscribe 全局取消订阅
func Unsubscribe(eventName string, listenerID string) error {
	return GetGlobalEventBus().Unsubscribe(eventName, listenerID)
}

// Publish 全局发布同步事件
func Publish(ctx context.Context, event Event) error {
	return GetGlobalEventBus().Publish(ctx, event)
}

// PublishAsync 全局发布异步事件
func PublishAsync(ctx context.Context, event Event) error {
	return GetGlobalEventBus().PublishAsync(ctx, event)
}
