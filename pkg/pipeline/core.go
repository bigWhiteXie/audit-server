package pipeline

import (
	"context"
	"sync"
	"time"

	"codexie.com/auditlog/pkg/plugins"
)

// 核心接口定义
type Plugin interface {
	Name() string
}

type Exporter interface {
	Plugin
	Export(ctx context.Context, data []byte) error
}

type Filter interface {
	Plugin
	Filter(data any) (any, error)
}

// 生命周期钩子
type LifecycleHook interface {
	Plugin
	BeforeExport(ctx context.Context, batch []any) context.Context
	OnError(ctx context.Context, err error, batch []any)
}

// 管道核心结构
type Pipeline[T any] struct {
	config  Config
	queue   chan T
	plugins struct {
		exporter   []plugins.Exporter[T]
		filters    []plugins.Filter[T]
		lifecycles []plugins.LifecycleHook[T]
	}
	state      *State
	localStore *LocalStorage[T]
	metrics    *Metrics
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

type Config struct {
	Name          string
	BatchSize     int
	BatchTimeout  time.Duration
	QueueSize     int
	StorageDir    string
	MetricsPrefix string
}

func New[T any](cfg Config) *Pipeline[T] {
	ctx, cancel := context.WithCancel(context.Background())

	p := &Pipeline[T]{
		config: cfg,
		queue:  make(chan T, cfg.QueueSize),
		state:  NewState(),
		ctx:    ctx,
		cancel: cancel,
	}

	// 初始化本地存储
	p.localStore = NewLocalStorage[T](cfg.StorageDir)

	// 初始化指标
	p.metrics = NewMetrics(cfg.MetricsPrefix)

	// 注册默认生命周期钩子
	p.RegisterLifecycleHook(&NoopLifecycleHook[T]{})

	// 启动处理器
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.processor()
	}()

	return p
}

func (p *Pipeline[T]) Push(data T) error {
	if p.state.IsBlocked() {
		return ErrPipelineBlocked
	}

	select {
	case p.queue <- data:
		p.metrics.QueueSize.WithLabelValues(p.config.Name).Inc()
		return nil
	default:
		return ErrQueueFull
	}
}

func (p *Pipeline[T]) processor() {
	batch := make([]T, 0, p.config.BatchSize)
	timer := time.NewTimer(p.config.BatchTimeout)
	defer timer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// 处理剩余数据
			if len(batch) > 0 {
				p.flushBatch(batch)
			}
			return

		case data := <-p.queue:
			p.metrics.QueueSize.WithLabelValues(p.config.Name).Dec()
			batch = append(batch, data)
			if len(batch) >= p.config.BatchSize {
				p.flushBatch(batch)
				batch = batch[:0]
				timer.Reset(p.config.BatchTimeout)
			}

		case <-timer.C:
			if len(batch) > 0 {
				p.flushBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(p.config.BatchTimeout)
		}
	}
}

func (p *Pipeline[T]) flushBatch(batch []T) {
	if len(p.plugins.exporter) == 0 {
		p.handleError(ErrPluginNotRegistered, batch)
		return
	}

	ctx := context.Background()

	// 执行前置钩子
	for _, hook := range p.plugins.lifecycles {
		ctx = hook.BeforeExport(ctx, batch)
	}

	// 过滤处理
	processed := make([]T, 0, len(batch))

	for i := range processed {
		filtered := true
		for _, filter := range p.plugins.filters {
			if !filter.Filter(processed[i]) {
				filtered = false
				break
			}
		}
		if filtered {
			processed[i] = processed[i]
		}
	}

	// 导出日志
	wg := sync.WaitGroup{}
	for _, exporter := range p.plugins.exporter {
		wg.Add(1)
		go func(exporter plugins.Exporter[T]) {
			defer wg.Done()
			start := time.Now()
			if err := exporter.Export(ctx, processed); err != nil {
				p.handleError(ErrExporterFailed, batch)
				return
			}
			p.metrics.ExportLatency.WithLabelValues(exporter.Name()).Observe(float64(time.Since(start).Milliseconds()))
		}(exporter)
	}
	wg.Wait()

	p.metrics.SuccessCounter.WithLabelValues(p.config.Name).Inc()
}

func (p *Pipeline[T]) handleError(err error, batch []T) {
	p.metrics.ErrorCounter.WithLabelValues(p.config.Name).Inc()
	p.state.EnterRecovering()
	// 尝试本地存储
	if saveErr := p.localStore.Save(batch); saveErr != nil {
		if saveErr == ErrDiskFull {
			p.state.EnterBlocked()
		}
	}

	// 执行错误钩子
	for _, hook := range p.plugins.lifecycles {
		hook.OnError(context.Background(), err, batch)
	}
}

// 关闭管道，等待所有数据处理完成
func (p *Pipeline[T]) Close() error {
	p.cancel()
	p.wg.Wait()
	return nil
}
