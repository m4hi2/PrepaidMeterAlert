package handlers

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/state"
	tele "gopkg.in/telebot.v3"
)

func (h *Handlers) OnMyMeters(c tele.Context) error {
	ctx := teleCtx(c)
	slog.InfoContext(ctx, "user opened meter list",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)
	user, err := h.getOrCreateUser(ctx, c.Sender())
	if err != nil {
		return err
	}
	return h.showMeterList(c, user, true)
}

func (h *Handlers) OnNavMeters(c tele.Context) error {
	ctx := teleCtx(c)
	slog.InfoContext(ctx, "user navigated to meter list",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)
	user, err := h.getOrCreateUser(ctx, c.Sender())
	if err != nil {
		return err
	}
	return h.showMeterList(c, user, true)
}

func (h *Handlers) showMeterList(c tele.Context, user *models.User, edit bool) error {
	meters, err := h.meterRepo.GetByUserID(teleCtx(c), user.ID)
	if err != nil {
		return err
	}
	if len(meters) == 0 {
		text := "You have no meters registered yet."
		m := &tele.ReplyMarkup{}
		m.Inline(
			m.Row(m.Data("➕ Add Meter", keyboards.UniqAddMeter)),
			m.Row(m.Data("🏠 Main Menu", keyboards.UniqNavMain)),
		)
		if edit {
			return c.Edit(text, m)
		}
		return c.Send(text, m)
	}

	btns := make([]keyboards.MeterButton, len(meters))
	for i, mt := range meters {
		label := meterLabel(mt)
		btns[i] = keyboards.MeterButton{ID: mt.ID.String(), Label: label}
	}
	text := "📋 *Your meters* — tap one to manage it:"
	if edit {
		return c.Edit(text, tele.ModeMarkdown, keyboards.MeterListMenu(btns))
	}
	return c.Send(text, tele.ModeMarkdown, keyboards.MeterListMenu(btns))
}

func (h *Handlers) OnMeterSelect(c tele.Context) error {
	ctx := teleCtx(c)
	id, err := uuid.Parse(c.Data())
	if err != nil {
		return c.Edit("Invalid meter. Please go back and try again.", keyboards.MainMenu())
	}
	slog.InfoContext(ctx, "user viewing meter detail",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"meter_id", id.String(),
	)
	meter, err := h.meterRepo.GetByID(ctx, id)
	if err != nil {
		return c.Edit("Meter not found.", keyboards.MainMenu())
	}
	return c.Edit(meterDetail(meter), tele.ModeMarkdown, keyboards.MeterActionsMenu(id.String()))
}

func (h *Handlers) OnMeterEditThreshold(c tele.Context) error {
	ctx := teleCtx(c)
	id, err := uuid.Parse(c.Data())
	if err != nil {
		return c.Edit("Invalid meter.", keyboards.MainMenu())
	}
	meter, err := h.meterRepo.GetByID(ctx, id)
	if err != nil {
		return c.Edit("Meter not found.", keyboards.MainMenu())
	}
	slog.InfoContext(ctx, "user editing meter threshold",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"meter_id", id.String(),
	)
	h.state.Set(c.Sender().ID, state.Conversation{
		Step:    state.StepEditThreshold,
		MeterID: id.String(),
	})
	return c.Send(
		fmt.Sprintf("Enter new threshold amount in BDT (current: *%.0f BDT*):", meter.Threshold),
		tele.ModeMarkdown,
		keyboards.CancelOnlyMenu(),
	)
}

