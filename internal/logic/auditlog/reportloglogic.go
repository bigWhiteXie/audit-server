package auditlog

import (
	"context"

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
	// todo: add your logic here and delete this line

	return
}
