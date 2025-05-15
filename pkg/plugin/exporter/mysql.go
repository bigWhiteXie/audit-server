package exporter

import (
	"context"
	"fmt"
	"time"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/internal/model"
	"codexie.com/auditlog/pkg/plugin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLExporter struct {
	db *gorm.DB
}

func NewExporter(cfgMap map[string]any) *MySQLExporter {
	db := cfgMap["db"].(*gorm.DB)
	return &MySQLExporter{
		db: db,
	}
}

func (e *MySQLExporter) Export(ctx context.Context, data []interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// 类型转换
	entities := make([]model.Entity, 0, len(data))
	for _, item := range data {
		entity, ok := item.(model.Entity)
		if !ok {
			return fmt.Errorf("invalid data type: %T does not implement Entity interface", item)
		}
		entities = append(entities, entity)
	}

	err := e.db.Transaction(func(tx *gorm.DB) error {
		batchSize := 1000
		for i := 0; i < len(entities); i += batchSize {
			end := i + batchSize
			if end > len(entities) {
				end = len(entities)
			}

			batch := entities[i:end]
			// 使用Clause.OnConflict实现INSERT IGNORE
			if err := batch[0].SaveBatch(ctx, tx, batch); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (e *MySQLExporter) Name() string {
	return "mysql"
}

func (e *MySQLExporter) Close() error {
	return nil
}

func init() {
	plugin.RegisterExporterFactory("mysql", func(config map[string]any) plugin.Exporter {
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
