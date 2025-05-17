package auditlog

import (
	"context"
	"strings"
	"time"

	"codexie.com/auditlog/internal/model"
	"codexie.com/auditlog/internal/svc"
	"codexie.com/auditlog/internal/types"
	"codexie.com/auditlog/pkg/util"

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
	// 从ctx中拿到user结构体
	user := l.ctx.Value("user").(*types.User)
	// 如果用户不是admin，则租户id只能为自己
	if util.ArrayContains[string](user.Roles, "admin") {
		req.TenantID = user.TenantID
	}

	// 将req中的字段封装成map(零值则忽略)
	queryMap := make(map[string]any)
	if req.TenantID != "" {
		queryMap["tenant_id"] = req.TenantID
	}
	if req.UserID != "" {
		queryMap["user_id"] = req.UserID
	}
	if req.Action != "" {
		queryMap["action"] = req.Action
	}
	if req.ResourceType != "" {
		queryMap["resource_type"] = req.ResourceType
	}
	if req.ResourceID != "" {
		queryMap["resource_id"] = req.ResourceID
	}
	if req.StartTime != 0 {
		queryMap["created_at >= ?"] = time.UnixMilli(req.StartTime)
	}
	if req.EndTime != 0 {
		queryMap["created_at <= ?"] = time.UnixMilli(req.EndTime)
	}

	queryMap["page"] = req.Page
	queryMap["page_size"] = req.PageSize
	queryMap["sort_field"] = req.SortField
	queryMap["sort_order"] = req.SortOrder

	logs := l.queryLogsByMap(queryMap)
	return &types.QueryResponse{List: logs, Total: len(logs)}, nil
}

func (l *QueryLogsLogic) queryLogsByMap(req map[string]any) []types.AuditLog {
	var logs []*model.AuditLog

	query := l.svcCtx.DB.Model(&model.AuditLog{})
	for k, v := range req {
		if strings.Contains(k, "?") {
			query = query.Where(k, v)
		} else {
			query = query.Where(k+" = ?", v)
		}
	}

	// 分页、排序
	query = query.Offset((req["page"].(int) - 1) * req["page_size"].(int)).Limit(req["page_size"].(int))
	sortField := req["sort_field"].(string)
	query = query.Order(sortField + " " + req["sort_order"].(string))

	if err := query.Find(&logs).Error; err != nil {
		l.Logger.Errorf("query audit logs failed: %v", err)
		return nil
	}

	auditLogs := make([]types.AuditLog, 0, len(logs))
	for _, log := range logs {
		auditLogs = append(auditLogs, types.AuditLog{
			TenantID:     log.TenantID,
			UserID:       log.UserID,
			Username:     log.Username,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID,
			ResourceName: log.ResourceName,
			Result:       log.Result,
			Message:      log.Message,
			ClientIP:     log.ClientIP,
			Module:       log.Module,
			TraceID:      log.TraceID,
		})
	}

	return auditLogs
}
