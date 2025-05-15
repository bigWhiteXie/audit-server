package lifecycle

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"codexie.com/auditlog/internal/constant"
	"codexie.com/auditlog/internal/model"
	"codexie.com/auditlog/pkg/plugin"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type LogIdHook struct {
	redis        *redis.Client
	db           *gorm.DB
	singleFilght singleflight.Group
}

func NewLogIdHook(conf map[string]any) *LogIdHook {
	return &LogIdHook{
		redis:        conf["redis"].(*redis.Client),
		db:           conf["db"].(*gorm.DB),
		singleFilght: singleflight.Group{},
	}
}

// Name 返回插件名称
func (h *LogIdHook) Name() string { return "logid" }

// BeforeExport 导出前钩子
func (h *LogIdHook) BeforeExport(ctx context.Context, batch []interface{}) context.Context {
	entity := batch[0].(model.Entity)
	schedulePos := &model.SchedulePos{
		Name: entity.Name(),
	}
	key := fmt.Sprintf("%s:%s", constant.SchedulePosKey, entity.Name())
	res, err := h.redis.Get(ctx, key).Result()
	if err != nil {
		_, err, _ := h.singleFilght.Do(key, func() (interface{}, error) {
			schedulePos, err = schedulePos.GetSchedulePos(h.db, schedulePos.Name)
			if err != nil {
				return nil, err
			}
			endPos := strconv.Itoa(schedulePos.ScheduleEndPos)
			h.redis.Set(ctx, key, endPos, 0)
			res = endPos
			return schedulePos, nil
		})
		if err != nil {
			logx.Errorf("获取日志ID失败: %v", err)
			time.Sleep(time.Second * 10)
			return h.BeforeExport(ctx, batch)
		}
	}

	for _, entity := range batch {
		if entity, ok := entity.(model.Entity); ok {
			logId := fmt.Sprintf("%s_%s", uuid.New().String(), res)
			entity.SetId(logId)
		}
	}

	return ctx
}

// OnError 错误处理钩子
func (h *LogIdHook) OnError(ctx context.Context, err error, batch []interface{}) {
	// 无操作
}

func init() {
	plugin.RegisterLifecycleFactory("logid", func(config map[string]any) plugin.LifecycleHook {
		return NewLogIdHook(config)
	})
}
