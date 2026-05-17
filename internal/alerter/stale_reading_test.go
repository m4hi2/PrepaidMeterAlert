package alerter

import (
	"testing"
	"time"
)

func TestEffectiveReadingTime(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

	_, ok := effectiveReadingTime(nil, now)
	if ok {
		t.Fatal("nil reading time should be skipped")
	}

	past := now.Add(-24 * time.Hour)
	got, ok := effectiveReadingTime(&past, now)
	if !ok || !got.Equal(past) {
		t.Fatalf("past reading: got %v ok=%v", got, ok)
	}

	future := now.Add(2 * time.Hour)
	got, ok = effectiveReadingTime(&future, now)
	if !ok || !got.Equal(now) {
		t.Fatalf("future reading should clamp to now: got %v", got)
	}
}
