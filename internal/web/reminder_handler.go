package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/and1truong/liveboard/internal/reminder"
)

// ReminderPageModel is the template data for the /reminders page.
type ReminderPageModel struct {
	LayoutSettings
	Title          string
	SiteName       string
	Boards         []BoardSummary
	AllTags        []string
	BoardSlug      string
	Pending        []ReminderView
	Fired          []ReminderView
	History        []HistoryView
	ReminderEnabled bool
	Timezone       string
	HistoryMode    string
}

// ReminderView is a template-friendly representation of a reminder.
type ReminderView struct {
	ID             string
	Type           string
	BoardSlug      string
	CardID         string
	CardTitle      string
	BoardName      string
	Mode           string
	RelativeOffset string
	FireAt         string
	FireAtRel      string // "in 2 hours", "3 days ago"
	Fired          bool
	Recurring      bool
	Recurrence     string // human-readable recurrence
}

// HistoryView is a template-friendly representation of a history entry.
type HistoryView struct {
	ID             string
	BoardSlug      string
	CardTitle      string
	Message        string
	FiredAt        string
	AcknowledgedAt string
}

// RemindersPage handles GET /reminders.
func (h *Handler) RemindersPage(w http.ResponseWriter, _ *http.Request) {
	settings := h.loadSettings()
	infos, _ := h.ws.ListBoardSummaries()
	summaries := sortBoardsWithPins(toBoardSummariesFast(infos), settings.PinnedBoards)

	data, err := h.ReminderStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build board name lookup
	boardNames := make(map[string]string)
	for _, s := range summaries {
		boardNames[s.Slug] = s.Name
	}

	// Build card title lookup for card reminders
	cardTitles := make(map[string]string)
	for _, r := range data.Reminders {
		if r.Type == reminder.ReminderTypeCard && r.CardID != "" {
			if title := h.findCardTitle(r.BoardSlug, r.CardID); title != "" {
				cardTitles[r.CardID] = title
			}
		}
	}

	now := time.Now()
	var pending, fired []ReminderView
	for _, r := range data.Reminders {
		rv := ReminderView{
			ID:             r.ID,
			Type:           string(r.Type),
			BoardSlug:      r.BoardSlug,
			CardID:         r.CardID,
			CardTitle:      cardTitles[r.CardID],
			BoardName:      boardNames[r.BoardSlug],
			Mode:           string(r.Mode),
			RelativeOffset: r.RelativeOffset,
			FireAt:         r.FireAt.Format("2006-01-02 15:04"),
			FireAtRel:      reminderRelTime(r.FireAt, now),
			Fired:          r.Fired,
			Recurring:      r.Mode == reminder.ModeRecurring,
		}
		if r.Recurrence != nil {
			rv.Recurrence = formatRecurrence(r.Recurrence)
		}
		if r.Fired {
			fired = append(fired, rv)
		} else {
			pending = append(pending, rv)
		}
	}

	// Sort pending by fire time ascending
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].FireAt < pending[j].FireAt
	})

	var history []HistoryView
	for _, h := range data.History {
		hv := HistoryView{
			ID:        h.ID,
			BoardSlug: h.BoardSlug,
			CardTitle: h.CardTitle,
			Message:   h.Message,
			FiredAt:   h.FiredAt.Format("2006-01-02 15:04"),
		}
		if h.AcknowledgedAt != nil {
			hv.AcknowledgedAt = h.AcknowledgedAt.Format("2006-01-02 15:04")
		}
		history = append(history, hv)
	}

	model := ReminderPageModel{
		LayoutSettings:  h.layoutSettings(settings),
		Title:           "Reminders — " + settings.SiteName,
		SiteName:        settings.SiteName,
		Boards:          summaries,
		AllTags:         collectAllTags(summaries),
		BoardSlug:       "__reminders__",
		Pending:         pending,
		Fired:           fired,
		History:         history,
		ReminderEnabled: settings.ReminderEnabled,
		Timezone:        settings.ReminderTimezone,
		HistoryMode:     string(data.HistoryMode),
	}

	renderFullPage(w, h.reminderPageTpl, model)
}

