package auditlog

import (
	"context"
	"errors"
	"strings"

	"codexie.com/auditlog/internal/svc"
	"codexie.com/auditlog/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReportLogLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReportLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReportLogLogic {
	return &ReportLogLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReportLogLogic) ReportLog(req *types.AuditLog) (resp *types.BaseResponse, err error) {
	auditLog := req.ToAuditLog()

	for _, p := range l.svcCtx.Piplines {
		if strings.ToLower(p.Name) == auditLog.Name() {
			if err := p.Push(auditLog); err != nil {
				// todo 返回预定义异常
				logx.Errorf("failed to push audit log to pipeline: %v", err)
				return nil, err
			}
			return &types.BaseResponse{
				Code:    "200",
				Message: "success",
			}, nil
		}
	}

	// todo 返回预定义异常
	return nil, errors.New("pipeline not found")
}
