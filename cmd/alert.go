package cmd

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/alerter"
	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/database"
	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/database/repo"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/m4hi2/MeterAlertBot/internal/datasources/desco"
	"github.com/m4hi2/MeterAlertBot/internal/datasources/nesco"
	"github.com/m4hi2/MeterAlertBot/internal/telemetry"
	"github.com/muesli/coral"
	"golang.org/x/time/rate"
	tele "gopkg.in/telebot.v3"
)

var alertCmd = &coral.Command{
	Use:   "alert",
	Short: "Fetch meter balances and send low-balance notifications",
	RunE:  runAlert,
}

func runAlert(cmd *coral.Command, _ []string) error {
	cfg := config.Get()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Telemetry.Enabled {
		shutdown, err := telemetry.Setup(ctx, telemetry.Config{
			Endpoint:    cfg.Telemetry.OTLPEndpoint,
			ServiceName: cfg.Telemetry.ServiceName,
			Environment: cfg.Telemetry.Environment,
		})
		if err != nil {
			return err
		}
		defer func() {
			flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := shutdown(flushCtx); err != nil {
				slog.Error("telemetry shutdown error", "error", err)
			}
		}()
		slog.InfoContext(ctx, "telemetry enabled", "endpoint", cfg.Telemetry.OTLPEndpoint)
	}

	db, err := database.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	bunHook, err := telemetry.NewBunHook()
	if err != nil {
		return err
	}
	db.AddQueryHook(bunHook)

	userRepo := repo.NewUserRepo(db)
	meterRepo := repo.NewMeterRepo(db)
	providerRepo := repo.NewProviderRepo(db)
	notifLogRepo := repo.NewNotificationLogRepo(db)

	registry := datasources.Registry{
		models.ProviderCodeDESCO: desco.NewService(cfg.Desco),
		models.ProviderCodeNESCO: nesco.NewService(cfg.Nesco),
	}

	if cfg.Telegram.Token == "" {
		return errors.New("MA_TELEGRAM_TOKEN is required")
	}
	bot, err := tele.NewBot(tele.Settings{Token: cfg.Telegram.Token})
	if err != nil {
		return err
	}

	tgLimiter := rate.NewLimiter(rate.Limit(cfg.Telegram.RateLimit), int(cfg.Telegram.RateLimit))

	a, err := alerter.New(meterRepo, userRepo, notifLogRepo, providerRepo, registry, bot, tgLimiter)
	if err != nil {
		return err
	}

	return a.Run(ctx)
}
