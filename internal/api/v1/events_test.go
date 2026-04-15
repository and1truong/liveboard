package v1_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	v1 "github.com/and1truong/liveboard/internal/api/v1"
)

// openEventStream connects to /api/v1/events (workspace-wide) and returns the
// response; caller must close Body.
func openEventStream(t *testing.T, srv *httptest.Server, ctx context.Context) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil || resp == nil {
		t.Fatalf("connect: %v", err)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("want Content-Type=text/event-stream, got %q", ct)
	}
	return resp
}

// waitForEventLine scans the SSE body for a line starting with the given prefix
// (e.g. "event: board.updated"). Returns true if found before context expiry.
func waitForEventLine(body *bufio.Scanner, prefix string) bool {
	for body.Scan() {
		if strings.HasPrefix(body.Text(), prefix) {
			return true
		}
	}
	return false
}

func TestEvents_BoardUpdated(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := openEventStream(t, srv, ctx)
	defer resp.Body.Close() //nolint:errcheck

	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				deps.SSE.Publish("demo")
			}
		}
	}()

	if !waitForEventLine(bufio.NewScanner(resp.Body), "event: board.updated") {
		t.Fatal("did not receive board.updated event")
	}
}

func TestEvents_BoardListUpdated(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := openEventStream(t, srv, ctx)
	defer resp.Body.Close() //nolint:errcheck

	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				deps.SSE.PublishBoardList()
			}
		}
	}()

	if !waitForEventLine(bufio.NewScanner(resp.Body), "event: board.list.updated") {
		t.Fatal("did not receive board.list.updated event")
	}
}

func TestEvents_NoQueryParamRequired(t *testing.T) {
	// Regression: prior version rejected /events without ?board=.
	deps := newTestDepsWithSSE(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))
	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp := openEventStream(t, srv, ctx)
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
