package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TimeStampedModel struct {
	bun.BaseModel

	ID        uuid.UUID  `bun:"id,pk,type:uuid,default:uuidv7()"`
	CreatedAt *time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt *time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}

func (m *TimeStampedModel) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	now := time.Now()

	switch query.(type) {
	case *bun.InsertQuery:
		m.CreatedAt = &now
		m.UpdatedAt = &now
	case *bun.UpdateQuery:
		m.UpdatedAt = &now
	}

	return nil
}
