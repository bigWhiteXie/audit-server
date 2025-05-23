package pipeline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	maxFileSize    = 100 * 1024 * 1024 // 10MB
	minDiskSpace   = 100 * 1024 * 1024 // 100MB
	batchSize      = 100               // 每批恢复的数据量
	maxBufferLimit = 10 * 1024 * 1024  // 10MB
)

type ExportErrData[T any] struct {
	Name string `json:"name"`
	Data *T     `json:"data"`
}

// LocalStorage 本地存储，支持泛型
type LocalStorage[T any] struct {
	mu          sync.Mutex
	storageDir  string
	batchSize   int
	currentFile *os.File
	currentSize int64
}

// NewLocalStorage 创建新的本地存储
func NewLocalStorage[T any](storageDir string, batchSize int) *LocalStorage[T] {
	return &LocalStorage[T]{
		storageDir: storageDir,
		batchSize:  batchSize,
	}
}

// Save 保存数据到本地文件
func (s *LocalStorage[T]) Save(name string, batch []T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查磁盘空间
	if s.isDiskFull() {
		return ErrDiskFull
	}

	if s.currentFile == nil {
		if err := s.rotateFile(); err != nil {
			return err
		}
	}

	errData := make([]ExportErrData[T], len(batch))
	for i, data := range batch {
		errData[i] = ExportErrData[T]{
			Name: name,
			Data: &data,
		}
	}

	// 序列化数据并写入
	data, err := json.Marshal(errData)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	n, err := s.currentFile.Write(data)
	if err != nil {
		return ErrFileWriteFailed
	}

	s.currentSize += int64(n)

	if s.currentSize > maxFileSize {
		return s.rotateFile()
	}
	return nil
}

// Recover 从本地文件恢复数据，通过channel异步返回
func (s *LocalStorage[T]) Recover() (<-chan []ExportErrData[T], error) {
	dataCh := make(chan []ExportErrData[T])
	errCh := make(chan error, 1)

	defer close(dataCh)
	defer close(errCh)

	// 获取所有备份文件
	files, err := filepath.Glob(filepath.Join(s.storageDir, "pipeline-*.log"))
	if err != nil {
		return dataCh, fmt.Errorf("failed to list backup files: %w", err)
	}

	// 按文件名排序，确保按时间顺序处理
	sort.Strings(files)

	for _, file := range files {
		if err := s.recoverFile(file, dataCh); err != nil {
			return dataCh, fmt.Errorf("failed to recover file %s: %w", file, err)
		}

		// 恢复完成后删除文件
		if err := os.Remove(file); err != nil {
			errCh <- fmt.Errorf("failed to remove processed file %s: %w", file, err)
		}
	}

	return dataCh, nil
}

// recoverFile 从单个文件恢复数据
func (s *LocalStorage[T]) recoverFile(filePath string, dataCh chan<- []ExportErrData[T]) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小，处理大行
	maxCapacity := s.batchSize * 1024
	if maxCapacity > maxBufferLimit {
		maxCapacity = maxBufferLimit
	}
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var batch []ExportErrData[T]
		if err := json.Unmarshal(line, &batch); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}

		// 发送恢复的数据批次
		dataCh <- batch
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

func (s *LocalStorage[T]) rotateFile() error {
	if s.currentFile != nil {
		s.currentFile.Close()
	}

	// 确保目录存在
	if err := os.MkdirAll(s.storageDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(s.storageDir,
		fmt.Sprintf("pipeline-%s.log", time.Now().Format("20060102-150405")))
	f, err := os.Create(filename)
	if err != nil {
		return ErrFileCreateFailed
	}

	s.currentFile = f
	s.currentSize = 0
	return nil
}

// Close 关闭存储
func (s *LocalStorage[T]) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentFile != nil {
		return s.currentFile.Close()
	}
	return nil
}
