package handlers

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/datasources"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/state"
	tele "gopkg.in/telebot.v3"
)

func (h *Handlers) OnAddMeter(c tele.Context) error {
	ctx := teleCtx(c)
	slog.InfoContext(ctx, "user initiated add meter",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)
	providers, err := h.providerRepo.GetActive(ctx)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return c.Edit("No providers are currently available. Please try again later.", keyboards.MainMenu())
	}
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = string(p)
	}
	h.state.Set(c.Sender().ID, state.Conversation{Step: state.StepAddProvider})
	return c.Edit("Select your provider:", keyboards.ProviderMenu(names))
}

func (h *Handlers) OnProvider(c tele.Context) error {
	ctx := teleCtx(c)
	conv, ok := h.state.Get(c.Sender().ID)
	if !ok || conv.Step != state.StepAddProvider {
		return c.Edit("Session expired. Please start over.", keyboards.MainMenu())
	}
	slog.InfoContext(ctx, "user selected provider",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"provider", c.Data(),
	)
	conv.Draft.Provider = c.Data()
	conv.Step = state.StepAddAccount
	h.state.Set(c.Sender().ID, conv)
	return c.Edit(
		fmt.Sprintf("✅ Provider: *%s*\n\nEnter your account number:", strings.ToUpper(c.Data())),
		tele.ModeMarkdown,
		keyboards.SkipOrCancelMenu(),
	)
}

func (h *Handlers) OnSkip(c tele.Context) error {
	ctx := teleCtx(c)
	conv, ok := h.state.Get(c.Sender().ID)
	if !ok {
		return c.Edit("Session expired. Please start over.", keyboards.MainMenu())
	}
	slog.InfoContext(ctx, "user skipped step",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"step", string(conv.Step),
	)
	switch conv.Step {
	case state.StepAddAccount:
		conv.Draft.AccountNumber = ""
		conv.Step = state.StepAddNumber
		h.state.Set(c.Sender().ID, conv)
		return c.Edit("Enter your meter number:", keyboards.SkipOrCancelMenu())
	case state.StepAddNumber:
		conv.Draft.MeterNumber = ""
		conv.Step = state.StepAddNickname
		h.state.Set(c.Sender().ID, conv)
		return c.Edit("Enter a nickname for this meter (optional):", keyboards.SkipOrCancelMenu())
	case state.StepAddNickname:
		conv.Draft.Nickname = ""
		conv.Step = state.StepAddThreshold
		h.state.Set(c.Sender().ID, conv)
		return c.Edit("Enter the balance threshold in BDT (e.g. 200):", keyboards.CancelOnlyMenu())
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Nothing to skip here."})
	}
}

func (h *Handlers) OnNotifyMode(c tele.Context) error {
	ctx := teleCtx(c)
	conv, ok := h.state.Get(c.Sender().ID)
	if !ok || conv.Step != state.StepAddMode {
		return c.Edit("Session expired. Please start over.", keyboards.MainMenu())
	}
	slog.InfoContext(ctx, "user selected notify mode",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"mode", c.Data(),
	)
	conv.Draft.NotifyMode = c.Data()
	d := conv.Draft

	if fetcher, ok := h.fetchers.Get(models.ProviderCode(d.Provider)); ok {
		bal, err := fetcher.GetBalance(ctx, datasources.Identifier{
			AccountNumber: d.AccountNumber,
			MeterNumber:   d.MeterNumber,
		})
		if err != nil {
			slog.WarnContext(ctx, "meter verification failed",
				"username", c.Sender().Username,
				"provider", d.Provider,
				"error", err,
			)
			h.state.Clear(c.Sender().ID)
			return c.Edit("❌ Could not verify meter. Please check your account/meter number and try again.", keyboards.MainMenu())
		}
		conv.Draft.AccountNumber = bal.Identifier.AccountNumber
		conv.Draft.MeterNumber = bal.Identifier.MeterNumber
		conv.Draft.Balance = bal.Balance
		d = conv.Draft
	}

	conv.Step = state.StepAddConfirm
	h.state.Set(c.Sender().ID, conv)

	meterDisplay := d.MeterNumber
	if meterDisplay == "" {
		meterDisplay = "(not provided)"
	}
	nicknameDisplay := d.Nickname
	if nicknameDisplay == "" {
		nicknameDisplay = "(none)"
	}
	summary := fmt.Sprintf(
		"📋 *Meter Summary*\n\n"+
			"Provider: %s\n"+
			"Account #: %s\n"+
			"Meter #: %s\n"+
			"Nickname: %s\n"+
			"Threshold: %.0f BDT\n"+
			"Alert mode: %s\n",
		strings.ToUpper(d.Provider),
		d.AccountNumber,
		meterDisplay,
		nicknameDisplay,
		d.Threshold,
		d.NotifyMode,
	)
	if d.Balance > 0 {
		summary += fmt.Sprintf("Current balance: *%.2f BDT*\n", d.Balance)
	}
	summary += "\nConfirm adding this meter?"
	return c.Edit(summary, tele.ModeMarkdown, keyboards.ConfirmMenu())
}

