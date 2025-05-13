package filter

import (
	"codexie.com/auditlog/pkg/plugins"
)

// Field 字段过滤器
type Field struct {
	allowedFields map[string]bool
}

// NewField 创建新的字段过滤器
func NewField(allowedFields []string) *Field {
	allowed := make(map[string]bool)
	for _, field := range allowedFields {
		allowed[field] = true
	}

	return &Field{
		allowedFields: allowed,
	}
}

// Name 返回插件名称
func (f *Field) Name() string { return "field-filter" }

// Filter 过滤数据
func (f *Field) Filter(data interface{}) bool {
	return true
}

// 确保Field实现了Filter接口
var _ plugins.Filter = (*Field)(nil)
