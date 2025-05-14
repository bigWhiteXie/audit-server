package exporter

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"codexie.com/auditlog/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	testTablePrefix = "audit_log"
)

var (
	user     = "root"
	password = "xxxx"
	host     = "192.168.126.100"
	port     = 3306
	database = "auditlog"
)

func TestMySQLExporter_Export(t *testing.T) {
	// 使用测试容器或本地测试数据库
	dsn, _ := os.LookupEnv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, port, database)
	}

	// 初始化测试数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database")
	data := &model.AuditLog{
		LogId: "12345_202301",
	}
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", data.TableName()))

	db.Table(data.TableName()).AutoMigrate(&model.AuditLog{})

	// 初始化Exporter
	cfgMap := map[string]string{
		"host":     host,
		"port":     strconv.Itoa(port),
		"user":     user,
		"password": password,
		"database": database,
	}
	exporter := NewExporter(cfgMap)

	// 生成测试数据
	testData := make([]interface{}, 0, 1000)
	for i := 0; i < 1000; i++ {
		testData = append(testData, &model.AuditLog{
			LogId:        fmt.Sprintf("%d_202301", i),
			TenantID:     "test-tenant",
			UserID:       fmt.Sprintf("user-%04d", i),
			Username:     fmt.Sprintf("user%d", i),
			Action:       "CREATE",
			ResourceType: "VM",
			ResourceID:   fmt.Sprintf("vm-%04d", i),
			ResourceName: fmt.Sprintf("Virtual Machine %04d", i),
			Result:       "SUCCESS",
			Message:      "Resource created successfully",
			TimeStamp:    time.Now().UnixNano(),
			ClientIP:     "192.168.1.1",
			FromService:  "api-service",
			TraceID:      fmt.Sprintf("trace-%04d", i),
		})
	}

	// 执行导出
	t.Run("Normal Export", func(t *testing.T) {
		err := exporter.Export(context.Background(), testData)
		assert.NoError(t, err, "Export should succeed")

		// 验证数据
		var count int64
		db.Table(fmt.Sprintf("%s_202301", testTablePrefix)).Count(&count)
		assert.Equal(t, int64(1000), count, "All records should be inserted")

		// 验证随机抽样数据
		var sampleRecord model.AuditLog
		err = db.Table(fmt.Sprintf("%s_202301", testTablePrefix)).
			Where("log_id = ?", fmt.Sprintf("%d_202301", 420)).
			First(&sampleRecord).
			Error
		assert.NoError(t, err, "Should find sample record")
		assert.Equal(t, "user-0420", sampleRecord.UserID)
		assert.Equal(t, "vm-0420", sampleRecord.ResourceID)
	})

	t.Run("Duplicate Insert", func(t *testing.T) {
		// 再次插入相同数据（测试INSERT IGNORE）
		err := exporter.Export(context.Background(), testData[:100])
		assert.NoError(t, err, "Duplicate insert should not return error")

		// 验证数据总数不变
		var count int64
		db.Table(fmt.Sprintf("%s_202301", testTablePrefix)).Count(&count)
		assert.Equal(t, int64(1000), count, "Duplicate records should be ignored")
	})
}