func (h *Handlers) OnConfirm(c tele.Context) error {
	ctx := teleCtx(c)
	conv, ok := h.state.Get(c.Sender().ID)
	if !ok || conv.Step != state.StepAddConfirm {
		return c.Edit("Session expired. Please start over.", keyboards.MainMenu())
	}
	if c.Data() != "yes" {
		slog.InfoContext(ctx, "user declined meter confirmation",
			"username", c.Sender().Username,
			"chat_id", c.Chat().ID,
		)
		h.state.Clear(c.Sender().ID)
		return c.Edit("Cancelled. What would you like to do?", keyboards.MainMenu())
	}

	user, err := h.getOrCreateUser(ctx, c.Sender())
	if err != nil {
		return err
	}
	d := conv.Draft
	fetchStatus := models.FetchStatusPending
	var lastFetchAt *time.Time
	if d.Balance > 0 {
		fetchStatus = models.FetchStatusSuccess
		now := time.Now()
		lastFetchAt = &now
	}
	meter := &models.Meter{
		UserID:             user.ID,
		ProviderCode:       models.ProviderCode(d.Provider),
		MeterNumber:        d.MeterNumber,
		AccountNumber:      d.AccountNumber,
		Nickname:           d.Nickname,
		Threshold:          d.Threshold,
		NotifyMode:         models.NotifyMode(d.NotifyMode),
		Balance:            d.Balance,
		LastFetchAt:        lastFetchAt,
		FetchStatus:        fetchStatus,
		NotificationStatus: models.NStatusNotNeeded,
	}
	if err := h.meterRepo.Create(ctx, meter); err != nil {
		h.state.Clear(c.Sender().ID)
		_ = c.Edit("❌ Something went wrong. Please try again.", keyboards.MainMenu())
		return fmt.Errorf("create meter: %w", err)
	}
	h.state.Clear(c.Sender().ID)
	slog.InfoContext(ctx, "meter added",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"provider", d.Provider,
		"account_number", d.AccountNumber,
	)
	trace.SpanFromContext(ctx).AddEvent("meter.added", trace.WithAttributes(
		attribute.String("provider", d.Provider),
		attribute.String("account_number", d.AccountNumber),
	))
	name := d.AccountNumber
	if d.MeterNumber != "" {
		name = d.MeterNumber
	}
	if d.Nickname != "" {
		name = d.Nickname
	}
	return c.Edit(
		fmt.Sprintf("✅ Meter *%s* added! You'll be notified when the balance drops below *%.0f BDT*.", name, d.Threshold),
		tele.ModeMarkdown,
		keyboards.MainMenu(),
	)
}

func (h *Handlers) handleAddAccount(c tele.Context, conv state.Conversation) error {
	num := strings.TrimSpace(c.Text())
	if num == "" || len(num) > 20 {
		return c.Send("Please enter a valid account number (up to 20 characters):", keyboards.CancelOnlyMenu())
	}
	conv.Draft.AccountNumber = num
	conv.Step = state.StepAddNumber
	h.state.Set(c.Sender().ID, conv)
	return c.Send(
		fmt.Sprintf("✅ Account #: *%s*\n\nEnter your meter number (optional):", num),
		tele.ModeMarkdown,
		keyboards.SkipOrCancelMenu(),
	)
}

func (h *Handlers) handleAddNumber(c tele.Context, conv state.Conversation) error {
	num := strings.TrimSpace(c.Text())
	if len(num) > 20 {
		return c.Send("Meter number must be 20 characters or less:", keyboards.SkipOrCancelMenu())
	}
	conv.Draft.MeterNumber = num
	conv.Step = state.StepAddNickname
	h.state.Set(c.Sender().ID, conv)
	return c.Send("Enter a nickname for this meter (optional):", keyboards.SkipOrCancelMenu())
}

func (h *Handlers) handleAddNickname(c tele.Context, conv state.Conversation) error {
	name := strings.TrimSpace(c.Text())
	if len(name) > 30 {
		return c.Send("Nickname must be 30 characters or less:", keyboards.SkipOrCancelMenu())
	}
	conv.Draft.Nickname = name
	conv.Step = state.StepAddThreshold
	h.state.Set(c.Sender().ID, conv)
	return c.Send("Enter the balance threshold in BDT (e.g. 200):", keyboards.CancelOnlyMenu())
}

func (h *Handlers) handleAddThreshold(c tele.Context, conv state.Conversation) error {
	val, err := strconv.ParseFloat(strings.TrimSpace(c.Text()), 64)
	if err != nil || val <= 0 {
		return c.Send("Please enter a valid positive number (e.g. 200):", keyboards.CancelOnlyMenu())
	}
	conv.Draft.Threshold = val
	conv.Step = state.StepAddMode
	h.state.Set(c.Sender().ID, conv)
	return c.Send(
		fmt.Sprintf("✅ Threshold: *%.0f BDT*\n\nSelect notification mode:", val),
		tele.ModeMarkdown,
		keyboards.NotifyModeMenu(),
	)
}
