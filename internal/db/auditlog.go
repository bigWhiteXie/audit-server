package db

import (
	"fmt"
	"sync"
	"time"

	"codexie.com/auditlog/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 分表配置（每月一张表）
const (
	TablePrefix    = "audit_log_"
	DateLayout     = "200601"
	MaxTableMonths = 6 // 保留最近6个月分表
)

var (
	curTable   = ""
	tableMutex sync.Mutex
	db         *gorm.DB
)

func InitGormDB(mysqlConf config.MySQLConf) {
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

	//判断表是否存在
	if !gormDB.Migrator().HasTable(&AuditLog{}) {
		gormDB.Migrator().CreateTable(&AuditLog{})
	}

	db = gormDB
}

// 审计日志实体（对应分表结构 audit_log_YYYYMM）
type AuditLog struct {
	ID           string    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`       // 自增主键
	LogId        string    `gorm:"column:log_id;index;size:64" json:"log_id"`          // 日志ID
	TenantID     string    `gorm:"column:tenant_id;index;size:32" json:"tenant_id"`    // 租户ID
	UserID       string    `gorm:"column:user_id;index;size:64" json:"user_id"`        // 用户ID
	Username     string    `gorm:"column:username;size:128" json:"username"`           // 用户名
	Action       string    `gorm:"column:action;index;size:64" json:"action"`          // 操作类型
	ResourceType string    `gorm:"column:resource_type;size:64" json:"resource_type"`  // 资源类型
	ResourceID   string    `gorm:"column:resource_id;size:128" json:"resource_id"`     // 资源ID
	ResourceName string    `gorm:"column:resource_name;size:255" json:"resource_name"` // 资源名称
	Result       string    `gorm:"column:result;index;size:16" json:"result"`          // 操作结果
	Message      string    `gorm:"column:message;type:text" json:"message"`            // 详细信息
	TimeStamp    int64     `gorm:"column:timestamp;index" json:"timestamp"`            // 日志时间戳
	ClientIP     string    `gorm:"column:client_ip;size:64" json:"client_ip"`          // 客户端IP
	FromService  string    `gorm:"column:from_service;size:64" json:"from_service"`    // 来源服务
	TraceID      string    `gorm:"column:trace_id;size:128" json:"trace_id"`           // 链路追踪ID
	CreatedAt    time.Time `gorm:"column:created_at;index" json:"created_at"`          // 创建时间
}

// TableName 设置表名（实际表名会根据分表规则动态生成）
func (AuditLog) TableName() string {
	return TablePrefix + time.Now().Format(DateLayout)

}

func (log *AuditLog) BeforeCreate(tx *gorm.DB) error {
	// 生成当前应属表名
	targetTable := log.TableName()

	// 快速路径：无需切表
	if targetTable == curTable {
		return nil
	}

	// 慢路径：需要切表
	tableMutex.Lock()
	defer tableMutex.Unlock()

	// 双重检查锁模式
	if targetTable != curTable {
		// 创建新表（自动迁移）
		if err := tx.Table(targetTable).AutoMigrate(&AuditLog{}); err != nil {
			return err
		}
		// 更新当前表缓存
		curTable = targetTable
	}
	return nil
}

// 实现分表逻辑的接口（根据created_at字段自动路由）
func (log *AuditLog) TableNameWithSuffix() string {
	return TablePrefix + log.CreatedAt.Format(DateLayout)
}
