package types

import (
	"context"

	"codexie.com/auditlog/internal/constant"
	"codexie.com/auditlog/pkg/apierr"
	"github.com/zeromicro/go-zero/core/logx"
)

func (q *QueryRequest) Validate(ctx context.Context) error {
	// 参数合法性校验
	if q.Page < 0 || q.Page > constant.MAX_PAGE {
		return apierr.WithErrf(logx.WithContext(ctx), "E00001", "page must be less than %d", constant.MAX_PAGE)
	}
	if q.PageSize < 0 || q.PageSize > constant.MAX_PAGE_SIZE {
		return apierr.WithErrf(logx.WithContext(ctx), "E00001", "page_size must be less than %d", constant.MAX_PAGE_SIZE)
	}

	if q.SortField != "" && !constant.ValidFields[q.SortField] {
		return apierr.WithErrf(logx.WithContext(ctx), "E00001", "sort_field must be one of %v", constant.ValidFields)
	}

	// 设置参数默认字段
	if q.SortField == "" {
		q.SortField = "created_at"
	}
	if q.SortOrder == "" {
		q.SortOrder = "desc"
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 10
	}

	return nil
}
