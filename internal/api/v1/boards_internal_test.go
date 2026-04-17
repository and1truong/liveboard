package v1

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/and1truong/liveboard/pkg/models"
)

// noFlushWriter is an http.ResponseWriter that deliberately does not implement
// http.Flusher, so getEvents returns 500 immediately instead of blocking on SSE.
type noFlushWriter struct {
	h    http.Header
	code int
	buf  bytes.Buffer
}

func (w *noFlushWriter) Header() http.Header         { return w.h }
func (w *noFlushWriter) Write(b []byte) (int, error) { return w.buf.Write(b) }
func (w *noFlushWriter) WriteHeader(code int)        { w.code = code }

func TestGetEvents_noFlusher(t *testing.T) {
	// Call the handler directly (bypassing chi's writer wrapper which adds Flush).
	deps := Deps{} // SSE nil is fine; noFlusher triggers early return
	w := &noFlushWriter{h: make(http.Header)}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	deps.getEvents(w, r)
	if w.code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.code)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		t    time.Time
		want string
	}{
		{"zero", time.Time{}, ""},
		{"just_now", now.Add(-10 * time.Second), "just now"},
		{"one_min", now.Add(-90 * time.Second), "1m ago"},
		{"two_min", now.Add(-150 * time.Second), "2m ago"},
		{"one_hour", now.Add(-90 * time.Minute), "1h ago"},
		{"two_hours", now.Add(-150 * time.Minute), "2h ago"},
		{"one_day", now.Add(-36 * time.Hour), "1d ago"},
		{"two_days", now.Add(-60 * time.Hour), "2d ago"},
		{"old_date", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), "Jan 1, 2020"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := relativeTime(tc.t)
			if got != tc.want {
				t.Errorf("relativeTime(%v) = %q, want %q", tc.t, got, tc.want)
			}
		})
	}
}

func TestRelativeTime_pluralMinutes(t *testing.T) {
	now := time.Now()
	for _, m := range []int{3, 5, 10, 30, 59} {
		t.Run(fmt.Sprintf("%dm", m), func(t *testing.T) {
			got := relativeTime(now.Add(-time.Duration(m)*time.Minute - 30*time.Second))
			want := fmt.Sprintf("%dm ago", m)
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestRelativeTime_pluralHours(t *testing.T) {
	now := time.Now()
	for _, h := range []int{2, 5, 23} {
		t.Run(fmt.Sprintf("%dh", h), func(t *testing.T) {
			got := relativeTime(now.Add(-time.Duration(h)*time.Hour - 30*time.Minute))
			want := fmt.Sprintf("%dh ago", h)
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestRelativeTime_pluralDays(t *testing.T) {
	now := time.Now()
	for _, d := range []int{3, 7, 14, 29} {
		t.Run(fmt.Sprintf("%dd", d), func(t *testing.T) {
			got := relativeTime(now.Add(-time.Duration(d)*24*time.Hour - time.Hour))
			want := fmt.Sprintf("%dd ago", d)
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestBoardFileSlug_noFilePath(t *testing.T) {
	b := &models.Board{Name: "My Board"}
	got := boardFileSlug("/tmp/ws", b)
	if got != "My Board" {
		t.Errorf("want %q, got %q", "My Board", got)
	}
}

func TestBoardFileSlug_nestedRelativePath(t *testing.T) {
	b := &models.Board{Name: "ideas", FilePath: "/tmp/ws/Work/ideas.md"}
	got := boardFileSlug("/tmp/ws", b)
	if got != "Work/ideas" {
		t.Errorf("want %q, got %q", "Work/ideas", got)
	}
}

func TestBoardFileSlug_rootFilePath(t *testing.T) {
	b := &models.Board{Name: "ideas", FilePath: "/tmp/ws/ideas.md"}
	got := boardFileSlug("/tmp/ws", b)
	if got != "ideas" {
		t.Errorf("want %q, got %q", "ideas", got)
	}
}

func TestRelativeTime_oldDateFormat(t *testing.T) {
	ts := time.Date(2019, 6, 15, 0, 0, 0, 0, time.UTC)
	got := relativeTime(ts)
	if !strings.Contains(got, "2019") {
		t.Errorf("expected year in output, got %q", got)
	}
}
