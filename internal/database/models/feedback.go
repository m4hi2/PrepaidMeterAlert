package models

import "github.com/uptrace/bun"

type Feedback struct {
	bun.BaseModel `bun:"table:feedbacks"`

	TimeStampedModel

	Platform   Platform `bun:"platform,notnull,type:varchar(10)"`
	PlatformID string   `bun:"platform_id,notnull,type:varchar(20)"`
	Text       string   `bun:"text,notnull,type:text"`
}
