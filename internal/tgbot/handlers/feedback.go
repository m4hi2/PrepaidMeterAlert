package handlers

import (
	"strconv"
	"strings"

	"github.com/m4hi2/MeterAlertBot/internal/database/models"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/state"
	tele "gopkg.in/telebot.v3"
)

func (h *Handlers) OnFeedback(c tele.Context) error {
	h.state.Set(c.Sender().ID, state.Conversation{Step: state.StepFeedback})
	return c.Edit("💬 Please type your feedback:", keyboards.CancelOnlyMenu())
}

func (h *Handlers) handleFeedback(c tele.Context, conv state.Conversation) error {
	text := strings.TrimSpace(c.Text())
	if text == "" {
		return c.Send("Please type your feedback:", keyboards.CancelOnlyMenu())
	}
	fb := &models.Feedback{
		Platform:   models.PlatformTelegram,
		PlatformID: strconv.FormatInt(c.Sender().ID, 10),
		Text:       text,
	}
	if err := h.feedbackRepo.Create(teleCtx(c), fb); err != nil {
		return c.Send("Failed to save feedback. Please try again.", keyboards.CancelOnlyMenu())
	}
	h.state.Clear(c.Sender().ID)
	return c.Send("✅ Thank you for your feedback!", keyboards.MainMenu())
}
