package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m4hi2/MeterAlertBot/internal/config"
	"github.com/m4hi2/MeterAlertBot/internal/database"
	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/database/repo"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/m4hi2/MeterAlertBot/internal/datasources/desco"
	"github.com/m4hi2/MeterAlertBot/internal/datasources/nesco"
	"github.com/m4hi2/MeterAlertBot/internal/telemetry"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot"
	"github.com/muesli/coral"
)

var serveCmd = &coral.Command{
	Use:   "serve",
	Short: "Run the alert daemon",
	RunE:  runServe,
}

func runServe(cmd *coral.Command, _ []string) error {
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
	feedbackRepo := repo.NewFeedbackRepo(db)
	_ = repo.NewNotificationLogRepo(db)

	fetchers := datasources.Registry{
		models.ProviderCodeDESCO: desco.NewService(cfg.Desco),
		models.ProviderCodeNESCO: nesco.NewService(cfg.Nesco),
	}

	bot, err := tgbot.New(cfg.Telegram, userRepo, meterRepo, providerRepo, feedbackRepo, fetchers)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "meterbot started")
	go bot.Start(ctx)

	<-ctx.Done()
	slog.Info("shutting down")
	return nil
}
