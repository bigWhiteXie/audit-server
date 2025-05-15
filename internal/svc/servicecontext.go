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
	"codexie.com/auditlog/internal/model"
	"codexie.com/auditlog/pkg/pipeline"
	"codexie.com/auditlog/pkg/plugin"
	_ "codexie.com/auditlog/pkg/plugin/exporter"
	_ "codexie.com/auditlog/pkg/plugin/filter"
	_ "codexie.com/auditlog/pkg/plugin/lifecycle"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	maxRowsPerTable = 3_000_000        // 单表最大行数
	checkInterval   = 30 * time.Minute // 检查间隔优化为30分钟
	delLogInterval  = 24 * time.Hour   // 删除日志间隔优化为1天
)

type ServiceContext struct {
	Config   config.Config
	DB       *gorm.DB
	Piplines []*pipeline.Pipeline
	Redis    *redis.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := &ServiceContext{
		Config: c,
	}

	ctx.initDB(c.MySQL)
	ctx.InitRedis(c.Redis)
	ctx.initTables()

	ctx.initPiplines(c.Pipelines)

	return ctx
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

func (s *ServiceContext) initTables() {
	entities := []model.Entity{
		&model.AuditLog{},
	}
	s.genTables(entities)

	go checkTableJob(s.DB, entities)
}

func (s *ServiceContext) genTables(entities []model.Entity) {
	schedulePos := &model.SchedulePos{}
	for _, entity := range entities {
		pos, err := schedulePos.GetSchedulePos(s.DB, entity.Name())
		if err != nil {
			panic(err)
		}
		s.Redis.Set(context.Background(), fmt.Sprintf("%s:%s", constant.SchedulePosKey, entity.Name()), pos.ScheduleEndPos, 0)
		s.DB.Table(fmt.Sprintf("%s_%d", entity.Name(), pos.Id)).AutoMigrate(pos, entity)
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

func checkTableJob(db *gorm.DB, entities []model.Entity) {
	checkSizeTicker := time.NewTicker(checkInterval)
	delLogTicker := time.NewTicker(delLogInterval)
	defer checkSizeTicker.Stop()
	defer delLogTicker.Stop()

	for {
		select {
		case <-checkSizeTicker.C:
			for _, entity := range entities {
				checkTableSize(db, entity)
			}
		case <-delLogTicker.C:
			for _, entity := range entities {
				delLogs(db, entity)
			}
		}
	}
}

func checkTableSize(db *gorm.DB, entity model.Entity) {
	// todo: 加分布式锁
	schedulePos := &model.SchedulePos{}
	// 获取当前表位置
	pos, err := schedulePos.GetSchedulePos(db, entity.Name())
	if err != nil {
		logx.Error("获取表位置失败", "entity", entity.Name(), "error", err)
		return
	}

	curTable := fmt.Sprintf("%s_%d", entity.Name(), pos.ScheduleEndPos)
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM " + curTable).Scan(&count).Error
	if err != nil {
		logx.Error("获取表行数失败", "entity", entity.Name(), "error", err)
		return
	}

	// 如果当前表行数超过300万，则创建新的表
	if count > maxRowsPerTable {
		pos.ScheduleEndPos++
		newTableName := fmt.Sprintf("%s_%d", entity.Name(), pos.ScheduleEndPos)
		db.Table(newTableName).AutoMigrate(entity)
		pos.Save(db, pos)
	}
}

func delLogs(db *gorm.DB, entity model.Entity) {
	pos := &model.SchedulePos{}
	pos.GetSchedulePos(db, entity.Name())
	//从pos.ScheduleBeginPos到pos.ScheduleEndPos中查询是否有超过6个月的日志记录
	for i := pos.ScheduleBeginPos; i <= pos.ScheduleEndPos; i++ {
		tableName := fmt.Sprintf("%s_%d", entity.Name(), i)
		res := db.Table(tableName).Where("created_at < ?", time.Now().AddDate(0, -6, 0)).Delete(&entity)
		if res.RowsAffected == 0 {
			break
		}
	}
}
