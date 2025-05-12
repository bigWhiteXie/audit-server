package auditlog

import (
	"context"

	"codexie.com/auditlog/internal/svc"
	"codexie.com/auditlog/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryLogsLogic {
	return &QueryLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryLogsLogic) QueryLogs(req *types.QueryRequest) (resp *types.QueryResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
