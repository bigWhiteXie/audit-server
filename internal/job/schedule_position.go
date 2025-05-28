package job

import (
	"fmt"
	"time"

	"codexie.com/auditlog/internal/model"
	"gorm.io/gorm"
)

const (
	TableRecordThreshold = 30000000 // 单表最大记录数阈值
)

type SchedulePosition struct {
	nextRunTime time.Time
	db          *gorm.DB
}

func NewScheduleJob(db *gorm.DB) *SchedulePosition {
	return &SchedulePosition{
		db: db,
	}
}

func (s *SchedulePosition) Name() string {
	return "SchedulePositionTask"
}

func (s *SchedulePosition) Priority() int {
	return 1
}

func (s *SchedulePosition) ExeInterval() int64 {
	return 60 // 每60秒执行一次
}

func (s *SchedulePosition) Run() error {
	posModel := &model.SchedulePos{}
	posList, err := posModel.GetPosList(s.db)
	if err != nil {
		return fmt.Errorf("failed to get schedule_pos list: %w", err)
	}
	for _, pos := range posList {
		tableName := fmt.Sprintf("%s_%d", pos.Name, pos.ScheduleEndPos)
		var count int64
		err := s.db.Table(tableName).Count(&count).Error
		if err != nil {
			return fmt.Errorf("failed to count records in %s: %w", tableName, err)
		}
		if count >= TableRecordThreshold {
			pos.ScheduleEndPos++
			newTableName := fmt.Sprintf("%s_%d", pos.Name, pos.ScheduleEndPos)
			// 自动迁移新表结构
			entity := model.GetModel(pos.Name)
			err = s.db.Table(newTableName).AutoMigrate(entity)
			if err != nil {
				return fmt.Errorf("failed to migrate new table %s: %w", newTableName, err)
			}
			// 更新schedule_pos记录
			err = posModel.Save(s.db, pos)
			if err != nil {
				return fmt.Errorf("failed to update schedule_pos: %w", err)
			}
		}
	}
	return nil
}

func (s *SchedulePosition) NextRunTime() time.Time {
	return s.nextRunTime
}

func (s *SchedulePosition) SetNextRunTime(t time.Time) {
	s.nextRunTime = t
}
