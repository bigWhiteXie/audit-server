package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/internal/constant"
	"codexie.com/auditlog/internal/job"
	"codexie.com/auditlog/internal/model"
	"codexie.com/auditlog/pkg/pipeline"
	"codexie.com/auditlog/pkg/plugin"
	_ "codexie.com/auditlog/pkg/plugin/exporter"
	_ "codexie.com/auditlog/pkg/plugin/filter"
	_ "codexie.com/auditlog/pkg/plugin/lifecycle"
	"codexie.com/auditlog/pkg/scheduler"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxRowsPerTable = 3_000_000        // 单表最大行数
	checkInterval   = 30 * time.Minute // 检查间隔优化为30分钟
	delLogInterval  = 24 * time.Hour   // 删除日志间隔优化为1天
)

type ServiceContext struct {
	Config    config.Config
	DB        *gorm.DB
	Piplines  []*pipeline.Pipeline
	Redis     *redis.Client
	Scheduler *scheduler.Scheduler
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := &ServiceContext{
		Config: c,
	}

	ctx.initDB(c.MySQL)
	ctx.InitRedis(c.Redis)
	ctx.initTables()

	ctx.initPiplines(c.Pipelines)
	ctx.initScheduler(c.Scheduler)

	return ctx
}

func (s *ServiceContext) ReleaseAll() {
	for _, p := range s.Piplines {
		p.Close()
	}

	s.Scheduler.Stop()
}

// 设置数据库
func (s *ServiceContext) initDB(mysqlConf config.MySQLConf) {
	datasource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		mysqlConf.User,
		mysqlConf.Password,
		mysqlConf.Host,
		mysqlConf.Port,
		mysqlConf.Database,
		"utf8",
		true,
		"Local")

	gormDB, err := gorm.Open(mysql.Open(datasource))
	if err != nil {
		panic(err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxOpenConns(mysqlConf.MaxOpenConns)
	sqlDB.SetMaxIdleConns(mysqlConf.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(mysqlConf.ConnMaxLifetime) * time.Second)

	s.DB = gormDB
}

// 设置piplines
func (s *ServiceContext) initPiplines(piplineConfigs []config.PiplineConfig) {
	piplines := make([]*pipeline.Pipeline, 0, len(piplineConfigs))
	logx.Info("初始化piplines")
	for _, piplineConfig := range piplineConfigs {
		jsonStr, _ := json.Marshal(piplineConfig)
		logx.Infof("init piplines: %s", string(jsonStr))

		p := pipeline.New(piplineConfig)
		for _, expConf := range piplineConfig.Plugins.Exporters {
			conf := make(map[string]any)
			for k, v := range expConf.Config {
				if strings.HasPrefix(v, "#svc.") {
					fieldName := strings.TrimPrefix(v, "#svc.")
					conf[k] = s.getFieldVal(fieldName)
				} else {
					conf[k] = v
				}
			}
			exporter := plugin.GetExporter(expConf.Name, conf)
			p.RegisterExporter(exporter)
		}

		for _, filterConf := range piplineConfig.Plugins.Filters {
			conf := make(map[string]any)
			for k, v := range filterConf.Config {
				if strings.HasPrefix(v, "#svc.") {
					fieldName := strings.TrimPrefix(v, "#svc.")
					conf[k] = s.getFieldVal(fieldName)
				} else {
					conf[k] = v
				}
			}
			filter := plugin.GetFilter(filterConf.Name, conf)
			p.RegisterFilter(filter)
		}

		for _, lifecycleConf := range piplineConfig.Plugins.Lifecycles {
			conf := make(map[string]any)
			for k, v := range lifecycleConf.Config {
				if strings.HasPrefix(v, "#svc.") {
					fieldName := strings.TrimPrefix(v, "#svc.")
					conf[k] = s.getFieldVal(fieldName)
				} else {
					conf[k] = v
				}
			}
			lifecycle := plugin.GetLifecycle(lifecycleConf.Name, conf)
			p.RegisterLifecycleHook(lifecycle)
		}

		piplines = append(piplines, p)
	}

	s.Piplines = piplines
}

func (s *ServiceContext) InitRedis(redisConf config.RedisConf) {
	myRedis := redis.NewClient(&redis.Options{
		Addr:     redisConf.Host,
		Password: redisConf.Pass,
		DB:       0,
	})

	_, err := myRedis.Ping(context.Background()).Result()
	if err != nil {
		panic("redis connect failed: " + err.Error())
	}
	s.Redis = myRedis
}

func (s *ServiceContext) initScheduler(conf scheduler.ScheduleConfig) {
	s.Scheduler = scheduler.NewScheduler(s.DB, conf)
	scheduleJob := job.NewScheduleJob(s.DB)
	s.Scheduler.RegisterTask(scheduleJob)
}

func (s *ServiceContext) initTables() {
	entities := []model.Entity{
		&model.AuditLog{},
	}
	s.genTables(entities)
}

func (s *ServiceContext) genTables(entities []model.Entity) {
	schedulePos := &model.SchedulePos{}
	s.DB.AutoMigrate(schedulePos)
	s.DB.AutoMigrate(&scheduler.ScheduleTask{})

	//创建实体对象表
	for _, entity := range entities {
		pos, err := schedulePos.GetSchedulePos(s.DB, entity.Name())
		if err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		}
		if err == gorm.ErrRecordNotFound {
			pos = &model.SchedulePos{
				Name:             entity.Name(),
				ScheduleBeginPos: 1,
				ScheduleEndPos:   1,
			}
			if err := s.DB.Model(pos).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoNothing: true,
			}).Create(pos).Error; err != nil {
				panic(err)
			}
		}
		s.Redis.Set(context.Background(), fmt.Sprintf("%s:%s", constant.SchedulePosKey, entity.Name()), pos.ScheduleEndPos, 0)
		s.DB.Table(fmt.Sprintf("%s_%d", entity.Name(), pos.ScheduleEndPos)).AutoMigrate(entity)
	}
}

func (s *ServiceContext) getFieldVal(fieldName string) any {
	sValue := reflect.ValueOf(s).Elem()
	field := sValue.FieldByName(fieldName)

	if field.IsValid() && field.CanInterface() {
		return field.Interface()
	}

	logx.Errorf("ServiceContext字段不存在: %s", fieldName)
	return nil
}
