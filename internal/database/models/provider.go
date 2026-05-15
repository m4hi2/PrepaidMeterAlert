package models

import "github.com/uptrace/bun"

type Provider struct {
	bun.BaseModel `bun:"table:providers"`

	TimeStampedModel

	Code    ProviderCode `bun:"code,notnull,type:varchar(10),unique"`
	Enabled bool         `bun:"enabled,notnull,default:false"`
}
