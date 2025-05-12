package plugins

import (
	"context"
)

// Plugin 所有插件的基础接口
type Plugin interface {
	Name() string
}

// Exporter 数据导出插件接口，支持泛型
type Exporter[T any] interface {
	Plugin
	Export(ctx context.Context, data []T) error
}

// Filter 数据过滤插件接口，支持泛型
type Filter[T any] interface {
	Plugin
	Filter(data T) bool
}

// LifecycleHook 生命周期钩子插件接口，支持泛型
type LifecycleHook[T any] interface {
	Plugin
	BeforeExport(ctx context.Context, batch []T) context.Context
	OnError(ctx context.Context, err error, batch []T)
}
