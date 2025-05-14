package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"codexie.com/auditlog/pkg/plugin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 分表配置（每月一张表）
const (
	TablePrefix = "audit_log"
)

// 审计日志实体（对应分表结构 audit_log_YYYYMM）
type AuditLog struct {
	Id           int       `gorm:"primaryKey,autoIncrement" json:"id"`                       // 自增主键
	LogId        string    `gorm:"column:log_id;index;size:64" json:"log_id"`                // 日志ID
	TenantID     string    `gorm:"column:tenant_id;index;size:32" json:"tenant_id"`          // 租户ID
	UserID       string    `gorm:"column:user_id;index;size:64" json:"user_id"`              // 用户ID
	Username     string    `gorm:"column:username;size:128" json:"username"`                 // 用户名
	Action       string    `gorm:"column:action;index;size:64" json:"action"`                // 操作类型
	ResourceType string    `gorm:"column:resource_type;size:64" json:"resource_type"`        // 资源类型
	ResourceID   string    `gorm:"column:resource_id;size:128" json:"resource_id"`           // 资源ID
	ResourceName string    `gorm:"column:resource_name;size:255" json:"resource_name"`       // 资源名称
	Result       string    `gorm:"column:result;index;size:16" json:"result"`                // 操作结果
	Message      string    `gorm:"column:message;type:text" json:"message"`                  // 详细信息
	TimeStamp    int64     `gorm:"column:timestamp;index" json:"timestamp"`                  // 日志时间戳
	ClientIP     string    `gorm:"column:client_ip;size:64" json:"client_ip"`                // 客户端IP
	FromService  string    `gorm:"column:from_service;size:64" json:"from_service"`          // 来源服务
	TraceID      string    `gorm:"column:trace_id;size:128" json:"trace_id"`                 // 链路追踪ID
	CreatedAt    time.Time `gorm:"column:created_at;index;autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;index;autoUpdateTime" json:"updated_at"` // 更新时间
}

// TableName 设置表名（实际表名会根据分表规则动态生成）
func (log *AuditLog) TableName() string {
	tableSuffix := strings.Split(log.LogId, "_")
	if len(tableSuffix) > 1 {
		return fmt.Sprintf("%s_%s", TablePrefix, tableSuffix[1])
	}
	return TablePrefix
}

func (log *AuditLog) Name() string {
	return TablePrefix
}

func (log *AuditLog) SaveBatch(ctx context.Context, tx *gorm.DB, batch []plugin.Entity) error {
	// 转换为具体类型
	auditLogs := make([]*AuditLog, 0, len(batch))
	for _, entity := range batch {
		al, ok := entity.(*AuditLog)
		if !ok {
			return fmt.Errorf("invalid entity type: %T, expected *AuditLog", entity)
		}
		auditLogs = append(auditLogs, al)
	}

	// 获取表名（同批次表名相同）
	if len(auditLogs) == 0 {
		return nil
	}
	tableName := auditLogs[0].TableName()

	// 使用具体类型进行插入
	return tx.Table(tableName).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(auditLogs).Error
}
