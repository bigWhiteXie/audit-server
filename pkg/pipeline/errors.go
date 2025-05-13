package pipeline

import "errors"

// 错误定义
var (
	// 管道状态相关错误
	ErrPipelineBlocked    = errors.New("pipeline is blocked due to disk full")
	ErrQueueFull          = errors.New("pipeline queue is full")
	ErrPipelineNotStarted = errors.New("pipeline is not started")

	// 导出相关错误
	ErrExporterFailed    = errors.New("exporter failed to export data")
	ErrEncodingFailed    = errors.New("encoder failed to encode data")
	ErrCompressionFailed = errors.New("compressor failed to compress data")

	// 存储相关错误
	ErrDiskFull         = errors.New("local disk storage is full")
	ErrFileCreateFailed = errors.New("failed to create local storage file")
	ErrFileWriteFailed  = errors.New("failed to write to local storage file")

	// 插件相关错误
	ErrPluginNotRegistered = errors.New("required plugin is not registered")
	ErrInvalidPluginType   = errors.New("invalid plugin type")
)
