package handler

import (
	"context"
	"net/http"

	"codexie.com/auditlog/internal/types"
	"codexie.com/auditlog/pkg/apierr"
	"github.com/zeromicro/go-zero/core/logx"
)

func ApiErrorHandler(ctx context.Context, err error) (int, any) {
	switch err.(type) {
	case *apierr.CodeError:
		// 打印错误日志
		logx.Errorw("api error", logx.Field("error", err))
		return http.StatusBadRequest, &types.BaseResponse{
			Code:    err.(*apierr.CodeError).RootCode(),
			Message: err.Error(),
		}
	default:
		return http.StatusInternalServerError, &types.BaseResponse{
			Code:    "500",
			Message: err.Error(),
		}
	}
}
