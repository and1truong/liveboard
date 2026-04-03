package reminder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryMode controls how fired reminders are retained.
type HistoryMode string

// HistoryMode values control how fired reminders are retained.
const (
	HistoryKeepAll    HistoryMode = "keep_all"
	HistoryPurgeOnAck HistoryMode = "purge_on_ack"
	HistoryAutoPurge  HistoryMode = "auto_purge_30d"
)

// ReminderType distinguishes card-level from board-level reminders.
type ReminderType string //nolint:revive // stutter is acceptable for clarity

// ReminderType values.
const (
	ReminderTypeCard  ReminderType = "card"
	ReminderTypeBoard ReminderType = "board"
)

// ReminderMode distinguishes how the fire time is determined.
type ReminderMode string //nolint:revive // stutter is acceptable for clarity

// ReminderMode values.
const (
	ModeRelative  ReminderMode = "relative"
	ModeAbsolute  ReminderMode = "absolute"
	ModeRecurring ReminderMode = "recurring"
)

// Recurrence describes a simple repeating schedule.
type Recurrence struct {
	Frequency  string `json:"frequency"`              // "daily", "weekly", "monthly"
	Day        string `json:"day,omitempty"`          // e.g. "monday" (for weekly)
	DayOfMonth int    `json:"day_of_month,omitempty"` // e.g. 1 (for monthly)
	Time       string `json:"time"`                   // "09:00" (HH:MM)
}

// Reminder is a single scheduled reminder.
type Reminder struct {
	ID             string       `json:"id"`
	Type           ReminderType `json:"type"`
	BoardSlug      string       `json:"board_slug"`
	CardID         string       `json:"card_id,omitempty"`
	Mode           ReminderMode `json:"mode"`
	RelativeOffset string       `json:"relative_offset,omitempty"` // e.g. "-1d", "1h"
	AbsoluteTime   *time.Time   `json:"absolute_time,omitempty"`
	Recurrence     *Recurrence  `json:"recurrence,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	FireAt         time.Time    `json:"fire_at"`
	LastFired      *time.Time   `json:"last_fired,omitempty"`
	Fired          bool         `json:"fired"`
	Acknowledged   bool         `json:"acknowledged"`
	SnoozedUntil   *time.Time   `json:"snoozed_until,omitempty"`
}

// HistoryEntry records a fired reminder for the history log.
type HistoryEntry struct {
	ID             string     `json:"id"`
	BoardSlug      string     `json:"board_slug"`
	CardID         string     `json:"card_id,omitempty"`
	CardTitle      string     `json:"card_title,omitempty"`
	Message        string     `json:"message,omitempty"`
	FiredAt        time.Time  `json:"fired_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

// StoreData is the JSON structure persisted to settings.reminder.json.
type StoreData struct {
	Enabled     bool           `json:"enabled"`
	Timezone    string         `json:"timezone"`
	HistoryMode HistoryMode    `json:"history_mode"`
	Reminders   []Reminder     `json:"reminders"`
	History     []HistoryEntry `json:"history"`
}

// Store manages reminder persistence with file-level locking.
type Store struct {
	mu  sync.Mutex
	dir string // workspace directory
}

// NewStore creates a Store for the given workspace directory.
func NewStore(workspaceDir string) *Store {
	return &Store{dir: workspaceDir}
}

func (s *Store) filePath() string {
	return filepath.Join(s.dir, "settings.reminder.json")
}

// Load reads the reminder data from disk. Returns empty data if file doesn't exist.
func (s *Store) Load() (*StoreData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *Store) loadUnlocked() (*StoreData, error) {
	data := &StoreData{
		Timezone:    "Local",
		HistoryMode: HistoryAutoPurge,
	}
	raw, err := os.ReadFile(s.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, fmt.Errorf("read reminders: %w", err)
	}
	if err := json.Unmarshal(raw, data); err != nil {
		return nil, fmt.Errorf("parse reminders: %w", err)
	}
	return data, nil
}

// Save writes the reminder data to disk.
func (s *Store) Save(data *StoreData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(data)
}

func (s *Store) saveUnlocked(data *StoreData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal reminders: %w", err)
	}
	return os.WriteFile(s.filePath(), raw, 0644)
}

// Mutate loads, applies fn, and saves atomically (under lock).
func (s *Store) Mutate(fn func(*StoreData) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if err := fn(data); err != nil {
		return err
	}
	return s.saveUnlocked(data)
}

// AddReminder adds a reminder to the store.
func (s *Store) AddReminder(r Reminder) error {
	return s.Mutate(func(d *StoreData) error {
		// Replace existing reminder for same card if card-level
		if r.Type == ReminderTypeCard && r.CardID != "" {
			for i, existing := range d.Reminders {
				if existing.Type == ReminderTypeCard && existing.CardID == r.CardID && existing.BoardSlug == r.BoardSlug {
					d.Reminders[i] = r
					return nil
				}
			}
		}
		d.Reminders = append(d.Reminders, r)
		return nil
	})
}

