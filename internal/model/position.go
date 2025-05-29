package model

import (
	"fmt"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var TaskPosNsp SchedulePos

// SchedulePos taskPos
type SchedulePos struct {
	Id               uint64    `gorm:"column:id;primary_key;auto_increment"`
	Name             string    `gorm:"column:name;uniqueIndex:idx_name;type:varchar(155)"`
	ScheduleBeginPos int       `gorm:"column:schedule_begin_pos;not null"`
	ScheduleEndPos   int       `gorm:"column:schedule_end_pos;not null"`
	CreateTime       time.Time `gorm:"column:create_time;not null;autoCreateTime"`
	ModifyTime       time.Time `gorm:"column:modify_time;not null;autoUpdateTime"`
}

// TableName 表名
func (p *SchedulePos) TableName() string {
	return "schedule_pos"
}

// Create 创建记录
func (p *SchedulePos) Create(db *gorm.DB, task *SchedulePos) error {
	err := db.Table(p.TableName()).Create(task).Error
	return err
}

// Save 保存记录
func (p *SchedulePos) Save(db *gorm.DB, task *SchedulePos) error {
	err := db.Table(p.TableName()).Save(task).Error
	if err != nil {
		logx.Error("保存表位置失败", "entity", task.Name, "error", err)
	}
	return err
}

// GetSchedulePos 获取记录
func (p *SchedulePos) GetSchedulePos(db *gorm.DB, name string) (*SchedulePos, error) {
	var taskPos = new(SchedulePos)
	err := db.Table(p.TableName()).Where("name = ?", name).First(&taskPos).Error

	if err == gorm.ErrRecordNotFound {
		p.Name = name
		p.ScheduleBeginPos = 1
		p.ScheduleEndPos = 1
		err := db.Model(p).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(p).Error

		// 重新查询记录
		if err == nil {
			err = db.Table(p.TableName()).Where("name = ?", name).First(&taskPos).Error
		}
	}

	if err != nil {
		return nil, err
	}

	return taskPos, nil
}

// GetNextPos 获取下一个调度指针
func (p *SchedulePos) GetNextPos(pos string) string {
	posInt, err := strconv.Atoi(pos)
	if err != nil {
		logx.Errorf("pos %s maybe not int", pos)
		return ""
	}
	return fmt.Sprintf("%d", posInt+1)
}

// GetTaskPosList 获取记录列表
func (p *SchedulePos) GetPosList(db *gorm.DB) ([]*SchedulePos, error) {
	var taskList = make([]*SchedulePos, 0)
	err := db.Table(p.TableName()).Find(&taskList).Error
	if err != nil {
		return nil, err
	}
	return taskList, nil
}
