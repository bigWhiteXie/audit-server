package plugin

import (
	"sync"
)

var (
	exporterFactoryRegistry  = make(map[string]func(config map[string]any) Exporter)
	filterFactoryRegistry    = make(map[string]func(config map[string]any) Filter)
	lifecycleFactoryRegistry = make(map[string]func(config map[string]any) LifecycleHook)
	mutex                    sync.RWMutex
)

// RegisterExporter 注册Exporter插件工厂
func RegisterExporterFactory(name string, factory func(config map[string]any) Exporter) {
	mutex.Lock()
	defer mutex.Unlock()
	exporterFactoryRegistry[name] = factory
}

// GetExporter 根据名称和配置获取Exporter实例
func GetExporter(name string, config map[string]any) Exporter {
	mutex.RLock()
	defer mutex.RUnlock()
	if factory, ok := exporterFactoryRegistry[name]; ok {
		return factory(config)
	}
	return nil
}

// RegisterFilter 注册Filter插件工厂
func RegisterFilterFactory(name string, factory func(config map[string]any) Filter) {
	mutex.Lock()
	defer mutex.Unlock()
	filterFactoryRegistry[name] = factory
}

// GetFilter 根据名称和配置获取Filter实例
func GetFilter(name string, config map[string]any) Filter {
	mutex.RLock()
	defer mutex.RUnlock()
	if factory, ok := filterFactoryRegistry[name]; ok {
		return factory(config)
	}
	return nil
}

// RegisterLifecycle 注册Lifecycle插件工厂
func RegisterLifecycleFactory(name string, factory func(config map[string]any) LifecycleHook) {
	mutex.Lock()
	defer mutex.Unlock()
	lifecycleFactoryRegistry[name] = factory
}

// GetLifecycle 根据名称和配置获取Lifecycle实例
func GetLifecycle(name string, config map[string]any) LifecycleHook {
	mutex.RLock()
	defer mutex.RUnlock()
	if factory, ok := lifecycleFactoryRegistry[name]; ok {
		return factory(config)
	}
	return nil
}
