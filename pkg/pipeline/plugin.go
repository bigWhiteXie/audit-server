package pipeline

import (
	"context"

	"codexie.com/auditlog/pkg/plugins"
)

// 插件注册方法
func (p *Pipeline[T]) RegisterExporter(exporter plugins.Exporter[T]) {
	p.plugins.exporter[exporter.Name()] = exporter
}

func (p *Pipeline[T]) RegisterFilter(filter plugins.Filter[T]) {
	p.plugins.filters = append(p.plugins.filters, filter)
}

func (p *Pipeline[T]) RegisterLifecycleHook(hook plugins.LifecycleHook[T]) {
	p.plugins.lifecycles = append(p.plugins.lifecycles, hook)
}

// 默认插件实现需要修改为泛型实现
type NoopLifecycleHook[T any] struct{}

func (h *NoopLifecycleHook[T]) Name() string { return "noop-lifecycle" }

func (h *NoopLifecycleHook[T]) BeforeExport(ctx context.Context, batch []T) context.Context {
	return ctx
}

func (h *NoopLifecycleHook[T]) OnError(ctx context.Context, err error, batch []T) {
	// 无操作
}
