package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/and1truong/liveboard/internal/api"
	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/defaults"
	"github.com/and1truong/liveboard/internal/workspace"
)

// App holds the desktop application state.
type App struct {
	ctx       context.Context
	srv       *api.Server
	url       string
	version   string
	navigated bool
}

// NewApp creates a new desktop app instance.
func NewApp(version string) *App {
	return &App{version: version}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	dir, _ := defaults.DesktopWorkDir()
	ws := workspace.Open(dir)
	eng := board.New()

	a.srv = api.NewServer(ws, eng, false, a.version)

	addr, err := a.srv.ListenAndServe("127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
	a.url = fmt.Sprintf("http://%s", addr.String())
	log.Printf("LiveBoard server listening on %s", a.url)
}

func (a *App) domReady(ctx context.Context) {
	if !a.navigated {
		a.navigated = true
		runtime.WindowExecJS(ctx, fmt.Sprintf(`window.location.href = "%s"`, a.url))
		return
	}
	// After navigating to the server, mark as desktop for CSS adjustments.
	runtime.WindowExecJS(ctx, `document.documentElement.classList.add("desktop-app")`)
}

func (a *App) shutdown(_ context.Context) {
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
