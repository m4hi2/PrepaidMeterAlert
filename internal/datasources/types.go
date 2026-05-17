package datasources

import "time"

type Identifier struct {
	AccountNumber string
	MeterNumber   string
}

type Balance struct {
	Identifier
	Balance     float64
	ReadingTime *time.Time // provider-reported reading timestamp; nil if unavailable
}

type AccountDetails struct {
	Identifier
	CustomerName    string
	CustomerContact string
	SanctionedLoad  string
	Address         string
}
