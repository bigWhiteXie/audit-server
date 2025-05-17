package apierr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithErr(t *testing.T) {
	t.Run("without error in args", func(t *testing.T) {
		err := WithErr("E1001", "validation failed for %s", "user")
		assert.Equal(t, "validation failed for user", err.Error())
		assert.Equal(t, "E1001", err.RootCode())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("with error in args", func(t *testing.T) {
		originalErr := errors.New("database connection failed")
		err := WithErr("E1002", "operation failed: %v", originalErr)
		assert.Equal(t, "operation failed: database connection failed", err.Error())
		assert.Equal(t, "E1002", err.RootCode())
		assert.Equal(t, originalErr, err.Unwrap())
	})

	t.Run("with nested code error", func(t *testing.T) {
		innerErr := WithErr("E1003", "inner error")
		outerErr := WithErr("E1004", "wrapper error: %v", innerErr)

		assert.Equal(t, "wrapper error: inner error", outerErr.Error())
		assert.Equal(t, "E1003", outerErr.RootCode())
		assert.Equal(t, innerErr, outerErr.Unwrap())
	})
}

func TestCodeError_Error(t *testing.T) {
	err := &CodeError{
		RootCauseCode: "E1005",
		Msg:           "authentication failed",
	}
	assert.Equal(t, "authentication failed", err.Error())
}

func TestCodeError_RootCode(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		err := WithErr("E1006", "simple error")
		assert.Equal(t, "E1006", err.RootCode())
	})

	t.Run("wrapped code error", func(t *testing.T) {
		rootErr := WithErr("E1007", "root error")
		wrappedErr := rootErr.Wrap(errors.New("underlying error"))
		assert.Equal(t, "E1007", wrappedErr.RootCode())
	})

	t.Run("mixed error types", func(t *testing.T) {
		baseErr := errors.New("base error")
		codeErr := WithErr("E1008", "code error: %v", baseErr)
		wrappedErr := codeErr.Wrap(errors.New("network error"))

		assert.Equal(t, "E1008", wrappedErr.RootCode())
		assert.Equal(t, "network error", wrappedErr.Unwrap().Error())
	})
}

func TestCodeError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithErr("E1009", "wrapped error: %v", originalErr)
	assert.Equal(t, originalErr, err.Unwrap())
}

func TestCodeError_Wrap(t *testing.T) {
	originalErr := errors.New("original error")
	codeErr := WithErr("E1010", "code error")
	wrappedErr := codeErr.Wrap(originalErr)

	assert.Equal(t, originalErr, wrappedErr.Unwrap())
	assert.Equal(t, "code error", wrappedErr.Error())
	assert.Equal(t, "E1010", wrappedErr.RootCode())
}

// 测试WithErrf需要mock日志记录器，这里使用简单实现
type mockLogger struct {
	lastLog string
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {
	m.lastLog = fmt.Sprintf(format, args...)
}

func (m *mockLogger) Error(args ...any) {
	m.lastLog = fmt.Sprint(args...)
}

func TestWithErrf(t *testing.T) {
	mockLog := &mockLogger{}

	err := WithErrf(mockLog, "E1011", "failed to process %s", "request")

	assert.Equal(t, "failed to process request", err.Error())
	assert.Equal(t, "[E1011] failed to process request", mockLog.lastLog)
	assert.Equal(t, "E1011", err.RootCode())
}
