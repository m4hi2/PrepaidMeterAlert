package keyboards

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

const (
	UniqAddMeter           = "add_meter"
	UniqMyMeters           = "my_meters"
	UniqHelp               = "help"
	UniqProvider           = "provider"
	UniqNotifyMode         = "notify_mode"
	UniqConfirm            = "confirm"
	UniqSkip               = "skip"
	UniqCancel             = "cancel"
	UniqMeterSelect        = "meter_select"
	UniqMeterEditThreshold = "meter_edit_threshold"
	UniqMeterDelete        = "meter_delete"
	UniqMeterDeleteConfirm = "meter_delete_confirm"
	UniqNavMain            = "nav_main"
	UniqNavMeters          = "nav_meters"
	UniqFeedback           = "feedback"
)

func MainMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(
			m.Data("➕ Add Meter", UniqAddMeter),
			m.Data("📋 My Meters", UniqMyMeters),
		),
		m.Row(m.Data("💬 Feedback", UniqFeedback)),
		m.Row(m.Data("❓ Help", UniqHelp)),
	)
	return m
}

func ProviderMenu(providers []string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(providers)+1)
	for _, p := range providers {
		rows = append(rows, m.Row(m.Data(strings.ToUpper(p), UniqProvider, p)))
	}
	rows = append(rows, m.Row(m.Data("❌ Cancel", UniqCancel)))
	m.Inline(rows...)
	return m
}

func SkipOrCancelMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(
		m.Data("⏭ Skip", UniqSkip),
		m.Data("❌ Cancel", UniqCancel),
	))
	return m
}

func CancelOnlyMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(m.Data("❌ Cancel", UniqCancel)))
	return m
}

func NotifyModeMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(
			m.Data("Single Alert", UniqNotifyMode, "single"),
			m.Data("Daily Alert", UniqNotifyMode, "daily"),
		),
		m.Row(m.Data("❌ Cancel", UniqCancel)),
	)
	return m
}

func ConfirmMenu() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(
		m.Data("✅ Confirm", UniqConfirm, "yes"),
		m.Data("❌ Cancel", UniqConfirm, "no"),
	))
	return m
}

type MeterButton struct {
	ID    string
	Label string
}

func MeterListMenu(meters []MeterButton) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(meters)+1)
	for _, mb := range meters {
		rows = append(rows, m.Row(m.Data(mb.Label, UniqMeterSelect, mb.ID)))
	}
	rows = append(rows, m.Row(m.Data("🏠 Main Menu", UniqNavMain)))
	m.Inline(rows...)
	return m
}

func MeterActionsMenu(meterID string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("✏️ Edit Threshold", UniqMeterEditThreshold, meterID)),
		m.Row(m.Data("🗑️ Delete", UniqMeterDelete, meterID)),
		m.Row(
			m.Data("← Back", UniqNavMeters),
			m.Data("🏠 Main Menu", UniqNavMain),
		),
	)
	return m
}

func DeleteConfirmMenu(meterID string) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(
		m.Data("Yes, Delete", UniqMeterDeleteConfirm, meterID),
		m.Data("Keep it", UniqNavMeters),
	))
	return m
}
