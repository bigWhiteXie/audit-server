package lifecycle

import (
	"context"

	"codexie.com/auditlog/pkg/plugins"
)

// NoopHook 无操作生命周期钩子
type NoopHook[T any] struct{}

// NewNoopHook 创建新的无操作生命周期钩子
func NewNoopHook[T any]() *NoopHook[T] {
	return &NoopHook[T]{}
}

// Name 返回插件名称
func (h *NoopHook[T]) Name() string { return "noop-lifecycle" }

// BeforeExport 导出前钩子
func (h *NoopHook[T]) BeforeExport(ctx context.Context, batch []T) context.Context {
	return ctx
}

// OnError 错误处理钩子
func (h *NoopHook[T]) OnError(ctx context.Context, err error, batch []T) {
	// 无操作
}

// 确保NoopHook实现了LifecycleHook接口
var _ plugins.LifecycleHook[any] = (*NoopHook[any])(nil)