// HandleSetReminder handles POST /reminders/set.
func (h *Handler) HandleSetReminder(w http.ResponseWriter, r *http.Request) {
	boardSlug := r.FormValue("board_slug")
	cardID := r.FormValue("card_id")
	reminderType := r.FormValue("type") // "card" or "board"
	mode := r.FormValue("mode")         // "relative", "absolute", "recurring"
	offset := r.FormValue("offset")     // "-1d", etc.
	absoluteTime := r.FormValue("absolute_time")
	dueDate := r.FormValue("due_date")

	settings := h.loadSettings()
	tz := settings.ReminderTimezone
	if tz == "" {
		tz = "Local"
	}

	rem := reminder.Reminder{
		ID:        reminder.GenerateID(),
		Type:      reminder.ReminderType(reminderType),
		BoardSlug: boardSlug,
		CardID:    cardID,
		Mode:      reminder.ReminderMode(mode),
		CreatedAt: time.Now(),
	}

	switch reminder.ReminderMode(mode) {
	case reminder.ModeRelative:
		rem.RelativeOffset = offset
		fireAt, err := reminder.ComputeFireAt(offset, dueDate, tz)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rem.FireAt = fireAt
	case reminder.ModeAbsolute:
		t, err := time.Parse("2006-01-02T15:04", absoluteTime)
		if err != nil {
			http.Error(w, "invalid absolute_time", http.StatusBadRequest)
			return
		}
		loc, _ := time.LoadLocation(tz)
		if loc == nil {
			loc = time.Local
		}
		rem.AbsoluteTime = &t
		rem.FireAt = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	case reminder.ModeRecurring:
		var rec reminder.Recurrence
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			// Try form values
			rec.Frequency = r.FormValue("frequency")
			rec.Day = r.FormValue("day")
			rec.Time = r.FormValue("time")
		}
		rem.Recurrence = &rec
		next, err := reminder.NextRecurrence(&rec, time.Now(), tz)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rem.FireAt = next
	}

	if err := h.ReminderStore.AddReminder(rem); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success — HTMX will handle the response
	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// HandleDismissReminder handles POST /reminders/dismiss/{id}.
func (h *Handler) HandleDismissReminder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.ReminderStore.AcknowledgeReminder(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// HandleSnoozeReminder handles POST /reminders/snooze/{id}.
func (h *Handler) HandleSnoozeReminder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	durationStr := r.FormValue("duration") // "15m", "1h", "1d", "tomorrow"

	var d time.Duration
	switch durationStr {
	case "15m":
		d = 15 * time.Minute
	case "1h":
		d = time.Hour
	case "1d", "tomorrow":
		d = 24 * time.Hour
	default:
		dur, err := time.ParseDuration(durationStr)
		if err != nil {
			http.Error(w, "invalid duration", http.StatusBadRequest)
			return
		}
		d = dur
	}

	if err := h.ReminderStore.SnoozeReminder(id, d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// HandleDeleteReminder handles DELETE /reminders/{id}.
func (h *Handler) HandleDeleteReminder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.ReminderStore.RemoveReminder(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// HandleClearFired handles POST /reminders/clear-fired.
func (h *Handler) HandleClearFired(w http.ResponseWriter, _ *http.Request) {
	if err := h.ReminderStore.ClearFired(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// HandleClearHistory handles POST /reminders/clear-history.
func (h *Handler) HandleClearHistory(w http.ResponseWriter, _ *http.Request) {
	if err := h.ReminderStore.ClearHistory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// HandleUpdateReminderSettings handles POST /reminders/settings.
func (h *Handler) HandleUpdateReminderSettings(w http.ResponseWriter, r *http.Request) {
	enabled := r.FormValue("enabled") == "true"
	tz := r.FormValue("timezone")
	historyMode := r.FormValue("history_mode")

	// Update app settings
	settings := h.loadSettings()
	settings.ReminderEnabled = enabled
	if tz != "" {
		settings.ReminderTimezone = tz
	}
	if err := h.saveSettings(settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update reminder store settings
	if historyMode != "" {
		_ = h.ReminderStore.Mutate(func(d *reminder.StoreData) error {
			d.Enabled = enabled
			d.Timezone = tz
			d.HistoryMode = reminder.HistoryMode(historyMode)
			return nil
		})
	}

	w.Header().Set("HX-Trigger", "reminder-updated")
	w.WriteHeader(http.StatusOK)
}

// findCardTitle looks up a card title by board slug and card ID.
func (h *Handler) findCardTitle(boardSlug, cardID string) string {
	b, err := h.ws.LoadBoard(boardSlug)
	if err != nil {
		return ""
	}
	for _, col := range b.Columns {
		for _, card := range col.Cards {
			if card.Metadata != nil && card.Metadata["id"] == cardID {
				return card.Title
			}
		}
	}
	return ""
}

func reminderRelTime(t time.Time, now time.Time) string {
	diff := time.Until(t)
	if diff < 0 {
		diff = -diff
		if diff < time.Minute {
			return "just now"
		}
		if diff < time.Hour {
			return fmt.Sprintf("%dm ago", int(diff.Minutes()))
		}
		if diff < 24*time.Hour {
			return fmt.Sprintf("%dh ago", int(diff.Hours()))
		}
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	if diff < time.Minute {
		return "now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("in %dm", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("in %dh", int(diff.Hours()))
	}
	return fmt.Sprintf("in %dd", int(diff.Hours()/24))
}

func formatRecurrence(rec *reminder.Recurrence) string {
	switch rec.Frequency {
	case "daily":
		return fmt.Sprintf("Daily at %s", rec.Time)
	case "weekly":
		return fmt.Sprintf("Weekly on %s at %s", rec.Day, rec.Time)
	case "monthly":
		return fmt.Sprintf("Monthly on day %d at %s", rec.DayOfMonth, rec.Time)
	default:
		return rec.Frequency
	}
}
