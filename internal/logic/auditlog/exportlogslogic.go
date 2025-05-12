package auditlog

import (
	"context"

	"codexie.com/auditlog/internal/svc"
	"codexie.com/auditlog/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExportLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportLogsLogic {
	return &ExportLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ExportLogsLogic) ExportLogs(req *types.ExportRequest) (resp *types.ExportResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
