package tgbot

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/database/repo"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/handlers"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/middleware"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/state"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	b *tele.Bot
}

func New(
	cfg config.TelegramConfig,
	userRepo repo.UserRepository,
	meterRepo repo.MeterRepository,
	providerRepo repo.ProviderRepository,
	feedbackRepo repo.FeedbackRepository,
	fetchers datasources.Registry,
) (*Bot, error) {
	if cfg.Token == "" {
		return nil, errors.New("MA_TELEGRAM_TOKEN is required")
	}
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		OnError: func(err error, c tele.Context) {
			var username string
			var chatID int64
			if c.Sender() != nil {
				username = c.Sender().Username
			}
			if c.Chat() != nil {
				chatID = c.Chat().ID
			}
			if strings.Contains(err.Error(), "message is not modified") {
				slog.Warn("telegram handler", "error", err, "chat_id", chatID, "username", username)
				return
			}
			slog.Error("telegram handler error", "error", err, "chat_id", chatID, "username", username)
		},
	})
	if err != nil {
		return nil, err
	}
	otelMW, err := middleware.NewOtel()
	if err != nil {
		return nil, err
	}
	b.Use(otelMW.Handle)
	h := handlers.New(state.NewStore(), userRepo, meterRepo, providerRepo, feedbackRepo, fetchers)
	registerHandlers(b, h)
	return &Bot{b: b}, nil
}

func (bot *Bot) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		bot.b.Stop()
	}()
	go bot.b.Start()
	slog.InfoContext(ctx, "telegram bot connected")
}

func registerHandlers(b *tele.Bot, h *handlers.Handlers) {
	b.Handle("/start", h.OnStart)

	b.Handle(&tele.InlineButton{Unique: keyboards.UniqAddMeter}, h.OnAddMeter)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqMyMeters}, h.OnMyMeters)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqHelp}, h.OnHelp)

	b.Handle(&tele.InlineButton{Unique: keyboards.UniqProvider}, h.OnProvider)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqSkip}, h.OnSkip)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqCancel}, h.OnCancel)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqNotifyMode}, h.OnNotifyMode)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqConfirm}, h.OnConfirm)

	b.Handle(&tele.InlineButton{Unique: keyboards.UniqMeterSelect}, h.OnMeterSelect)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqMeterEditThreshold}, h.OnMeterEditThreshold)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqMeterDelete}, h.OnMeterDelete)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqMeterDeleteConfirm}, h.OnMeterDeleteConfirm)

	b.Handle(&tele.InlineButton{Unique: keyboards.UniqFeedback}, h.OnFeedback)

	b.Handle(&tele.InlineButton{Unique: keyboards.UniqNavMain}, h.OnNavMain)
	b.Handle(&tele.InlineButton{Unique: keyboards.UniqNavMeters}, h.OnNavMeters)

	b.Handle(tele.OnText, h.OnText)
}
