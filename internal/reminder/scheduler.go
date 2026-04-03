package reminder

import (
	"context"
	"log"
	"time"
)

// BoardStats holds summary stats for a board-level reminder notification.
type BoardStats struct {
	TotalOpen  int
	Overdue    int
	DueThisWeek int
}

// BoardStatsFunc loads summary stats for a board by slug.
type BoardStatsFunc func(slug string) BoardStats

// NotifyFunc is called when a reminder fires.
// It receives the reminder, an optional card title, and optional board stats.
type NotifyFunc func(r Reminder, cardTitle string, stats *BoardStats)

// Scheduler checks pending reminders on a tick interval and fires them.
type Scheduler struct {
	store      *Store
	interval   time.Duration
	notifyFn   NotifyFunc
	statsFn    BoardStatsFunc
	cancel     context.CancelFunc
}

// NewScheduler creates a scheduler that ticks at the given interval.
func NewScheduler(store *Store, interval time.Duration, notifyFn NotifyFunc, statsFn BoardStatsFunc) *Scheduler {
	return &Scheduler{
		store:    store,
		interval: interval,
		notifyFn: notifyFn,
		statsFn:  statsFn,
	}
}

// Start begins the scheduler loop in a goroutine. Call Stop to halt.
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.run(ctx)
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run once immediately on start
	s.tick()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	now := time.Now()

	err := s.store.Mutate(func(d *StoreData) error {
		// Auto-purge old history if configured
		if d.HistoryMode == HistoryAutoPurge {
			cutoff := now.Add(-30 * 24 * time.Hour)
			var kept []HistoryEntry
			for _, h := range d.History {
				if h.FiredAt.After(cutoff) {
					kept = append(kept, h)
				}
			}
			d.History = kept
		}

		for i := range d.Reminders {
			r := &d.Reminders[i]

			// Skip already fired (waiting for ack) or acknowledged
			if r.Fired {
				continue
			}

			// Check snooze
			if r.SnoozedUntil != nil && now.Before(*r.SnoozedUntil) {
				continue
			}

			// Check if it's time to fire
			if now.Before(r.FireAt) {
				continue
			}

			// Fire the reminder
			r.Fired = true
			fireTime := now
			r.LastFired = &fireTime

			// For recurring reminders, compute next fire time
			if r.Mode == ModeRecurring && r.Recurrence != nil {
				next, err := NextRecurrence(r.Recurrence, now, d.Timezone)
				if err != nil {
					log.Printf("reminder: failed to compute next recurrence for %s: %v", r.ID, err)
				} else {
					r.FireAt = next
					r.Fired = false // Will fire again at next time
				}
			}

			// Notify asynchronously to avoid holding the lock too long
			go s.fireReminder(*r)
		}

		return nil
	})

	if err != nil {
		log.Printf("reminder: tick error: %v", err)
	}
}

func (s *Scheduler) fireReminder(r Reminder) {
	var cardTitle string
	var stats *BoardStats

	if r.Type == ReminderTypeBoard && s.statsFn != nil {
		bs := s.statsFn(r.BoardSlug)
		stats = &bs
	}

	if s.notifyFn != nil {
		s.notifyFn(r, cardTitle, stats)
	}
}
