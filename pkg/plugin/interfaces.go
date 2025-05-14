package plugin

import (
	"context"

	"gorm.io/gorm"
)

// Plugin 所有插件的基础接口
type Plugin interface {
	Name() string
}

// Exporter 数据导出插件接口，支持泛型
type Exporter interface {
	Plugin
	Export(ctx context.Context, data []interface{}) error
}

// Filter 数据过滤插件接口，支持泛型
type Filter interface {
	Plugin
	Filter(data interface{}) bool
}

// LifecycleHook 生命周期钩子插件接口，支持泛型
type LifecycleHook interface {
	Plugin
	BeforeExport(ctx context.Context, batch []interface{}) context.Context
	OnError(ctx context.Context, err error, batch []interface{})
}

type Entity interface {
	Name() string
	TableName() string
	SaveBatch(ctx context.Context, db *gorm.DB, data []Entity) error
}
