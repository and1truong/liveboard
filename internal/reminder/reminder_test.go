package reminder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/and1truong/liveboard/pkg/models"
)

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	id2 := GenerateID()
	if id1 == "" || id2 == "" {
		t.Fatal("GenerateID returned empty string")
	}
	if id1 == id2 {
		t.Fatal("GenerateID returned duplicate IDs")
	}
	if len(id1) != 26 { // ULID is 26 chars
		t.Fatalf("expected 26-char ULID, got %d: %s", len(id1), id1)
	}
}

func TestParseOffset(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"15m", 15 * time.Minute},
		{"-15m", 15 * time.Minute},
		{"1h", time.Hour},
		{"-1h", time.Hour},
		{"1d", 24 * time.Hour},
		{"-3d", 3 * 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
	}
	for _, tt := range tests {
		got, err := ParseOffset(tt.input)
		if err != nil {
			t.Errorf("ParseOffset(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseOffset(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseOffsetInvalid(t *testing.T) {
	for _, input := range []string{"", "x", "1", "1z", "abc"} {
		_, err := ParseOffset(input)
		if err == nil {
			t.Errorf("ParseOffset(%q) expected error", input)
		}
	}
}

func TestComputeFireAt(t *testing.T) {
	fireAt, err := ComputeFireAt("-1d", "2026-04-10", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 4, 9, 9, 0, 0, 0, time.UTC)
	if !fireAt.Equal(expected) {
		t.Errorf("ComputeFireAt(-1d, 2026-04-10) = %v, want %v", fireAt, expected)
	}

	// At due time
	fireAt, err = ComputeFireAt("0", "2026-04-10", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	expected = time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	if !fireAt.Equal(expected) {
		t.Errorf("ComputeFireAt(0, 2026-04-10) = %v, want %v", fireAt, expected)
	}
}

func TestStoreRoundtrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Initially empty
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Reminders) != 0 {
		t.Fatalf("expected 0 reminders, got %d", len(data.Reminders))
	}

	// Add a reminder
	r := Reminder{
		ID:        GenerateID(),
		Type:      ReminderTypeCard,
		BoardSlug: "test-board",
		CardID:    "card123",
		Mode:      ModeRelative,
		RelativeOffset: "-1d",
		CreatedAt: time.Now(),
		FireAt:    time.Now().Add(24 * time.Hour),
	}
	if err := store.AddReminder(r); err != nil {
		t.Fatal(err)
	}

	// Reload and verify
	data, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Reminders) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(data.Reminders))
	}
	if data.Reminders[0].CardID != "card123" {
		t.Errorf("expected card123, got %s", data.Reminders[0].CardID)
	}

	// File should exist
	if _, err := os.Stat(filepath.Join(dir, "settings.reminder.json")); err != nil {
		t.Fatalf("settings.reminder.json not created: %v", err)
	}

	// Remove
	if err := store.RemoveReminder(r.ID); err != nil {
		t.Fatal(err)
	}
	data, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Reminders) != 0 {
		t.Fatalf("expected 0 reminders after remove, got %d", len(data.Reminders))
	}
}

func TestStoreSnooze(t *testing.T) {
	store := NewStore(t.TempDir())
	r := Reminder{
		ID:     GenerateID(),
		Type:   ReminderTypeCard,
		CardID: "card1",
		Mode:   ModeRelative,
		FireAt: time.Now().Add(-time.Hour),
		Fired:  true,
	}
	_ = store.AddReminder(r)

	if err := store.SnoozeReminder(r.ID, 15*time.Minute); err != nil {
		t.Fatal(err)
	}

	data, _ := store.Load()
	if data.Reminders[0].Fired {
		t.Error("expected fired=false after snooze")
	}
	if data.Reminders[0].SnoozedUntil == nil {
		t.Error("expected snoozed_until to be set")
	}
}

func TestStoreRemoveByCardID(t *testing.T) {
	store := NewStore(t.TempDir())
	_ = store.AddReminder(Reminder{ID: "r1", Type: ReminderTypeCard, BoardSlug: "b1", CardID: "c1", FireAt: time.Now()})
	_ = store.AddReminder(Reminder{ID: "r2", Type: ReminderTypeCard, BoardSlug: "b1", CardID: "c2", FireAt: time.Now()})

	if err := store.RemoveByCardID("b1", "c1"); err != nil {
		t.Fatal(err)
	}

	data, _ := store.Load()
	if len(data.Reminders) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(data.Reminders))
	}
	if data.Reminders[0].CardID != "c2" {
		t.Errorf("wrong card remaining: %s", data.Reminders[0].CardID)
	}
}

func TestEnsureCardIDs(t *testing.T) {
	board := &models.Board{
		Columns: []models.Column{
			{
				Name: "Todo",
				Cards: []models.Card{
					{Title: "Card 1"},
					{Title: "Card 2", Metadata: map[string]string{"id": "existing"}},
				},
			},
		},
	}

	changed := EnsureCardIDs(board)
	if !changed {
		t.Error("expected changed=true")
	}

	id1 := board.Columns[0].Cards[0].Metadata["id"]
	id2 := board.Columns[0].Cards[1].Metadata["id"]
	if id1 == "" {
		t.Error("card 1 should have an ID")
	}
	if id2 != "existing" {
		t.Errorf("card 2 ID should be preserved, got %s", id2)
	}

	// Second call should not change
	changed = EnsureCardIDs(board)
	if changed {
		t.Error("expected changed=false on second call")
	}
}

func TestNextRecurrence(t *testing.T) {
	ref := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC) // Monday

	// Daily at 09:00
	next, err := NextRecurrence(&Recurrence{Frequency: "daily", Time: "09:00"}, ref, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("daily: got %v, want %v", next, expected)
	}

	// Weekly on wednesday at 09:00
	next, err = NextRecurrence(&Recurrence{Frequency: "weekly", Day: "wednesday", Time: "09:00"}, ref, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	expected = time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("weekly wed: got %v, want %v", next, expected)
	}

	// Monthly on 15th at 10:00
	next, err = NextRecurrence(&Recurrence{Frequency: "monthly", DayOfMonth: 15, Time: "10:00"}, ref, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	expected = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("monthly 15th: got %v, want %v", next, expected)
	}
}
