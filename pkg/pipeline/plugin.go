package pipeline

import (
	"context"

	"codexie.com/auditlog/pkg/plugin"
)

// 插件注册方法
func (p *Pipeline) RegisterExporter(exporter plugin.Exporter) {
	p.plugins.exporter[exporter.Name()] = exporter
}

func (p *Pipeline) RegisterFilter(filter plugin.Filter) {
	p.plugins.filters = append(p.plugins.filters, filter)
}

func (p *Pipeline) RegisterLifecycleHook(hook plugin.LifecycleHook) {
	p.plugins.lifecycles = append(p.plugins.lifecycles, hook)
}

// 默认插件实现需要修改为泛型实现
type NoopLifecycleHook struct{}

func (h *NoopLifecycleHook) Name() string { return "noop-lifecycle" }

func (h *NoopLifecycleHook) BeforeExport(ctx context.Context, batch []interface{}) context.Context {
	return ctx
}

func (h *NoopLifecycleHook) OnError(ctx context.Context, err error, batch []interface{}) {
	// 无操作
}
