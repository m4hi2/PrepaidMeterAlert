package alerter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/database/repo"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
	tele "gopkg.in/telebot.v3"
)

type Alerter struct {
	meterRepo       repo.MeterRepository
	userRepo        repo.UserRepository
	notifLogRepo    repo.NotificationLogRepository
	providerRepo    repo.ProviderRepository
	registry        datasources.Registry
	bot             *tele.Bot
	tgLimiter      *rate.Limiter
	staleThreshold  time.Duration

	tracer          trace.Tracer
	runDuration     metric.Float64Histogram
	fetchSuccess    metric.Int64Counter
	fetchFailed     metric.Int64Counter
	notifSent       metric.Int64Counter
	notifSuppressed metric.Int64Counter
}

func New(
	meterRepo repo.MeterRepository,
	userRepo repo.UserRepository,
	notifLogRepo repo.NotificationLogRepository,
	providerRepo repo.ProviderRepository,
	registry datasources.Registry,
	bot *tele.Bot,
	tgLimiter *rate.Limiter,
	staleThreshold time.Duration,
) (*Alerter, error) {
	m := otel.Meter("meterbot/tgbot")

	runDuration, err := m.Float64Histogram(
		"tg.handler.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Time to handle a Telegram update end-to-end"),
	)
	if err != nil {
		return nil, fmt.Errorf("init run duration metric: %w", err)
	}

	notifSent, err := m.Int64Counter(
		"tg.messages.sent",
		metric.WithDescription("Number of Telegram responses sent or edited"),
	)
	if err != nil {
		return nil, fmt.Errorf("init notif sent metric: %w", err)
	}

	alertMeter := otel.Meter("meterbot/alerter")

	fetchSuccess, err := alertMeter.Int64Counter(
		"alerter.fetch.success",
		metric.WithDescription("Number of meters whose balance was fetched successfully"),
	)
	if err != nil {
		return nil, fmt.Errorf("init fetch success metric: %w", err)
	}

	fetchFailed, err := alertMeter.Int64Counter(
		"alerter.fetch.failed",
		metric.WithDescription("Number of meters whose balance fetch failed"),
	)
	if err != nil {
		return nil, fmt.Errorf("init fetch failed metric: %w", err)
	}

	notifSuppressed, err := alertMeter.Int64Counter(
		"alerter.notifications.suppressed",
		metric.WithDescription("Number of notifications suppressed (single mode, already notified)"),
	)
	if err != nil {
		return nil, fmt.Errorf("init notif suppressed metric: %w", err)
	}

	return &Alerter{
		meterRepo:       meterRepo,
		userRepo:        userRepo,
		notifLogRepo:    notifLogRepo,
		providerRepo:    providerRepo,
		registry:        registry,
		bot:             bot,
		tgLimiter:       tgLimiter,
		staleThreshold:  staleThreshold,
		tracer:          otel.Tracer("meterbot/alerter"),
		runDuration:     runDuration,
		fetchSuccess:    fetchSuccess,
		fetchFailed:     fetchFailed,
		notifSent:       notifSent,
		notifSuppressed: notifSuppressed,
	}, nil
}

func (a *Alerter) Run(ctx context.Context) error {
	ctx, span := a.tracer.Start(ctx, "alerter.run", trace.WithSpanKind(trace.SpanKindInternal))
	start := time.Now()
	defer func() {
		a.runDuration.Record(ctx, float64(time.Since(start).Milliseconds()),
			metric.WithAttributes(attribute.String("tg.handler", "alerter")))
		span.End()
	}()

	slog.InfoContext(ctx, "alerter run started")

	activeCodes, err := a.providerRepo.GetActive(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("get active providers: %w", err)
	}
	activeSet := make(map[models.ProviderCode]struct{}, len(activeCodes))
	for _, c := range activeCodes {
		activeSet[c] = struct{}{}
	}

	meters, err := a.meterRepo.GetAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("get meters: %w", err)
	}

	slog.InfoContext(ctx, "alerter fetching balances", "total_meters", len(meters))
	toNotify, toStaleNotify := a.fetchAll(ctx, meters, activeSet)
	slog.InfoContext(ctx, "alerter fetch complete", "to_notify", len(toNotify), "to_stale_notify", len(toStaleNotify))

	a.notifyAll(ctx, toNotify)
	a.notifyAllStale(ctx, toStaleNotify)

	slog.InfoContext(ctx, "alerter run complete")
	return nil
}

