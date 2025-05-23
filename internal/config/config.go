package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	MySQLConf MySQLConf
	// RedisConf RedisConf
	// KafkaConf KafkaConf

	Plugins PluginsConfig `yaml:"plugins" json:"plugins"`
}
