package config

// PluginItem 表示单个插件配置项
type PluginItem struct {
	Name   string            `yaml:",optional" json:"name"`
	Config map[string]string `yaml:",optional" json:"config"`
}

// PluginsConfig 表示所有插件的配置
type PluginsConfig struct {
	Exporters  []PluginItem `yaml:"exporters" json:"exporters,optional"`
	Filters    []PluginItem `yaml:"filters" json:"filters,optional"`
	Lifecycles []PluginItem `yaml:"lifecycles" json:"lifecycles,optional"`
}
