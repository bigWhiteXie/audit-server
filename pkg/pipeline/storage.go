package pipeline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	maxFileSize    = 100 * 1024 * 1024 // 10MB
	minDiskSpace   = 100 * 1024 * 1024 // 100MB
	batchSize      = 100               // 每批恢复的数据量
	maxBufferLimit = 10 * 1024 * 1024  // 10MB
)

type ExportErrData struct {
	Name   string        `json:"name"`
	Data   []interface{} `json:"data"`
	Finish bool
}

// LocalStorage 本地存储，支持泛型
type LocalStorage struct {
	mu             sync.Mutex
	storageDir     string
	batchSize      int
	currentFile    *os.File
	currentSize    int64
	recoveringFile string
}

// NewLocalStorage 创建新的本地存储
func NewLocalStorage(storageDir string, batchSize int) *LocalStorage {
	s := &LocalStorage{
		storageDir: storageDir,
		batchSize:  batchSize,
	}

	return s
}

// Save 保存数据到本地文件
func (s *LocalStorage) Save(name string, batch []interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentFile == nil {
		if err := s.rotateFile(); err != nil {
			return err
		}
	}

	// 检查磁盘空间
	if s.isDiskFull() {
		return ErrDiskFull
	}

	//分批写入磁盘
	for index := 0; index < len(batch); index += s.batchSize {
		len := math.Min(float64(s.batchSize), float64(len(batch)-index))
		errData := ExportErrData{
			Name: name,
			Data: batch[index : index+int(len)],
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
			s.rotateFile()
		}
	}

	return nil
}

// Recover 从本地文件恢复数据，通过channel异步返回
func (s *LocalStorage) Recover() (<-chan ExportErrData, error) {
	dataCh := make(chan ExportErrData)

	// 获取所有备份文件
	files, err := filepath.Glob(filepath.Join(s.storageDir, "pipeline-*.log"))
	if err != nil {
		close(dataCh)
		return dataCh, fmt.Errorf("failed to list backup files: %w", err)
	}

	// 按文件名排序，确保按时间顺序处理
	sort.Strings(files)

	// 启动goroutine异步读取数据
	go func() {
		defer close(dataCh)

		for _, file := range files {
			if err := s.recoverFile(file, dataCh); err != nil {
				// 打印错误日志并继续处理下一个文件
				logx.Errorf("failed to recover file %s: %v", file, err)
				continue
			}
			dataCh <- ExportErrData{
				Name:   file,
				Finish: true,
			}
		}
	}()

	return dataCh, nil
}

func (s *LocalStorage) RemoveFile(filePath string) error {
	return os.Remove(filePath)
}

// recoverFile 从单个文件恢复数据
func (s *LocalStorage) recoverFile(filePath string, dataCh chan<- ExportErrData) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	s.recoveringFile = filePath
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

		var data ExportErrData
		if err := json.Unmarshal(line, &data); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}

		// 发送恢复的数据批次，如果channel已关闭则退出
		dataCh <- data
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

func (s *LocalStorage) rotateFile() error {
	if s.currentFile != nil {
		s.currentFile.Close()
	}

	// 确保目录存在
	if err := os.MkdirAll(s.storageDir, 0777); err != nil {
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
func (s *LocalStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentFile != nil {
		return s.currentFile.Close()
	}
	return nil
}
