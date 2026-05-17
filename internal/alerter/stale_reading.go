package alerter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	tele "gopkg.in/telebot.v3"
)

func effectiveReadingTime(readingAt *time.Time, now time.Time) (time.Time, bool) {
	if readingAt == nil {
		return time.Time{}, false
	}
	t := *readingAt
	if t.After(now) {
		t = now
	}
	return t, true
}

func (a *Alerter) handleProviderReadingTime(ctx context.Context, meter *models.Meter, readingAt *time.Time, now time.Time) {
	t, ok := effectiveReadingTime(readingAt, now)
	if !ok {
		return
	}

	meter.ProviderReadingAt = &t

	if a.readingStaleThreshold <= 0 {
		return
	}

	stale := now.Sub(t) > a.readingStaleThreshold
	if stale && !meter.ReadingStaleNotified {
		a.notifyStaleReading(ctx, meter, t)
		meter.ReadingStaleNotified = true
		return
	}
	if !stale && meter.ReadingStaleNotified {
		meter.ReadingStaleNotified = false
	}
}

func (a *Alerter) notifyStaleReading(ctx context.Context, meter *models.Meter, readingAt time.Time) {
	user, err := a.userRepo.GetByID(ctx, meter.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user for stale-reading alert",
			"meter_id", meter.ID, "user_id", meter.UserID, "error", err)
		return
	}

	chatID, err := strconv.ParseInt(user.PlatformID, 10, 64)
	if err != nil {
		slog.ErrorContext(ctx, "invalid platform_id for stale-reading alert",
			"user_id", user.ID, "platform_id", user.PlatformID, "error", err)
		return
	}

	if err := a.tgLimiter.Wait(ctx); err != nil {
		slog.ErrorContext(ctx, "tg rate limiter wait failed for stale-reading alert",
			"meter_id", meter.ID, "error", err)
		return
	}

	name := meter.AccountNumber
	if meter.Nickname != "" {
		name = meter.Nickname
	}

	msg := fmt.Sprintf(
		"⚠️ Stale meter reading\nMeter: %s (%s)\nProvider last reported: %s\n\nThe utility reading may be outdated. Check your balance manually until readings update again.",
		name, meter.ProviderCode, readingAt.Format("2006-01-02"),
	)

	if _, err := a.bot.Send(&tele.Chat{ID: chatID}, msg, keyboards.MainMenu()); err != nil {
		slog.ErrorContext(ctx, "failed to send stale-reading alert",
			"meter_id", meter.ID, "chat_id", chatID, "error", err)
		return
	}

	slog.InfoContext(ctx, "stale-reading alert sent",
		"meter_id", meter.ID, "reading_at", readingAt)
}
