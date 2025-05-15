package model

import (
	"context"

	"gorm.io/gorm"
)

type Entity interface {
	Name() string
	TableName() string
	SaveBatch(ctx context.Context, db *gorm.DB, data []Entity) error
	CreateTable(db *gorm.DB) error
	SetId(id string)
}
