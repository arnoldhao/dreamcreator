package events

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"dreamcreator/backend/pkg/logger"

	"go.uber.org/zap"
)

// EventBus 事件总线接口
type EventBus interface {
	// 发布事件
	Publish(ctx context.Context, event Event) error
	PublishAsync(ctx context.Context, event Event)

	// 订阅事件
	Subscribe(eventType string, handler EventHandler) error
	SubscribeWithFilter(eventType string, handler EventHandler, filter EventFilter) error

	// 取消订阅
	Unsubscribe(eventType string, handler EventHandler) error

	// 生命周期管理
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// 健康检查
	HealthCheck() error
}

// EventFilter 事件过滤器
type EventFilter func(event Event) bool

// handlerWrapper 处理器包装
type handlerWrapper struct {
	handler EventHandler
	filter  EventFilter
}

// eventBus 事件总线实现
type eventBus struct {
	handlers map[string][]*handlerWrapper
	mu       sync.RWMutex
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc

	// 配置选项
	options *EventBusOptions
}

// EventBusOptions 事件总线配置
type EventBusOptions struct {
	MaxWorkers    int           `json:"max_workers"`
	BufferSize    int           `json:"buffer_size"`
	Timeout       time.Duration `json:"timeout"`
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay"`
	EnableMetrics bool          `json:"enable_metrics"`
	EnablePersist bool          `json:"enable_persist"`
}

// DefaultEventBusOptions 默认配置
func DefaultEventBusOptions() *EventBusOptions {
	return &EventBusOptions{
		MaxWorkers:    10,
		BufferSize:    1000,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
		EnableMetrics: true,
		EnablePersist: false,
	}
}

// NewEventBus 创建新的事件总线
func NewEventBus(options *EventBusOptions) EventBus {
	if options == nil {
		options = DefaultEventBusOptions()
	}

	return &eventBus{
		handlers: make(map[string][]*handlerWrapper),
		options:  options,
	}
}

// Start 启动事件总线
func (eb *eventBus) Start(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.running {
		return fmt.Errorf("event bus is already running")
	}

	eb.ctx, eb.cancel = context.WithCancel(ctx)
	eb.running = true

	logger.Info("EventBus started")
	return nil
}

// Stop 停止事件总线
func (eb *eventBus) Stop(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if !eb.running {
		return fmt.Errorf("event bus is not running")
	}

	eb.cancel()
	eb.running = false

	logger.Info("EventBus stopped")
	return nil
}

// Subscribe 订阅事件
func (eb *eventBus) Subscribe(eventType string, handler EventHandler) error {
	return eb.SubscribeWithFilter(eventType, handler, nil)
}

// SubscribeWithFilter 带过滤器的订阅
func (eb *eventBus) SubscribeWithFilter(eventType string, handler EventHandler, filter EventFilter) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	wrapper := &handlerWrapper{
		handler: handler,
		filter:  filter,
	}

	eb.handlers[eventType] = append(eb.handlers[eventType], wrapper)

	// 按优先级排序
	sort.Slice(eb.handlers[eventType], func(i, j int) bool {
		return eb.handlers[eventType][i].handler.GetPriority() > eb.handlers[eventType][j].handler.GetPriority()
	})

	return nil
}

// Unsubscribe 取消订阅
func (eb *eventBus) Unsubscribe(eventType string, handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers, exists := eb.handlers[eventType]
	if !exists {
		return fmt.Errorf("no handlers found for event type: %s", eventType)
	}

	// 查找并移除指定的处理器
	for i, wrapper := range handlers {
		if wrapper.handler == handler {
			// 移除该处理器
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)

			// 如果没有处理器了，删除该事件类型
			if len(eb.handlers[eventType]) == 0 {
				delete(eb.handlers, eventType)
			}

			return nil
		}
	}

	return fmt.Errorf("handler not found for event type: %s", eventType)
}

// Publish 同步发布事件
func (eb *eventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.handlers[event.GetType()]
	eb.mu.RUnlock()

	for _, wrapper := range handlers {
		if !wrapper.handler.CanHandle(event) {
			continue
		}

		if wrapper.filter != nil && !wrapper.filter(event) {
			continue
		}

		if err := eb.handleWithRetry(ctx, wrapper.handler, event); err != nil {
			logger.Error("Failed to handle event", zap.String("event_id", event.GetID()), zap.Error(err))
			// 继续处理其他处理器，不因一个失败而中断
		}
	}

	return nil
}

// PublishAsync 异步发布事件
func (eb *eventBus) PublishAsync(ctx context.Context, event Event) {
	go func() {
		if err := eb.Publish(ctx, event); err != nil {
			logger.Error("Async event publish failed", zap.String("event_id", event.GetID()), zap.Error(err))
		}
	}()
}

// handleWithRetry 带重试的事件处理
func (eb *eventBus) handleWithRetry(ctx context.Context, handler EventHandler, event Event) error {
	var lastErr error

	for i := 0; i <= eb.options.RetryAttempts; i++ {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, eb.options.Timeout)
		err := handler.Handle(ctxWithTimeout, event)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err
		if i < eb.options.RetryAttempts {
			time.Sleep(eb.options.RetryDelay)
		}
	}

	return lastErr
}

// HealthCheck 健康检查
func (eb *eventBus) HealthCheck() error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if !eb.running {
		return fmt.Errorf("event bus is not running")
	}

	return nil
}
