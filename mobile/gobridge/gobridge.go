// Package gobridge exposes the LiveBoard server to iOS via gomobile bind.
// Build with: gomobile bind -target=ios -o Gobridge.xcframework ./mobile/gobridge
package gobridge

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/and1truong/liveboard/internal/api"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

var (
	mu      sync.Mutex
	srv     *api.Server
	baseURL string
)

// Start launches the LiveBoard HTTP server on a random port for the given
// workspace directory. It returns the base URL (e.g. "http://127.0.0.1:54321").
// Call Stop before starting a new server.
func Start(workspaceDir, version string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	if srv != nil {
		return baseURL, nil // already running
	}

	ws := workspace.Open(workspaceDir)
	eng := board.New()

	srv = api.NewServer(ws, eng, false, false, true, version, "", "")

	addr, err := srv.ListenAndServe("127.0.0.1:0")
	if err != nil {
		srv = nil
		return "", fmt.Errorf("listen: %w", err)
	}
	baseURL = fmt.Sprintf("http://%s", addr.String())
	log.Printf("LiveBoard server listening on %s (workspace: %s)", baseURL, workspaceDir)
	return baseURL, nil
}

// Stop gracefully shuts down the running server.
func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	srv = nil
	baseURL = ""
}

// URL returns the base URL of the running server, or empty string if not running.
func URL() string {
	mu.Lock()
	defer mu.Unlock()
	return baseURL
}
