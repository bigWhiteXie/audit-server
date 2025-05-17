package apierr

import (
	"fmt"

	"github.com/pkg/errors"
)

func WithErr(code string, format string, args ...any) *CodeError {
	// 若参数中包含error，则将其放入错误链中
	var paramErr error
	for _, arg := range args {
		if err, ok := arg.(error); ok {
			paramErr = err
			break
		}
	}

	return &CodeError{
		RootCauseCode: code,
		Msg:           fmt.Sprintf(format, args...),
		err:           paramErr,
	}
}

func WithErrf(log Log, code string, format string, args ...any) *CodeError {
	msg := fmt.Sprintf(format, args...)
	log.Errorf("[%s] %s", code, msg)
	return WithErr(code, msg)
}

type CodeError struct {
	RootCauseCode string
	Msg           string
	err           error
}

func (e *CodeError) Error() string {
	return e.Msg
}

func (e *CodeError) RootCode() string {
	// 遍历e.err的错误链，找到根因码
	var cause *CodeError
	for err := e.err; err != nil; err = errors.Unwrap(err) {
		if codeErr, ok := err.(*CodeError); ok {
			cause = codeErr
		}
	}

	if cause != nil {
		return cause.RootCauseCode
	}

	return e.RootCauseCode
}

func (e *CodeError) Unwrap() error {
	return e.err
}

func (e *CodeError) Wrap(target error) *CodeError {
	return &CodeError{
		RootCauseCode: e.RootCauseCode,
		Msg:           e.Msg,
		err:           target,
	}
}
