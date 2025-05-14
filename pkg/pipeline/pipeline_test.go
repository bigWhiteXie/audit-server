package pipeline

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

// ConsoleExporter 简单实现，用于测试
type ConsoleExporter struct {
	mu    sync.Mutex
	count int
}

func (c *ConsoleExporter) Name() string { return "console-test" }

func (c *ConsoleExporter) Export(ctx context.Context, data []interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, item := range data {
		logx.Infof("export data: %v", item)
		c.count += 1
	}
	return nil
}

// FailingExporter 模拟导出失败
type FailingExporter struct{}

func (f *FailingExporter) Name() string { return "failing-test" }

func (f *FailingExporter) Export(ctx context.Context, data []interface{}) error {
	return fmt.Errorf("simulated export error")
}

func setupTestPipeline(t *testing.T, cfg config.PiplineConfig, exporter plugin.Exporter) *Pipeline {
	p := New(cfg)
	p.RegisterExporter(exporter)
	err := p.Start()
	require.NoError(t, err, "Pipeline Start should not return an error")
	return p
}

func TestPipeline_NormalOperation(t *testing.T) {
	cfg := config.PiplineConfig{
		Name:             "test-normal-pipeline",
		BatchSize:        10000, // 调整批次大小以便更快看到输出
		BatchTimeout:     1,
		StorageDir:       "./",
		RecoveryInterval: 1, // 加快恢复检查，虽然此测试不直接测恢复
	}

	consoleExporter := &ConsoleExporter{}
	p := setupTestPipeline(t, cfg, consoleExporter)
	defer p.Close()

	totalDataSent := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second) // 运行10秒多一点
	defer cancel()

	go func() {
		for { // 持续10秒
			select {
			case <-ctx.Done():
				return
			default:
				data := fmt.Sprintf("data-%d", totalDataSent+1)
				err := p.Push(data)
				if err != nil {
					// 在高并发下，队列满是可能的，这里我们允许一些推送失败
					// t.Logf("Push error (expected in high load): %v", err)
				} else {
					totalDataSent++
				}
			}
		}
	}()

	// 等待pipeline处理完剩余数据
	time.Sleep(time.Duration(cfg.BatchTimeout) * time.Second) // 给足够的时间让最后一个批次处理完

	p.Close()
	t.Logf("Total data sent: %d", totalDataSent)
	t.Logf("Total data exported by console: %d", consoleExporter.count)

	// 断言：由于队列大小和批处理机制，不一定所有发送的数据都会立即导出
	// 但应该有相当一部分数据被导出
	assert.True(t, consoleExporter.count > 0, "Expected some data to be exported")
	// 可以根据实际情况调整这个断言的阈值
	// 例如，如果BatchSize是500，10秒内发送10000条，至少应该有接近10000/500 = 20个批次
	// 但由于并发和定时器，可能不会完全精确
	assert.True(t, consoleExporter.count == totalDataSent, "Expected most of the sent data to be exported, accounting for queue and last batch")

	// 检查是否有不期望的错误日志文件（正常情况下不应该有）
	files, _ := filepath.Glob(filepath.Join("./", "pipeline-*.log"))
	assert.Empty(t, files, "No temporary error files should be created in normal operation")
}

func TestPipeline_ErrorOperation_LocalStorage(t *testing.T) {
	logIDir := "/usr/local/project/auditlog-server/test-log"
	cfg := config.PiplineConfig{
		Name:             "test-error-pipeline",
		BatchSize:        1000,
		BatchTimeout:     1,
		StorageDir:       logIDir,
		RecoveryInterval: 1, // 加快恢复检查
	}

	failingExporter := &FailingExporter{}
	p := setupTestPipeline(t, cfg, failingExporter)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // 运行较短时间
	defer cancel()

	sendCount := 0
	go func() {
	OuterLoop:
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				break OuterLoop
			default:
				data := fmt.Sprintf("error-data-%d", sendCount+1)
				err := p.Push(data) // 忽略错误，因为队列可能会满
				if err != nil {
					t.Logf("Push error (expected in high load): %v", err)
				} else {
					sendCount++
				}
			}
		}
	}()

	// 等待pipeline尝试处理并写入本地文件
	time.Sleep(time.Duration(cfg.BatchTimeout) + time.Duration(cfg.RecoveryInterval) + 500*time.Millisecond)
	p.Close()
	logIDir = path.Join(logIDir, cfg.Name)
	// 检查是否创建了临时文件
	files, err := filepath.Glob(filepath.Join(logIDir, "pipeline-*.log"))
	require.NoError(t, err, "Error reading storage directory")
	assert.NotEmpty(t, files, "Expected temporary error files to be created when exporter fails")

	t.Logf("Found %d temporary files: %v", len(files), files)

	// 可选：检查文件内容（部分）
	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		require.NoError(t, err)
		assert.True(t, strings.Contains(string(content), "error-data-"), "Temporary file should contain pushed data")
		t.Logf("Content of first temp file (first 100 chars): %s", string(content[:min(100, len(content))]))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
