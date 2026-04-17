# Reminders — Specification

## Status

**Removed from runtime.** This spec captures the behavior of the HTMX-era
reminder subsystem (scheduler, store, HTTP endpoints, SSE fire-events, desktop
notifications) as of 2026-04-17, so the feature can be re-implemented on the
shell/renderer stack. The source files (`internal/reminder/`,
`internal/web/reminder_handler.go`, `internal/templates/reminders.html`,
`/events/global` broker, `startReminderScheduler` in `internal/api/server.go`)
have been deleted; this document is the sole reference.

## Purpose

Reminders let a user schedule a one-off or recurring notification tied either
to a specific card (via card ID) or to an entire board. The scheduler polls
persisted reminders once a minute; matured reminders emit an SSE event the
browser turns into an in-app toast, and on desktop they additionally raise a
native OS notification.

## Domain model

### Reminder types

| Type    | Value    | Meaning                                                  |
|---------|----------|----------------------------------------------------------|
| Card    | `card`   | Bound to a card ID on a specific board                   |
| Board   | `board`  | Bound to a whole board; payload includes BoardStats      |

### Reminder modes

| Mode       | Value       | Fire time derivation                                                      |
|------------|-------------|---------------------------------------------------------------------------|
| Relative   | `relative`  | Card due date + offset (`-1d`, `-2h`, `30m`, `1w`) in workspace timezone, resolved at 09:00 |
| Absolute   | `absolute`  | User-picked date+time, parsed as `2006-01-02T15:04` in workspace timezone |
| Recurring  | `recurring` | Frequency (`daily`/`weekly`/`monthly`) + time-of-day `HH:MM`, optional weekday, optional day-of-month |

### Offset grammar

Relative offsets are a signed integer followed by a single-character unit:
`m` (minutes), `h` (hours), `d` (days), `w` (weeks). Leading `-` is stripped
and always applied as subtraction from the due date. Empty string or `"0"`
returns the due-date at 09:00 unchanged. Example: `-1d` = 24h before due.

### Recurrence

```go
type Recurrence struct {
    Frequency  string // "daily" | "weekly" | "monthly"
    Day        string // weekday name, weekly only
    DayOfMonth int    // 1–31, monthly only (clamped into month length)
    Time       string // "HH:MM"
}
```

- **daily**: next `HH:MM` after now
- **weekly**: next occurrence of `Day` at `HH:MM`
- **monthly**: next `DayOfMonth` at `HH:MM`; if the target month is shorter,
  the day is re-clamped (e.g. Feb 31 → Feb 28/29)

### Reminder record

```go
type Reminder struct {
    ID             string        // short random ID
    Type           ReminderType  // "card" | "board"
    BoardSlug      string
    CardID         string        // required for type=card
    Mode           ReminderMode
    RelativeOffset string        // mode=relative
    AbsoluteTime   *time.Time    // mode=absolute (raw user input)
    Recurrence     *Recurrence   // mode=recurring
    CreatedAt      time.Time
    FireAt         time.Time     // computed, canonical trigger time
    LastFired      *time.Time
    Fired          bool          // true after emission, cleared on recurring
    Acknowledged   bool
    SnoozedUntil   *time.Time
}
```

### History

```go
type HistoryEntry struct {
    ID, BoardSlug, CardID, CardTitle, Message string
    FiredAt        time.Time
    AcknowledgedAt *time.Time
}
```

### Store file

- Path: `<workspace>/settings.reminder.json`
- Layout: `{ enabled, timezone, history_mode, reminders: [], history: [] }`
- Locking: single `sync.RWMutex` on the `Store`; `Mutate(fn)` is the only safe
  read-modify-write. Concurrent mutations serialize.
- Absent file ⇒ empty defaults (`timezone: "Local"`, `history_mode: "auto_purge_30d"`).

### Settings flags (on `AppSettings` in `settings.json`)

- `reminder_enabled: bool` — master gate; when false the scheduler is not started
- `reminder_timezone: string` — IANA TZ name (e.g. `America/Los_Angeles`) or
  empty/`"Local"` for `time.Local`. Used by `ComputeFireAt` and `NextRecurrence`.

### History mode

| Value              | Behavior                                           |
|--------------------|----------------------------------------------------|
| `keep_all`         | History entries persist forever                    |
| `purge_on_ack`     | Entries are not added on acknowledgement           |
| `auto_purge_30d`   | Scheduler prunes entries older than 30 days on tick |

## Scheduler

- Constructor: `NewScheduler(store, interval=1m, notifyFn, statsFn)`.
- On `Start()`: fires `tick()` immediately, then every `interval`.
- On `Stop()`: cancels the goroutine and waits for in-flight `fireReminder`
  calls to drain (`sync.WaitGroup`).

### Tick

1. `store.Mutate` under store lock:
   - Optional `purgeOldEntries(d, now)` when `HistoryMode == auto_purge_30d`.
   - Iterate reminders; for each where `shouldFire(r, now)` is true:
     - Set `Fired = true`, `LastFired = now`.
     - If `Mode == recurring`: compute `NextRecurrence`, set `FireAt = next`,
       clear `Fired` so the record remains pending.
     - Spawn `fireReminder(r)` in a goroutine (tracked by WaitGroup).