// RemoveReminder deletes a reminder by ID.
func (s *Store) RemoveReminder(id string) error {
	return s.Mutate(func(d *StoreData) error {
		for i, r := range d.Reminders {
			if r.ID == id {
				d.Reminders = append(d.Reminders[:i], d.Reminders[i+1:]...)
				return nil
			}
		}
		return nil
	})
}

// RemoveByCardID removes all reminders for a specific card.
func (s *Store) RemoveByCardID(boardSlug, cardID string) error {
	return s.Mutate(func(d *StoreData) error {
		var kept []Reminder
		for _, r := range d.Reminders {
			if r.Type != ReminderTypeCard || r.CardID != cardID || r.BoardSlug != boardSlug {
				kept = append(kept, r)
			}
		}
		d.Reminders = kept
		return nil
	})
}

// AcknowledgeReminder marks a reminder as acknowledged.
func (s *Store) AcknowledgeReminder(id string) error {
	return s.Mutate(func(d *StoreData) error {
		for i, r := range d.Reminders {
			if r.ID == id {
				d.Reminders[i].Acknowledged = true
				now := time.Now()
				// Move to history
				entry := HistoryEntry{
					ID:             r.ID,
					BoardSlug:      r.BoardSlug,
					CardID:         r.CardID,
					FiredAt:        r.FireAt,
					AcknowledgedAt: &now,
				}
				d.History = append(d.History, entry)
				// Remove from active if not recurring
				if r.Mode != ModeRecurring {
					d.Reminders = append(d.Reminders[:i], d.Reminders[i+1:]...)
				} else {
					d.Reminders[i].Fired = false
					d.Reminders[i].Acknowledged = false
				}
				return nil
			}
		}
		return nil
	})
}

// SnoozeReminder defers a fired reminder by the given duration.
func (s *Store) SnoozeReminder(id string, duration time.Duration) error {
	return s.Mutate(func(d *StoreData) error {
		for i, r := range d.Reminders {
			if r.ID == id {
				snoozed := time.Now().Add(duration)
				d.Reminders[i].SnoozedUntil = &snoozed
				d.Reminders[i].FireAt = snoozed
				d.Reminders[i].Fired = false
				d.Reminders[i].Acknowledged = false
				return nil
			}
		}
		return nil
	})
}

// ClearFired removes all fired (non-recurring) reminders and optionally moves them to history.
func (s *Store) ClearFired() error {
	return s.Mutate(func(d *StoreData) error {
		var kept []Reminder
		now := time.Now()
		for _, r := range d.Reminders {
			if r.Fired && r.Mode != ModeRecurring {
				d.History = append(d.History, HistoryEntry{
					ID:             r.ID,
					BoardSlug:      r.BoardSlug,
					CardID:         r.CardID,
					FiredAt:        r.FireAt,
					AcknowledgedAt: &now,
				})
			} else {
				kept = append(kept, r)
			}
		}
		d.Reminders = kept
		return nil
	})
}

// ClearHistory removes all history entries.
func (s *Store) ClearHistory() error {
	return s.Mutate(func(d *StoreData) error {
		d.History = nil
		return nil
	})
}

// PurgeOldHistory removes history entries older than the given duration.
func (s *Store) PurgeOldHistory(maxAge time.Duration) error {
	return s.Mutate(func(d *StoreData) error {
		cutoff := time.Now().Add(-maxAge)
		var kept []HistoryEntry
		for _, h := range d.History {
			if h.FiredAt.After(cutoff) {
				kept = append(kept, h)
			}
		}
		d.History = kept
		return nil
	})
}

// FindReminderForCard returns the active reminder for a card, if any.
func (s *Store) FindReminderForCard(boardSlug, cardID string) *Reminder {
	data, err := s.Load()
	if err != nil {
		return nil
	}
	for _, r := range data.Reminders {
		if r.Type == ReminderTypeCard && r.CardID == cardID && r.BoardSlug == boardSlug {
			return &r
		}
	}
	return nil
}

// RecalculateRelativeReminder updates fire_at for a relative reminder when due date changes.
func (s *Store) RecalculateRelativeReminder(boardSlug, cardID, newDueDate, timezone string) error {
	return s.Mutate(func(d *StoreData) error {
		for i, r := range d.Reminders {
			if r.Type == ReminderTypeCard && r.CardID == cardID && r.BoardSlug == boardSlug && r.Mode == ModeRelative {
				fireAt, err := ComputeFireAt(r.RelativeOffset, newDueDate, timezone)
				if err != nil {
					return err
				}
				d.Reminders[i].FireAt = fireAt
				d.Reminders[i].Fired = false
				d.Reminders[i].Acknowledged = false
				d.Reminders[i].SnoozedUntil = nil
			}
		}
		return nil
	})
}
