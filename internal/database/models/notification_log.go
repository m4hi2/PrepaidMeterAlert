package models

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type NotificationLog struct {
	bun.BaseModel `bun:"table:notification_logs"`

	TimeStampedModel

	UserID     uuid.UUID `bun:"user_id,notnull,type:uuid"`
	MeterID    uuid.UUID `bun:"meter_id,notnull,type:uuid"`
	Platform   Platform  `bun:"platform,notnull,type:varchar(10)"`
	PlatformID string    `bun:"platform_id,notnull,type:varchar(20)"`
	Balance    float64   `bun:"balance,notnull"`
}
