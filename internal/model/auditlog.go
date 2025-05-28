package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 分表配置（每月一张表）
const (
	AuditLogName = "audit_log"
)

// 审计日志实体（对应分表结构 audit_log_YYYYMM）
type AuditLog struct {
	Id           int       `gorm:"primaryKey,autoIncrement" json:"id"`                       // 自增主键
	LogId        string    `gorm:"column:log_id;index;" json:"log_id"`                       // 日志ID
	TenantID     string    `gorm:"column:tenant_id;index;" json:"tenant_id"`                 // 租户ID
	UserID       string    `gorm:"column:user_id;index;" json:"user_id"`                     // 用户ID
	Username     string    `gorm:"column:username;" json:"username"`                         // 用户名
	Action       string    `gorm:"column:action;index;" json:"action"`                       // 操作类型
	ResourceType string    `gorm:"column:resource_type;" json:"resource_type"`               // 资源类型
	ResourceID   string    `gorm:"column:resource_id;" json:"resource_id"`                   // 资源ID
	ResourceName string    `gorm:"column:resource_name;" json:"resource_name"`               // 资源名称
	Result       string    `gorm:"column:result;index;" json:"result"`                       // 操作结果
	Message      string    `gorm:"column:message;type:text" json:"message"`                  // 详细信息
	TimeStamp    int64     `gorm:"column:timestamp;index" json:"timestamp"`                  // 日志时间戳
	ClientIP     string    `gorm:"column:client_ip;" json:"client_ip"`                       // 客户端IP
	Module       string    `gorm:"column:module;" json:"module"`                             // 模块
	TraceID      string    `gorm:"column:trace_id;" json:"trace_id"`                         // 链路追踪ID
	CreatedAt    time.Time `gorm:"column:created_at;index;autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;index;autoUpdateTime" json:"updated_at"` // 更新时间
}

// TableName 设置表名（实际表名会根据分表规则动态生成）
func (log *AuditLog) TableName() string {
	tableSuffix := strings.Split(log.LogId, "_")
	if len(tableSuffix) > 1 {
		return fmt.Sprintf("%s_%s", AuditLogName, tableSuffix[1])
	}
	return AuditLogName
}

func (log *AuditLog) Name() string {
	return AuditLogName
}

func (log *AuditLog) SetId(id string) {
	log.LogId = id
}

func (log *AuditLog) SaveBatch(ctx context.Context, tx *gorm.DB, batch []Entity) error {
	tabMap := make(map[string][]*AuditLog)
	for _, entity := range batch {
		al, ok := entity.(*AuditLog)
		if !ok {
			return fmt.Errorf("invalid entity type: %T, expected *AuditLog", entity)
		}
		tabMap[al.TableName()] = append(tabMap[al.TableName()], al)
	}

	for table, logs := range tabMap {
		err := tx.WithContext(ctx).Table(table).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "log_id"}},
			DoNothing: true,
		}).Create(logs).Error
		if err != nil {
			logx.Errorf("save audit log to table %s failed, err: %v", table, err)
			return err
		}
	}

	// 使用具体类型进行插入
	return nil
}

// CreateTable 创建任务信息表（需要任务类型，已经表的position）
func (log *AuditLog) CreateTable(db *gorm.DB) error {
	newTableName := log.TableName()
	return db.Table(newTableName).AutoMigrate(log)
}
