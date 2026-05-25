package models

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type NType string

const (
	NTypeLowBalance  NType = "low_balance"
	NTypeStaleReading NType = "stale_reading"
)

type NotificationLog struct {
	bun.BaseModel `bun:"table:notification_logs"`

	TimeStampedModel

	UserID            uuid.UUID `bun:"user_id,notnull,type:uuid"`
	MeterID           uuid.UUID `bun:"meter_id,notnull,type:uuid"`
	Platform          Platform  `bun:"platform,notnull,type:varchar(10)"`
	PlatformID        string    `bun:"platform_id,notnull,type:varchar(20)"`
	Balance           float64   `bun:"balance,notnull"`
	NotificationType  NType     `bun:"notification_type,notnull,type:varchar(20),default:'low_balance'"`
}
