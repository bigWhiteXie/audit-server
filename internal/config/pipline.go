package config

type PiplineConfig struct {
	Name             string        `json:",optional" yaml:"Name"`
	BatchSize        int           `json:",optional" yaml:"BatchSize"`
	BatchTimeout     int           `json:",optional" yaml:"BatchTimeout"`
	StorageDir       string        `json:",optional" yaml:"StorageDir"`
	MetricsPrefix    string        `json:",optional" yaml:"MetricsPrefix"`
	RecoveryInterval int           `json:",optional" yaml:"RecoveryInterval"`
	Plugins          PluginsConfig `json:",optional" yaml:"Plugins"`
}

// 设置默认配置值
func (c PiplineConfig) SetDefaults() {
	if c.RecoveryInterval <= 0 {
		c.RecoveryInterval = 30
	}
}
