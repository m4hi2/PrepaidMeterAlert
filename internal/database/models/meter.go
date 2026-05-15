package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ProviderCode string

const (
	ProviderCodeNESCO ProviderCode = "nesco"
	ProviderCodeDPDC  ProviderCode = "dpdc"
	ProviderCodeDESCO ProviderCode = "desco"
)

type NotifyMode string

const (
	NotifyModeSingle NotifyMode = "single"
	NotifyModeDaily  NotifyMode = "daily"
)

type FetchStatus string

const (
	FetchStatusPending FetchStatus = "pending"
	FetchStatusSuccess FetchStatus = "success"
	FetchStatusFailed  FetchStatus = "failed"
)

type NStatus string

const (
	NStatusNotNeeded NStatus = "not_needed"
	NStatusPending   NStatus = "pending"
	NStatusSuccess   NStatus = "success"
	NStatusFailed    NStatus = "failed"
)

type Meter struct {
	bun.BaseModel `bun:"table:meters"`

	TimeStampedModel

	UserID             uuid.UUID    `bun:"user_id,notnull,type:uuid"`
	ProviderCode       ProviderCode `bun:"provider,notnull,type:varchar(10)"`
	MeterNumber        string       `bun:"meter_number,notnull,type:varchar(20)"`
	AccountNumber      string       `bun:"account_number,notnull,type:varchar(20)"`
	Nickname           string       `bun:"nickname,nullzero,type:varchar(30)"`
	Threshold          float64      `bun:"threshold,notnull,default:100"`
	NotifyMode         NotifyMode   `bun:"notify_mode,notnull,type:varchar(10)"`
	Balance            float64      `bun:"balance,notnull,default:0"`
	LastFetchAt        *time.Time   `bun:"last_fetch_at,nullzero"`
	FetchStatus        FetchStatus  `bun:"fetch_status,notnull,type:varchar(10)"`
	NotificationStatus NStatus      `bun:"notification_status,notnull,type:varchar(10)"`
}
