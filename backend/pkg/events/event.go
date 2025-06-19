package events

import (
	"context"
	"time"
)

// Event 事件接口
type Event interface {
	GetID() string
	GetType() string
	GetSource() string
	GetTimestamp() time.Time
	GetData() interface{}
	GetMetadata() map[string]interface{}
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (e *BaseEvent) GetID() string                       { return e.ID }
func (e *BaseEvent) GetType() string                     { return e.Type }
func (e *BaseEvent) GetSource() string                   { return e.Source }
func (e *BaseEvent) GetTimestamp() time.Time             { return e.Timestamp }
func (e *BaseEvent) GetData() interface{}                { return e.Data }
func (e *BaseEvent) GetMetadata() map[string]interface{} { return e.Metadata }

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	GetPriority() int
	CanHandle(event Event) bool
}

// HandlerFunc 函数式事件处理器
type HandlerFunc func(ctx context.Context, event Event) error

func (f HandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

func (f HandlerFunc) GetPriority() int {
	return 0 // 默认优先级
}

func (f HandlerFunc) CanHandle(event Event) bool {
	return true // 默认可以处理所有事件
}
