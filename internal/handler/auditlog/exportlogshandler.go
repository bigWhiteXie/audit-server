package auditlog

import (
	"net/http"

	"codexie.com/auditlog/internal/logic/auditlog"
	"codexie.com/auditlog/internal/svc"
	"codexie.com/auditlog/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ExportLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ExportRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := auditlog.NewExportLogsLogic(r.Context(), svcCtx)
		resp, err := l.ExportLogs(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
