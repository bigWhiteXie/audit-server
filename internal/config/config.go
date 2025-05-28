package config

import (
	"codexie.com/auditlog/pkg/scheduler"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	MySQL     MySQLConf
	Redis     RedisConf
	Pipelines []PiplineConfig
	Scheduler scheduler.ScheduleConfig
}
