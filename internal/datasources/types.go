package datasources

import (
	"log/slog"
	"time"
)

type Identifier struct {
	AccountNumber string
	MeterNumber   string
}

type Balance struct {
	Identifier
	Balance     float64
	ReadingTime *time.Time
}

func ParseReadingTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	for _, f := range []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		"02-01-2006",
		"02/01/2006",
	} {
		if t, err := time.Parse(f, raw); err == nil {
			return &t
		}
	}
	slog.Debug("datasources: unhandled reading time format", "raw", raw)
	return nil
}

type AccountDetails struct {
	Identifier
	CustomerName    string
	CustomerContact string
	SanctionedLoad  string
	Address         string
}
