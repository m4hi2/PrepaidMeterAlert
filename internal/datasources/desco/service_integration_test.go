//go:build integration

package desco

import (
	"context"
	"testing"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
)

func TestGetBalance_Real(t *testing.T) {
	id := datasources.Identifier{
		AccountNumber: "41378832",
		// MeterNumber:   "METER001",
	}

	svc := NewService(config.Get().Desco)
	got, err := svc.GetBalance(context.Background(), id)
	if err != nil {
		t.Fatalf("GetBalance error: %v", err)
	}
	t.Logf("Balance: %+v", got)
}
