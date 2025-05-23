package config

// PluginItem 表示单个插件配置项
type PluginItem struct {
	Name   string                 `yaml:"name" json:"name"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// PluginsConfig 表示所有插件的配置
type PluginsConfig struct {
	Exporters  []PluginItem `yaml:"exporters" json:"exporters"`
	Filters    []PluginItem `yaml:"filters" json:"filters"`
	Lifecycles []PluginItem `yaml:"lifecycles" json:"lifecycles"`
}