type staleResult struct {
	meter  *models.Meter
	readingTime time.Time
}

func (a *Alerter) fetchAll(ctx context.Context, meters []*models.Meter, activeSet map[models.ProviderCode]struct{}) ([]*models.Meter, []*staleResult) {
	var toNotify []*models.Meter
	var toStaleNotify []*staleResult
	for _, meter := range meters {
		if _, active := activeSet[meter.ProviderCode]; !active {
			slog.DebugContext(ctx, "skipping meter: provider not active",
				"meter_id", meter.ID, "provider", meter.ProviderCode)
			continue
		}
		fetcher, ok := a.registry.Get(meter.ProviderCode)
		if !ok {
			slog.WarnContext(ctx, "skipping meter: no fetcher registered",
				"meter_id", meter.ID, "provider", meter.ProviderCode)
			continue
		}
		needsNotify, stale := a.fetchMeter(ctx, meter, fetcher)
		if needsNotify {
			toNotify = append(toNotify, meter)
		}
		if stale != nil {
			toStaleNotify = append(toStaleNotify, stale)
		}
	}
	return toNotify, toStaleNotify
}

func (a *Alerter) fetchMeter(ctx context.Context, meter *models.Meter, fetcher datasources.DataFetcher) (needsNotify bool, stale *staleResult) {
	providerAttr := attribute.String("meter.provider", string(meter.ProviderCode))
	fetchCtx, fetchSpan := a.tracer.Start(ctx, "alerter.fetch",
		trace.WithAttributes(
			attribute.String("meter.id", meter.ID.String()),
			providerAttr,
			attribute.String("meter.account_number", meter.AccountNumber),
		),
	)
	defer fetchSpan.End()

	prevBalance := meter.Balance

	bal, err := fetcher.GetBalance(fetchCtx, datasources.Identifier{
		AccountNumber: meter.AccountNumber,
		MeterNumber:   meter.MeterNumber,
	})
	now := time.Now()

	if err != nil {
		meter.FetchStatus = models.FetchStatusFailed
		fetchSpan.RecordError(err)
		fetchSpan.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(fetchCtx, "balance fetch failed",
			"meter_id", meter.ID, "provider", meter.ProviderCode, "error", err)
		a.fetchFailed.Add(fetchCtx, 1, metric.WithAttributes(providerAttr))
		if updateErr := a.meterRepo.Update(fetchCtx, meter); updateErr != nil {
			slog.ErrorContext(fetchCtx, "failed to update meter fetch status",
				"meter_id", meter.ID, "error", updateErr)
		}
		return false, nil
	}

	fetchedBalance := bal.Balance

	// Recharge detected (balance went up) or recovered above threshold — reset notification episode.
	if fetchedBalance > prevBalance || fetchedBalance >= meter.Threshold {
		meter.NotificationStatus = models.NStatusNotNeeded
	}

	meter.Balance = fetchedBalance
	meter.LastFetchAt = &now
	meter.FetchStatus = models.FetchStatusSuccess

	if bal.ReadingTime != nil {
		meter.LastReadingAt = bal.ReadingTime
	}

	a.fetchSuccess.Add(fetchCtx, 1, metric.WithAttributes(providerAttr))

	if updateErr := a.meterRepo.Update(fetchCtx, meter); updateErr != nil {
		slog.ErrorContext(fetchCtx, "failed to update meter balance",
			"meter_id", meter.ID, "error", updateErr)
		return false, nil
	}

	stale = a.checkStale(meter)

	return fetchedBalance < meter.Threshold, stale
}

func (a *Alerter) checkStale(meter *models.Meter) *staleResult {
	if a.staleThreshold <= 0 {
		return nil
	}

	if meter.LastReadingAt == nil {
		return nil
	}

	now := time.Now()

	if meter.LastReadingAt.After(now) {
		meter.StaleNotificationStatus = models.NStatusNotNeeded
		return nil
	}

	readingAge := now.Sub(*meter.LastReadingAt)
	if readingAge <= a.staleThreshold {
		meter.StaleNotificationStatus = models.NStatusNotNeeded
		return nil
	}

	return &staleResult{meter: meter, readingTime: *meter.LastReadingAt}
}

