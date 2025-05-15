package pipeline

import (
	"context"
	"path"
	"sync"
	"time"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/pkg/plugin"
	"github.com/zeromicro/go-zero/core/logx"
)

type plugins struct {
	exporter   map[string]plugin.Exporter
	filters    []plugin.Filter
	lifecycles []plugin.LifecycleHook
}

// 管道核心结构
type Pipeline struct {
	config.PiplineConfig

	queue      chan interface{}
	plugins    plugins
	blockData  []interface{}
	state      *State
	localStore *LocalStorage
	metrics    *Metrics
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	mu         sync.RWMutex
}

func New(cfg config.PiplineConfig) *Pipeline {
	cfg.SetDefaults()
	ctx, cancel := context.WithCancel(context.Background())

	p := &Pipeline{
		PiplineConfig: cfg,
		queue:         make(chan interface{}, cfg.BatchSize*10),
		state:         NewState(),
		ctx:           ctx,
		cancel:        cancel,
		blockData:     make([]interface{}, 0),
		plugins: plugins{
			exporter:   make(map[string]plugin.Exporter),
			filters:    make([]plugin.Filter, 0),
			lifecycles: make([]plugin.LifecycleHook, 0),
		},
	}

	// 初始化本地存储
	p.localStore = NewLocalStorage(path.Join(cfg.StorageDir, cfg.Name), cfg.BatchSize)

	// 初始化指标
	p.metrics = NewMetrics(cfg.Name)

	return p
}

// Start 启动管道处理
func (p *Pipeline) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return nil
	}
	logx.Infof("===========================pipeline %s started===========================", p.Name)
	// 启动主处理器
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.processor()
	}()

	// 启动恢复状态监控
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.recoveryMonitor()
	}()

	p.started = true
	return nil
}

func (p *Pipeline) Push(data interface{}) error {
	if p.state.IsBlocked() {
		return ErrPipelineBlocked
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.started {
		return ErrPipelineNotStarted
	}
	select {
	case p.queue <- data:
		p.metrics.QueueSize.WithLabelValues(p.Name).Inc()
		return nil
	default:
		return ErrQueueFull
	}
}

func (p *Pipeline) processor() {
	batch := make([]interface{}, 0, p.BatchSize)
	timer := time.NewTimer(time.Duration(p.BatchTimeout) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// 处理剩余数据
			for data := range p.queue {
				batch = append(batch, data)
			}
			if len(batch) > 0 {
				p.flushBatch(batch)
			}
			return

		case data := <-p.queue:
			p.metrics.QueueSize.WithLabelValues(p.Name).Dec()
			batch = append(batch, data)
			if len(batch) >= p.BatchSize {
				p.flushBatch(batch)
				batch = batch[:0]
				timer.Reset(time.Duration(p.BatchTimeout) * time.Second)
			}

		case <-timer.C:
			if len(batch) > 0 {
				p.flushBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(time.Duration(p.BatchTimeout) * time.Second)
		}
	}
}

// 恢复监控：尝试读取磁盘中的异常数据进行导出
func (p *Pipeline) recoveryMonitor() {
	ticker := time.NewTicker(time.Duration(p.RecoveryInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			switch p.state.GetStatus() {
			case StatusRecovering:
				p.tryRecoverFromDisk()
			case StatusBlocked:
				if !p.localStore.isDiskFull() {
					if err := p.localStore.Save(p.Name, p.blockData); err != nil {
						continue
					}
					p.blockData = p.blockData[:0]
					p.state.EnterRecovering()
				}
			}
		}
	}
}

// 尝试从磁盘恢复数据
func (p *Pipeline) tryRecoverFromDisk() {

	// 获取恢复数据通道
	dataCh, err := p.localStore.Recover()
	if err != nil {
		return
	}

	// 处理恢复的数据
	successCount := 0
	errorCount := 0
	exportSuccess := true
	for batch := range dataCh {
		// 尝试导出恢复的数据
		ctx := context.Background()
		// 单个文件读取结束
		if batch.Finish {
			if !exportSuccess {
				exportSuccess = true
				continue
			}
			go func() {
				// 导出成功则删除异常日志
				if err := p.localStore.RemoveFile(batch.Name); err != nil {
					logx.Errorf("failed to remove data: %v", err)
				}
			}()
			continue
		}
		exporter := p.plugins.exporter[batch.Name]
		if err := exporter.Export(ctx, batch.Data); err != nil {
			exportSuccess = false
			errorCount++
			logx.Errorf("failed to export data: %v", err)
			continue
		}

		successCount += len(batch.Data)
	}

	// 如果成功恢复了数据，切换到正常状态
	if successCount > 0 && errorCount == 0 {
		p.state.EnterNormal()
	}
}

func (p *Pipeline) flushBatch(batch []interface{}) {
	ctx := context.Background()

	// 阻塞状态则将数据缓存到内存中
	if p.state.GetStatus() == StatusBlocked {
		p.blockData = append(p.blockData, batch...)
		return
	}
	// 执行前置钩子
	for _, hook := range p.plugins.lifecycles {
		ctx = hook.BeforeExport(ctx, batch)
	}

	// 创建一个新的slice来存储通过过滤的数据
	filteredBatch := make([]interface{}, 0, len(batch))
	for i := range batch {
		shouldKeep := true
		// 检查所有过滤器
		for _, filter := range p.plugins.filters {
			if !filter.Filter(batch[i]) {
				shouldKeep = false
				break
			}
		}
		// 如果通过所有过滤器，则保留该数据
		if shouldKeep {
			filteredBatch = append(filteredBatch, batch[i])
		}
	}

	// 用过滤后的数据替换原始batch
	batch = filteredBatch

	// 导出日志
	wg := sync.WaitGroup{}
	for _, exporter := range p.plugins.exporter {
		wg.Add(1)
		go func(exporter plugin.Exporter) {
			defer wg.Done()
			start := time.Now()
			if err := exporter.Export(ctx, filteredBatch); err != nil {
				p.handleExportError(exporter.Name(), batch)
				// 执行错误钩子
				for _, hook := range p.plugins.lifecycles {
					hook.OnError(context.Background(), err, batch)
				}
				return
			}
			p.metrics.ExportLatency.WithLabelValues(exporter.Name()).Observe(float64(time.Since(start).Milliseconds()))
		}(exporter)
	}
	wg.Wait()

	p.metrics.SuccessCounter.WithLabelValues(p.Name).Inc()
}

func (p *Pipeline) handleExportError(name string, batch []interface{}) {
	p.metrics.ErrorCounter.WithLabelValues(p.Name).Add(float64(len(batch)))

	// 尝试本地存储
	if saveErr := p.localStore.Save(name, batch); saveErr != nil {
		if saveErr == ErrDiskFull {
			p.state.EnterBlocked()
		}
	} else {
		// 如果成功保存到本地，进入恢复模式
		if p.state.GetStatus() != StatusRecovering {
			p.state.EnterRecovering()
		}
	}
}

// 关闭管道，等待所有数据处理完成
func (p *Pipeline) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		return nil
	}
	p.started = false

	p.cancel()
	close(p.queue)
	p.wg.Wait()
	logx.Infof("===========================pipeline %s closed===========================", p.Name)
	return p.localStore.Close()
}
