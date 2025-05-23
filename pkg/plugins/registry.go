package plugins

import (
	"sync"
)

var (
	exporterRegistry  = make(map[string]func(config map[string]any) Exporter[any])
	filterRegistry    = make(map[string]func(config map[string]any) Filter[any])
	lifecycleRegistry = make(map[string]func(config map[string]any) LifecycleHook[any])
	mutex             sync.RWMutex
)

// RegisterExporter 注册Exporter插件工厂
func RegisterExporter(name string, factory func(config map[string]any) Exporter[any]) {
	mutex.Lock()
	defer mutex.Unlock()
	exporterRegistry[name] = factory
}

// GetExporter 根据名称和配置获取Exporter实例
func GetExporter(name string, config map[string]any) Exporter[any] {
	mutex.RLock()
	defer mutex.RUnlock()
	if factory, ok := exporterRegistry[name]; ok {
		return factory(config)
	}
	return nil
}

// RegisterFilter 注册Filter插件工厂
func RegisterFilter(name string, factory func(config map[string]any) Filter[any]) {
	mutex.Lock()
	defer mutex.Unlock()
	filterRegistry[name] = factory
}

// GetFilter 根据名称和配置获取Filter实例
func GetFilter(name string, config map[string]any) Filter[any] {
	mutex.RLock()
	defer mutex.RUnlock()
	if factory, ok := filterRegistry[name]; ok {
		return factory(config)
	}
	return nil
}

// RegisterLifecycle 注册Lifecycle插件工厂
func RegisterLifecycle(name string, factory func(config map[string]any) LifecycleHook[any]) {
	mutex.Lock()
	defer mutex.Unlock()
	lifecycleRegistry[name] = factory
}

// GetLifecycle 根据名称和配置获取Lifecycle实例
func GetLifecycle(name string, config map[string]any) LifecycleHook[any] {
	mutex.RLock()
	defer mutex.RUnlock()
	if factory, ok := lifecycleRegistry[name]; ok {
		return factory(config)
	}
	return nil
}