func needsNotification(meter *models.Meter) bool {
	switch meter.NotifyMode {
	case models.NotifyModeDaily:
		return true
	case models.NotifyModeSingle:
		return meter.NotificationStatus != models.NStatusSuccess
	default:
		return false
	}
}

func needsStaleNotification(meter *models.Meter) bool {
	return meter.StaleNotificationStatus != models.NStatusSuccess
}

func (a *Alerter) notifyAll(ctx context.Context, meters []*models.Meter) {
	for _, meter := range meters {
		if !needsNotification(meter) {
			slog.DebugContext(ctx, "notification suppressed",
				"meter_id", meter.ID, "notify_mode", meter.NotifyMode)
			a.notifSuppressed.Add(ctx, 1, metric.WithAttributes(
				attribute.String("meter.provider", string(meter.ProviderCode)),
				attribute.String("notify_mode", string(meter.NotifyMode)),
			))
			continue
		}

		a.notifyMeter(ctx, meter)
	}
}

func (a *Alerter) notifyMeter(ctx context.Context, meter *models.Meter) {
	user, err := a.userRepo.GetByID(ctx, meter.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user for notification",
			"meter_id", meter.ID, "user_id", meter.UserID, "error", err)
		return
	}

	chatID, err := strconv.ParseInt(user.PlatformID, 10, 64)
	if err != nil {
		slog.ErrorContext(ctx, "invalid platform_id for user",
			"user_id", user.ID, "platform_id", user.PlatformID, "error", err)
		return
	}

	notifAttrs := []attribute.KeyValue{
		attribute.String("tg.handler", "alerter"),
		attribute.String("tg.update_type", "notification"),
		attribute.String("meter.provider", string(meter.ProviderCode)),
		attribute.String("notify_mode", string(meter.NotifyMode)),
	}

	notifCtx, notifSpan := a.tracer.Start(ctx, "alerter.notify",
		trace.WithAttributes(append(notifAttrs,
			attribute.String("meter.id", meter.ID.String()),
			attribute.Int64("tg.chat_id", chatID),
			attribute.Float64("balance", meter.Balance),
			attribute.Float64("threshold", meter.Threshold),
		)...),
	)
	defer notifSpan.End()

	if err := a.tgLimiter.Wait(notifCtx); err != nil {
		notifSpan.RecordError(err)
		notifSpan.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(notifCtx, "tg rate limiter wait failed",
			"meter_id", meter.ID, "error", err)
		return
	}

	_, sendErr := a.bot.Send(&tele.Chat{ID: chatID}, buildMessage(meter), keyboards.MainMenu())
	if sendErr != nil {
		meter.NotificationStatus = models.NStatusFailed
		notifSpan.RecordError(sendErr)
		notifSpan.SetStatus(codes.Error, sendErr.Error())
		slog.ErrorContext(notifCtx, "failed to send notification",
			"meter_id", meter.ID, "chat_id", chatID, "error", sendErr)
	} else {
		meter.NotificationStatus = models.NStatusSuccess
		a.notifSent.Add(notifCtx, 1, metric.WithAttributes(notifAttrs...))
		slog.InfoContext(notifCtx, "notification sent",
			"meter_id", meter.ID, "chat_id", chatID, "balance", meter.Balance)

		if insertErr := a.notifLogRepo.Insert(notifCtx, &models.NotificationLog{
			UserID:           meter.UserID,
			MeterID:          meter.ID,
			Platform:         user.Platform,
			PlatformID:       user.PlatformID,
			Balance:          meter.Balance,
			NotificationType:  models.NTypeLowBalance,
		}); insertErr != nil {
			slog.ErrorContext(notifCtx, "failed to insert notification log",
				"meter_id", meter.ID, "error", insertErr)
		}
	}

	if updateErr := a.meterRepo.Update(notifCtx, meter); updateErr != nil {
		slog.ErrorContext(notifCtx, "failed to update meter notification status",
			"meter_id", meter.ID, "error", updateErr)
	}
}

func buildMessage(meter *models.Meter) string {
	name := meter.AccountNumber
	if meter.Nickname != "" {
		name = meter.Nickname
	}
	return fmt.Sprintf(
		"⚠️ Low Balance Alert\nMeter: %s (%s)\nBalance: %.2f BDT\nThreshold: %.2f BDT",
		name, meter.ProviderCode, meter.Balance, meter.Threshold,
	)
}

