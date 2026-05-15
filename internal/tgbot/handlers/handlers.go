package handlers

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/database/repo"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/state"
	tele "gopkg.in/telebot.v3"
)

// otelCtxKey matches the key set by the OTel middleware.
const otelCtxKey = "otel_ctx"

// teleCtx extracts the OTel-enriched context injected by the middleware.
// Falls back to context.Background() so handlers work without the middleware.
func teleCtx(c tele.Context) context.Context {
	if ctx, ok := c.Get(otelCtxKey).(context.Context); ok {
		return ctx
	}
	return context.Background()
}

type Handlers struct {
	state        *state.Store
	userRepo     repo.UserRepository
	meterRepo    repo.MeterRepository
	providerRepo repo.ProviderRepository
	feedbackRepo repo.FeedbackRepository
	fetchers     datasources.Registry
}

func New(
	st *state.Store,
	userRepo repo.UserRepository,
	meterRepo repo.MeterRepository,
	providerRepo repo.ProviderRepository,
	feedbackRepo repo.FeedbackRepository,
	fetchers datasources.Registry,
) *Handlers {
	return &Handlers{
		state:        st,
		userRepo:     userRepo,
		meterRepo:    meterRepo,
		providerRepo: providerRepo,
		feedbackRepo: feedbackRepo,
		fetchers:     fetchers,
	}
}

func (h *Handlers) getOrCreateUser(ctx context.Context, sender *tele.User) (*models.User, error) {
	platformID := strconv.FormatInt(sender.ID, 10)
	user, err := h.userRepo.GetByPlatformID(ctx, models.PlatformTelegram, platformID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	user = &models.User{
		Platform:   models.PlatformTelegram,
		PlatformID: platformID,
		FirstName:  sender.FirstName,
		LastName:   sender.LastName,
		Username:   sender.Username,
		IsBot:      sender.IsBot,
	}
	return user, h.userRepo.Create(ctx, user)
}
