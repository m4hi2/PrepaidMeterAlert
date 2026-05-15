package handlers

import (
	"log/slog"

	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	"github.com/m4hi2/MeterAlertBot/internal/tgbot/state"
	tele "gopkg.in/telebot.v3"
)

func (h *Handlers) OnText(c tele.Context) error {
	ctx := teleCtx(c)
	conv, ok := h.state.Get(c.Sender().ID)

	step := "idle"
	if ok && conv.Step != state.StepIdle {
		step = string(conv.Step)
	}
	slog.InfoContext(ctx, "tg: text input",
		"step", step,
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"input", c.Text(),
	)

	if !ok || conv.Step == state.StepIdle {
		return c.Send("Use the menu to interact with me.", keyboards.MainMenu())
	}
	switch conv.Step {
	case state.StepAddNumber:
		return h.handleAddNumber(c, conv)
	case state.StepAddAccount:
		return h.handleAddAccount(c, conv)
	case state.StepAddNickname:
		return h.handleAddNickname(c, conv)
	case state.StepAddThreshold:
		return h.handleAddThreshold(c, conv)
	case state.StepEditThreshold:
		return h.handleEditThreshold(c, conv)
	case state.StepFeedback:
		return h.handleFeedback(c, conv)
	default:
		return c.Send("Please use the buttons to continue.", keyboards.CancelOnlyMenu())
	}
}
