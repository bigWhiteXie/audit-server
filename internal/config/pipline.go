package config

import "time"

type PiplineConfig struct {
	Name             string        `json:"" yaml:"Name"`
	BatchSize        int           `json:"" yaml:"BatchSize"`
	BatchTimeout     time.Duration `json:"" yaml:"BatchTimeout"`
	StorageDir       string        `json:"" yaml:"StorageDir"`
	MetricsPrefix    string        `json:"" yaml:"MetricsPrefix"`
	RecoveryInterval time.Duration `json:"" yaml:"RecoveryInterval"`
	plugins          PluginsConfig
}

// 设置默认配置值
func (c PiplineConfig) SetDefaults() {
	if c.RecoveryInterval <= 0 {
		c.RecoveryInterval = 30 * time.Second
	}
}
