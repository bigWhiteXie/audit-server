package config

import "time"

type PiplineConfig struct {
	Name             string
	BatchSize        int
	BatchTimeout     time.Duration
	StorageDir       string
	MetricsPrefix    string
	RecoveryInterval time.Duration // 恢复检查间隔
}

// 设置默认配置值
func (c PiplineConfig) SetDefaults() {
	if c.RecoveryInterval <= 0 {
		c.RecoveryInterval = 30 * time.Second
	}
}
