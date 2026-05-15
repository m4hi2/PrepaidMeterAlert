package models

import "github.com/uptrace/bun"

type Platform string

const (
	PlatformTelegram Platform = "telegram"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	TimeStampedModel

	Platform   Platform `bun:"platform,notnull,type:varchar(10)"`
	PlatformID string   `bun:"platform_id,notnull,type:varchar(20)"`
	FirstName  string   `bun:"first_name,type:varchar(64)"`
	LastName   string   `bun:"last_name,type:varchar(64)"`
	Username   string   `bun:"username,type:varchar(64)"`
	IsBot      bool     `bun:"is_bot,notnull,default:false"`
}
