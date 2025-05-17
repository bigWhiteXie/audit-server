package main

import (
	"flag"
	"fmt"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/internal/handler"
	"codexie.com/auditlog/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/auditlog-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// =============启动pipelines=============
	for _, p := range ctx.Piplines {
		p.Start()
	}

	// =============启动任务调度=============
	go ctx.Scheduler.Start()
	httpx.SetErrorHandlerCtx(handler.ApiErrorHandler)
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
