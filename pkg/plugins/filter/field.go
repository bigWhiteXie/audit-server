package filter

import (
	"codexie.com/auditlog/pkg/plugins"
)

// Field 字段过滤器
type Field[T any] struct {
	allowedFields map[string]bool
}

// NewField 创建新的字段过滤器
func NewField[T any](allowedFields []string) *Field[T] {
	allowed := make(map[string]bool)
	for _, field := range allowedFields {
		allowed[field] = true
	}

	return &Field[T]{
		allowedFields: allowed,
	}
}

// Name 返回插件名称
func (f *Field[T]) Name() string { return "field-filter" }

// Filter 过滤数据
func (f *Field[T]) Filter(data T) bool {
	return true
}

// 确保Field实现了Filter接口
var _ plugins.Filter[any] = (*Field[any])(nil)