2. `shouldFire`: `!Fired && (SnoozedUntil == nil || now >= SnoozedUntil) && now >= FireAt`.

### Fire

- For `type=board`: call `statsFn(slug)` → `BoardStats{TotalOpen, Overdue, DueThisWeek}`.
- For `type=card`: resolve `cardTitle` by scanning the board's cards (the web
  handler did this by reloading the board; a replacement should do the same or
  denormalize into the store).
- Invoke `notifyFn(r, cardTitle, *BoardStats)` — fans out:
  1. SSE global event (`reminder-fire`, see payload below)
  2. Desktop native notification (macOS only; via `reminder.SendSystemNotification`)

## SSE contract

### Channel

Legacy endpoint: `GET /events/global` (text/event-stream, heartbeat `event: connected` on open).

### Event

```
event: reminder-fire
data: {"id":"abc123","type":"card","board_slug":"roadmap","card_id":"C001","card_title":"Ship beta","message":"3 open, 1 overdue, 2 due this week"}
```

- Always includes: `id`, `type`, `board_slug`, `card_id` (may be empty for board reminders), `card_title` (may be empty).
- Includes `message` only for board reminders, formatted as
  `"<TotalOpen> open, <Overdue> overdue, <DueThisWeek> due this week"`.

## Desktop notification

- Invoked only when the process is built with `isDesktop=true`.
- Implemented in `internal/reminder/notify_darwin.go` (NSUserNotification wrapper; no-op on other platforms).
- Title: `"Reminder"` for card, `"Board Reminder"` for board.
- Body: card title (card reminders), or `"<slug>: <TotalOpen> open, <Overdue> overdue"` (board reminders).

## HTTP API (legacy, to be redesigned)

All endpoints were form-encoded HTMX POSTs under `/reminders/*`. A renderer
re-implementation SHOULD prefer JSON endpoints under `/api/v1/reminders/*`.
Shape preserved here only for behavioral reference:

| Method  | Path                              | Body / params                                        | Effect                                              |
|---------|-----------------------------------|------------------------------------------------------|-----------------------------------------------------|
| POST    | `/reminders/set`                  | `board_slug`, `card_id?`, `type`, `mode`, `offset?`, `absolute_time?`, `due_date?`, `frequency?`, `day?`, `time?` | Create or replace (card-level) a reminder. |
| POST    | `/reminders/dismiss/{id}`         | —                                                    | Acknowledge; non-recurring → move to history + delete. Recurring → clear `Fired`/`Acknowledged`. |
| POST    | `/reminders/snooze/{id}`          | `duration` (`15m` \| `1h` \| `1d` \| `tomorrow` \| Go duration) | Set `SnoozedUntil = now+d`, clear fired state.       |
| DELETE  | `/reminders/{id}`                 | —                                                    | Hard-delete.                                        |
| POST    | `/reminders/clear-fired`          | —                                                    | Move all non-recurring fired reminders to history.  |
| POST    | `/reminders/clear-history`        | —                                                    | Empty `history`.                                    |
| POST    | `/reminders/settings`             | `enabled`, `timezone`, `history_mode`                | Updates both `AppSettings.reminder_*` and store metadata. |

### Store API (Go)

```go
Add|Remove|Acknowledge|SnoozeReminder(…) error
RemoveByCardID(boardSlug, cardID) error           // card deletion hook
RecalculateRelativeReminder(boardSlug, cardID, newDueDate, tz) error // due-date change hook
ClearFired() / ClearHistory() / PurgeOldHistory(maxAge) error
FindReminderForCard(boardSlug, cardID) *Reminder
```

### Card lifecycle hooks

Board mutation handlers wired two callbacks into `ReminderStore` so reminders
tracked card state without the reminder package importing the board engine:

- **On card completion** → `store.RemoveByCardID(slug, cardID)` (no reminder fires for done work).
- **On due-date change** → `store.RecalculateRelativeReminder(slug, cardID, newDue, tz)` to rebase relative reminders.

A re-implementation must re-wire these hooks at the card mutation site (either
via `internal/api/v1` dispatch or a board-engine observer).

## Re-implementation notes

- Expose reminder CRUD as JSON under `/api/v1/reminders`.
- Subscribe the renderer to `/api/v1/events` (existing board SSE) and add a
  `reminder.fire` event to that stream — no need for a parallel `/events/global`.
- Render the reminders UI inside the renderer's existing Popover / Modal
  components. `BoardSidebar.tsx` previously linked to `/reminders` as an
  anchor; that link is gone and the entry point should become a sidebar icon
  or command-palette entry.
- Preserve the `settings.reminder.json` file format so existing workspaces
  keep their reminders. The store package (`internal/reminder/`) is small and
  self-contained; resurrecting it is preferable to re-deriving the schema.
- Desktop notification helper (`notify_darwin.go`) is worth keeping as a
  general-purpose utility even before reminders return.
