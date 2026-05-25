package nesco

import (
	"context"
	"testing"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/stretchr/testify/require"
)

func TestNescoService_GetBalance(t *testing.T) {
	cfg := config.NescoConfig{
		BasePath:   "https://customer.nesco.gov.bd",
		Timeout:    15 * time.Second,
		Retry:      2,
		RetryDelay: 500 * time.Millisecond,
		RateLimit:  2,
	}

	svc := NewService(cfg)

	// Use a test account number known to have a balance.
	// This should be replaced with a real test account or mocked in a real unit test.
	id := datasources.Identifier{AccountNumber: "12345678"}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	bal, err := svc.GetBalance(ctx, id)
	require.NoError(t, err)
	require.NotZero(t, bal.Balance, "balance should be > 0 for test number (or real number)")
	t.Logf("NESCO Balance for %v: %.2f", id, bal.Balance)
}
