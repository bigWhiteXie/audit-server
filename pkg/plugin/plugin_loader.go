package plugin

import (
	"fmt"

	"codexie.com/auditlog/internal/config"
)

// LoadedPlugins 用于存储已加载的插件实例
// 可根据实际需要扩展类型
// 这里只存储any类型，实际使用时可断言为具体接口

type LoadedPlugins struct {
	Exporters  []any
	Filters    []any
	Lifecycles []any
}

// LoadPlugins 根据PluginsConfig加载并注册插件
func LoadPlugins(cfg config.PluginsConfig) (*LoadedPlugins, error) {
	loaded := &LoadedPlugins{}
	// 加载exporters
	for _, item := range cfg.Exporters {
		inst := GetExporter(item.Name, item.Config)
		if inst == nil {
			return nil, fmt.Errorf("exporter plugin '%s' not found or failed to initialize", item.Name)
		}
		loaded.Exporters = append(loaded.Exporters, inst)
	}
	// 加载filters
	for _, item := range cfg.Filters {
		inst := GetFilter(item.Name, item.Config)
		if inst == nil {
			return nil, fmt.Errorf("filter plugin '%s' not found or failed to initialize", item.Name)
		}
		loaded.Filters = append(loaded.Filters, inst)
	}
	// 加载lifecycles
	for _, item := range cfg.Lifecycles {
		inst := GetLifecycle(item.Name, item.Config)
		if inst == nil {
			return nil, fmt.Errorf("lifecycle plugin '%s' not found or failed to initialize", item.Name)
		}
		loaded.Lifecycles = append(loaded.Lifecycles, inst)
	}
	return loaded, nil
}
