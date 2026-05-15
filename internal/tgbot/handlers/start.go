package handlers

import (
	"fmt"
	"log/slog"

	"github.com/m4hi2/MeterAlertBot/internal/tgbot/keyboards"
	tele "gopkg.in/telebot.v3"
)

func (h *Handlers) OnStart(c tele.Context) error {
	ctx := teleCtx(c)
	if _, err := h.getOrCreateUser(ctx, c.Sender()); err != nil {
		return err
	}
	h.state.Clear(c.Sender().ID)
	slog.InfoContext(ctx, "user started bot",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)
	return c.Send(
		fmt.Sprintf("👋 Hello, %s!\n\nI'll alert you when your prepaid meter balance falls below your threshold.", c.Sender().FirstName),
		keyboards.MainMenu(),
	)
}

func (h *Handlers) OnHelp(c tele.Context) error {
	ctx := teleCtx(c)
	slog.InfoContext(ctx, "user opened help",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)
	text := "ℹ️ *PrepaidMeter Alert Bot*\n\n" +
		"• *Add Meter* — Register a prepaid meter to monitor\n" +
		"• *My Meters* — View and manage your registered meters\n\n" +
		"You'll receive an alert when a meter's balance drops below the threshold you set."
	return c.Edit(text, tele.ModeMarkdown, keyboards.MainMenu())
}

func (h *Handlers) OnCancel(c tele.Context) error {
	ctx := teleCtx(c)
	conv, _ := h.state.Get(c.Sender().ID)
	slog.InfoContext(ctx, "user cancelled",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
		"from_step", string(conv.Step),
	)
	h.state.Clear(c.Sender().ID)
	return c.Edit("Cancelled. What would you like to do?", keyboards.MainMenu())
}

func (h *Handlers) OnNavMain(c tele.Context) error {
	ctx := teleCtx(c)
	slog.InfoContext(ctx, "user navigated to main menu",
		"username", c.Sender().Username,
		"chat_id", c.Chat().ID,
	)
	h.state.Clear(c.Sender().ID)
	return c.Edit("Choose an option:", keyboards.MainMenu())
}
