package exporter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/pkg/plugin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLExporter struct {
	config.MySQLConf
	db *gorm.DB
}

func NewExporter(cfgMap map[string]string) *MySQLExporter {
	port, err := strconv.Atoi(cfgMap["port"])
	maxOpenConns, err := strconv.Atoi(cfgMap["max_open_conns"])
	maxIdleConns, err := strconv.Atoi(cfgMap["max_idle_conns"])
	connMaxLifetime, err := strconv.Atoi(cfgMap["conn_max_lifetime"])
	if err != nil {
		panic(err)
	}
	cfg := config.MySQLConf{}
	cfg.Host = cfgMap["host"]
	cfg.Port = int64(port)
	cfg.User = cfgMap["user"]
	cfg.Password = cfgMap["password"]
	cfg.Database = cfgMap["database"]
	cfg.MaxOpenConns = maxOpenConns
	cfg.MaxIdleConns = maxIdleConns
	cfg.ConnMaxLifetime = connMaxLifetime
	db := initDB(cfg)
	return &MySQLExporter{
		MySQLConf: cfg,
		db:        db,
	}
}

func (e *MySQLExporter) Export(ctx context.Context, data []interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// 类型转换
	entities := make([]plugin.Entity, 0, len(data))
	for _, item := range data {
		entity, ok := item.(plugin.Entity)
		if !ok {
			return fmt.Errorf("invalid data type: %T does not implement Entity interface", item)
		}
		entities = append(entities, entity)
	}

	// 批量插入（使用INSERT IGNORE）
	tx := e.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return fmt.Errorf("transaction error: %w", err)
	}

	batchSize := 1000
	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]
		// 使用Clause.OnConflict实现INSERT IGNORE
		if err := batch[0].SaveBatch(ctx, tx, batch); err != nil {
			tx.Rollback()
			return fmt.Errorf("batch insert failed: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit transaction failed: %w", err)
	}

	return nil
}

func (e *MySQLExporter) Name() string {
	return "mysql"
}

func (e *MySQLExporter) Close() error {
	return nil
}

func init() {
	plugin.RegisterExporterFactory("mysql", func(config map[string]string) plugin.Exporter {
		return NewExporter(config)
	})
}

func initDB(mysqlConf config.MySQLConf) *gorm.DB {
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

	return gormDB
}
