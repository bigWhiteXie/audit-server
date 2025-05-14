package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"codexie.com/auditlog/internal/config"
	"codexie.com/auditlog/internal/handler"
	"codexie.com/auditlog/internal/svc"
	"codexie.com/auditlog/pkg/pipeline"
	"codexie.com/auditlog/pkg/plugin"
	_ "codexie.com/auditlog/pkg/plugin/exporter"
	_ "codexie.com/auditlog/pkg/plugin/filter"
	_ "codexie.com/auditlog/pkg/plugin/lifecycle"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
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

	// =============初始化pipelines=============
	piplines := InitPiplines(c.Pipelines)
	for _, p := range piplines {
		p.Start()
	}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

func InitPiplines(piplineConfigs []config.PiplineConfig) []*pipeline.Pipeline {
	piplines := make([]*pipeline.Pipeline, 0, len(piplineConfigs))
	logx.Info("初始化piplines")
	for _, piplineConfig := range piplineConfigs {
		jsonStr, _ := json.Marshal(piplineConfig)
		logx.Infof("init piplines: %s", string(jsonStr))

		p := pipeline.New(piplineConfig)
		for _, expConf := range piplineConfig.Plugins.Exporters {
			exporter := plugin.GetExporter(expConf.Name, expConf.Config)
			p.RegisterExporter(exporter)
		}

		for _, filterConf := range piplineConfig.Plugins.Filters {
			filter := plugin.GetFilter(filterConf.Name, filterConf.Config)
			p.RegisterFilter(filter)
		}

		for _, lifecycleConf := range piplineConfig.Plugins.Lifecycles {
			lifecycle := plugin.GetLifecycle(lifecycleConf.Name, lifecycleConf.Config)
			p.RegisterLifecycleHook(lifecycle)
		}

		piplines = append(piplines, p)
	}

	return piplines
}
