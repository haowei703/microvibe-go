package event

import (
	"context"
	"time"
)

// Event 事件接口 - 所有事件必须实现此接口
type Event interface {
	// Name 返回事件名称
	Name() string

	// Timestamp 返回事件发生时间
	Timestamp() time.Time

	// Metadata 返回事件元数据
	Metadata() map[string]interface{}
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	EventName     string                 `json:"event_name"`
	EventTime     time.Time              `json:"event_time"`
	EventMetadata map[string]interface{} `json:"metadata"`
}

// Name 实现 Event 接口
func (e *BaseEvent) Name() string {
	return e.EventName
}

// Timestamp 实现 Event 接口
func (e *BaseEvent) Timestamp() time.Time {
	return e.EventTime
}

// Metadata 实现 Event 接口
func (e *BaseEvent) Metadata() map[string]interface{} {
	if e.EventMetadata == nil {
		e.EventMetadata = make(map[string]interface{})
	}
	return e.EventMetadata
}

// NewBaseEvent 创建基础事件
func NewBaseEvent(name string) *BaseEvent {
	return &BaseEvent{
		EventName:     name,
		EventTime:     time.Now(),
		EventMetadata: make(map[string]interface{}),
	}
}

// EventHandler 事件处理器函数类型
type EventHandler func(ctx context.Context, event Event) error

// EventListener 事件监听器
type EventListener struct {
	ID      string       // 监听器ID
	Handler EventHandler // 处理函数
	Async   bool         // 是否异步处理
}

// NewEventListener 创建事件监听器
func NewEventListener(id string, handler EventHandler, async bool) *EventListener {
	return &EventListener{
		ID:      id,
		Handler: handler,
		Async:   async,
	}
}

// EventFilter 事件过滤器函数类型
type EventFilter func(event Event) bool

// Priority 事件优先级
type Priority int

const (
	PriorityLow    Priority = 0  // 低优先级
	PriorityNormal Priority = 5  // 普通优先级
	PriorityHigh   Priority = 10 // 高优先级
)

// EventOptions 事件发布选项
type EventOptions struct {
	Priority Priority               // 事件优先级
	Metadata map[string]interface{} // 额外的元数据
	Timeout  time.Duration          // 处理超时时间
}

// DefaultEventOptions 默认事件选项
func DefaultEventOptions() *EventOptions {
	return &EventOptions{
		Priority: PriorityNormal,
		Metadata: make(map[string]interface{}),
		Timeout:  30 * time.Second,
	}
}
