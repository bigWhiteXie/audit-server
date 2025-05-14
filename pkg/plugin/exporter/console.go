package exporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

// Console 控制台导出器
type Console[T any] struct {
	writer io.Writer
}

// NewConsole 创建新的控制台导出器
func NewConsole[T any]() *Console[T] {
	return &Console[T]{
		writer: os.Stdout,
	}
}

// Name 返回插件名称
func (e *Console[T]) Name() string { return "console" }

// Export 将数据导出到控制台
func (e *Console[T]) Export(ctx context.Context, data []interface{}) error {
	for _, d := range data {
		_, err := fmt.Fprintf(e.writer, "[%s] %s\n", time.Now().Format(time.RFC3339), d)
		if err != nil {
			return err
		}
	}
	return nil
}
