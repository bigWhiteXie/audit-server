package types

import "codexie.com/auditlog/internal/model"

func (req *AuditLog) ToAuditLog() *model.AuditLog {
	return &model.AuditLog{
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		Username:     req.Username,
		Action:       req.Action,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		ResourceName: req.ResourceName,
		Result:       req.Result,
		Message:      req.Message,
		ClientIP:     req.ClientIP,
		Module:       req.Module,
		TraceID:      req.TraceID,
	}
}
