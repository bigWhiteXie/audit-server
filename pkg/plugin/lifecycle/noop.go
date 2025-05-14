package lifecycle

import (
	"context"
)

// NoopHook 无操作生命周期钩子
type NoopHook struct{}

// NewNoopHook 创建新的无操作生命周期钩子
func NewNoopHook() *NoopHook {
	return &NoopHook{}
}

// Name 返回插件名称
func (h *NoopHook) Name() string { return "noop-lifecycle" }

// BeforeExport 导出前钩子
func (h *NoopHook) BeforeExport(ctx context.Context, batch []interface{}) context.Context {
	return ctx
}

// OnError 错误处理钩子
func (h *NoopHook) OnError(ctx context.Context, err error, batch []interface{}) {
	// 无操作
}

// 确保NoopHook实现了LifecycleHook接口
var _ plugins.LifecycleHook = (*NoopHook)(nil)