func (h *Handlers) handleEditThreshold(c tele.Context, conv state.Conversation) error {
	ctx := teleCtx(c)
	val, err := strconv.ParseFloat(c.Text(), 64)
	if err != nil || val <= 0 {
		return c.Send("Please enter a valid positive number (e.g. 200):", keyboards.CancelOnlyMenu())
	}
	id, err := uuid.Parse(conv.MeterID)
	if err != nil {
		h.state.Clear(c.Sender().ID)
		return c.Send("Something went wrong. Please try again.", keyboards.MainMenu())
	}
	meter, err := h.meterRepo.GetByID(ctx, id)
	if err != nil {
		h.state.Clear(c.Sender().ID)
		return c.Send("Meter not found.", keyboards.MainMenu())
	}
	meter.Threshold = val
	if err := h.meterRepo.Update(ctx, meter); err != nil {
		h.state.Clear(c.Sender().ID)
		return fmt.Errorf("update threshold: %w", err)
	}
	slog.InfoContext(ctx, "meter threshold updated",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"meter_id", id.String(),
		"threshold", val,
	)
	h.state.Clear(c.Sender().ID)
	return c.Send(
		fmt.Sprintf("✅ Threshold updated.\n\n%s", meterDetail(meter)),
		tele.ModeMarkdown,
		keyboards.MeterActionsMenu(id.String()),
	)
}

func (h *Handlers) OnMeterDelete(c tele.Context) error {
	ctx := teleCtx(c)
	id, err := uuid.Parse(c.Data())
	if err != nil {
		return c.Edit("Invalid meter.", keyboards.MainMenu())
	}
	slog.InfoContext(ctx, "user initiated meter delete",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"meter_id", id.String(),
	)
	meter, err := h.meterRepo.GetByID(ctx, id)
	if err != nil {
		return c.Edit("Meter not found.", keyboards.MainMenu())
	}
	return c.Edit(
		fmt.Sprintf("Are you sure you want to delete meter *%s*?", meterLabel(meter)),
		tele.ModeMarkdown,
		keyboards.DeleteConfirmMenu(id.String()),
	)
}

func (h *Handlers) OnMeterDeleteConfirm(c tele.Context) error {
	ctx := teleCtx(c)
	id, err := uuid.Parse(c.Data())
	if err != nil {
		return c.Edit("Invalid meter.", keyboards.MainMenu())
	}
	if err := h.meterRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete meter: %w", err)
	}
	slog.InfoContext(ctx, "meter deleted",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"meter_id", id.String(),
	)
	trace.SpanFromContext(ctx).AddEvent("meter.deleted", trace.WithAttributes(
		attribute.String("meter_id", id.String()),
	))
	user, err := h.getOrCreateUser(ctx, c.Sender())
	if err != nil {
		return err
	}
	return h.showMeterList(c, user, true)
}

func meterLabel(m *models.Meter) string {
	provider := strings.ToUpper(string(m.ProviderCode))
	name := m.AccountNumber
	if m.MeterNumber != "" {
		name = m.MeterNumber
	}
	if m.Nickname != "" {
		name = m.Nickname
	}
	return fmt.Sprintf("%s - %s (%.0f BDT)", provider, name, m.Balance)
}

func meterDetail(m *models.Meter) string {
	provider := strings.ToUpper(string(m.ProviderCode))
	lastChecked := "never"
	if m.LastFetchAt != nil {
		lastChecked = m.LastFetchAt.Format("02 Jan 2006 15:04")
	}
	meterDisplay := m.MeterNumber
	if meterDisplay == "" {
		meterDisplay = "(not provided)"
	}
	nicknameDisplay := m.Nickname
	if nicknameDisplay == "" {
		nicknameDisplay = "(none)"
	}
	return fmt.Sprintf(
		"⚡ *%s Meter*\n━━━━━━━━━━━━━━\n"+
			"Account #: %s\n"+
			"Meter #: %s\n"+
			"Nickname: %s\n"+
			"Balance: *%.2f BDT*\n"+
			"Threshold: %.0f BDT\n"+
			"Alert mode: %s\n"+
			"Last checked: %s",
		provider,
		m.AccountNumber,
		meterDisplay,
		nicknameDisplay,
		m.Balance,
		m.Threshold,
		string(m.NotifyMode),
		lastChecked,
	)
}