func (a *Alerter) notifyAllStale(ctx context.Context, results []*staleResult) {
	for _, sr := range results {
		if !needsStaleNotification(sr.meter) {
			slog.DebugContext(ctx, "stale notification suppressed",
				"meter_id", sr.meter.ID, "stale_notification_status", sr.meter.StaleNotificationStatus)
			a.notifSuppressed.Add(ctx, 1, metric.WithAttributes(
				attribute.String("meter.provider", string(sr.meter.ProviderCode)),
				attribute.String("notification_type", "stale_reading"),
			))
			continue
		}
		a.notifyStaleMeter(ctx, sr.meter, sr.readingTime)
	}
}

func (a *Alerter) notifyStaleMeter(ctx context.Context, meter *models.Meter, readingTime time.Time) {
	user, err := a.userRepo.GetByID(ctx, meter.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user for stale notification",
			"meter_id", meter.ID, "user_id", meter.UserID, "error", err)
		return
	}

	chatID, err := strconv.ParseInt(user.PlatformID, 10, 64)
	if err != nil {
		slog.ErrorContext(ctx, "invalid platform_id for user",
			"user_id", user.ID, "platform_id", user.PlatformID, "error", err)
		return
	}

	notifAttrs := []attribute.KeyValue{
		attribute.String("tg.handler", "alerter"),
		attribute.String("tg.update_type", "stale_notification"),
		attribute.String("meter.provider", string(meter.ProviderCode)),
	}

	notifCtx, notifSpan := a.tracer.Start(ctx, "alerter.notify_stale",
		trace.WithAttributes(append(notifAttrs,
			attribute.String("meter.id", meter.ID.String()),
			attribute.Int64("tg.chat_id", chatID),
		)...),
	)
	defer notifSpan.End()

	if err := a.tgLimiter.Wait(notifCtx); err != nil {
		notifSpan.RecordError(err)
		notifSpan.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(notifCtx, "tg rate limiter wait failed",
			"meter_id", meter.ID, "error", err)
		return
	}

	msg := buildStaleMessage(meter, readingTime)
	_, sendErr := a.bot.Send(&tele.Chat{ID: chatID}, msg, keyboards.MainMenu())
	if sendErr != nil {
		meter.StaleNotificationStatus = models.NStatusFailed
		notifSpan.RecordError(sendErr)
		notifSpan.SetStatus(codes.Error, sendErr.Error())
		slog.ErrorContext(notifCtx, "failed to send stale notification",
			"meter_id", meter.ID, "chat_id", chatID, "error", sendErr)
	} else {
		meter.StaleNotificationStatus = models.NStatusSuccess
		a.notifSent.Add(notifCtx, 1, metric.WithAttributes(notifAttrs...))
		slog.InfoContext(notifCtx, "stale notification sent",
			"meter_id", meter.ID, "chat_id", chatID)

		if insertErr := a.notifLogRepo.Insert(notifCtx, &models.NotificationLog{
			UserID:           meter.UserID,
			MeterID:          meter.ID,
			Platform:         user.Platform,
			PlatformID:       user.PlatformID,
			Balance:          meter.Balance,
			NotificationType: models.NTypeStaleReading,
		}); insertErr != nil {
			slog.ErrorContext(notifCtx, "failed to insert stale notification log",
				"meter_id", meter.ID, "error", insertErr)
		}
	}

	if updateErr := a.meterRepo.Update(notifCtx, meter); updateErr != nil {
		slog.ErrorContext(notifCtx, "failed to update meter stale notification status",
			"meter_id", meter.ID, "error", updateErr)
	}
}

func buildStaleMessage(meter *models.Meter, readingTime time.Time) string {
	name := meter.AccountNumber
	if meter.Nickname != "" {
		name = meter.Nickname
	}
	days := int(time.Since(readingTime).Hours() / 24)
	return fmt.Sprintf(
		"⏰ Stale Reading Alert\nMeter: %s (%s)\nLast provider reading: %s (%d day(s) ago)\n\nThe provider may not be updating this meter. Check your balance manually.",
		name, meter.ProviderCode, readingTime.Format("2006-01-02"), days,
	)
}

