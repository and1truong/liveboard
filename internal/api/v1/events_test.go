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

func TestSSEReceivesBoardUpdated(t *testing.T) {
	deps := newTestDepsWithSSE(t)
	r := chi.NewRouter()
	r.Mount("/api/v1", v1.Router(deps))

	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/events?board=demo", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if resp == nil || err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("want Content-Type=text/event-stream, got %q", ct)
	}

	// Trigger an update in a goroutine so the test handler is already subscribed.
	go func() {
		time.Sleep(50 * time.Millisecond)
		deps.SSE.Publish("demo")
	}()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: board.updated") {
			return // success
		}
	}
	t.Fatal("did not receive board.updated event")
}
