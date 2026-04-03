package reminder

import (
	"context"
	"log"
	"time"
)

// BoardStats holds summary stats for a board-level reminder notification.
type BoardStats struct {
	TotalOpen   int
	Overdue     int
	DueThisWeek int
}

// BoardStatsFunc loads summary stats for a board by slug.
type BoardStatsFunc func(slug string) BoardStats

// NotifyFunc is called when a reminder fires.
// It receives the reminder, an optional card title, and optional board stats.
type NotifyFunc func(r Reminder, cardTitle string, stats *BoardStats)

// Scheduler checks pending reminders on a tick interval and fires them.
type Scheduler struct {
	store    *Store
	interval time.Duration
	notifyFn NotifyFunc
	statsFn  BoardStatsFunc
	cancel   context.CancelFunc
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

// purgeOldEntries removes history entries older than 30 days.
func purgeOldEntries(d *StoreData, now time.Time) {
	if d.HistoryMode != HistoryAutoPurge {
		return
	}
	cutoff := now.Add(-30 * 24 * time.Hour)
	var kept []HistoryEntry
	for _, h := range d.History {
		if h.FiredAt.After(cutoff) {
			kept = append(kept, h)
		}
	}
	d.History = kept
}

// shouldFire checks whether a reminder is ready to fire at the given time.
func shouldFire(r *Reminder, now time.Time) bool {
	if r.Fired {
		return false
	}
	if r.SnoozedUntil != nil && now.Before(*r.SnoozedUntil) {
		return false
	}
	return !now.Before(r.FireAt)
}

func (s *Scheduler) tick() {
	now := time.Now()

	err := s.store.Mutate(func(d *StoreData) error {
		purgeOldEntries(d, now)

		for i := range d.Reminders {
			r := &d.Reminders[i]
			if !shouldFire(r, now) {
				continue
			}

			r.Fired = true
			fireTime := now
			r.LastFired = &fireTime

			if r.Mode == ModeRecurring && r.Recurrence != nil {
				next, recErr := NextRecurrence(r.Recurrence, now, d.Timezone)
				if recErr != nil {
					log.Printf("reminder: failed to compute next recurrence for %s: %v", r.ID, recErr)
				} else {
					r.FireAt = next
					r.Fired = false
				}
			}

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
