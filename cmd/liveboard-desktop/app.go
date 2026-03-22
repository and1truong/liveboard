package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
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

	// Record initial workspace in recent list
	cfg := defaults.LoadDesktopConfig()
	cfg.AddRecent(dir)
	cfg.Save()

	a.startServer(dir)
}

func (a *App) startServer(dir string) {
	ws := workspace.Open(dir)
	eng := board.New()

	a.srv = api.NewServer(ws, eng, false, a.version)

	addr, err := a.srv.ListenAndServe("127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
	a.url = fmt.Sprintf("http://%s", addr.String())
	log.Printf("LiveBoard server listening on %s (workspace: %s)", a.url, dir)
}

func (a *App) switchWorkspace(dir string) {
	// Shutdown current server
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	// Start new server with the selected workspace
	a.startServer(dir)

	// Persist to config
	cfg := defaults.LoadDesktopConfig()
	cfg.AddRecent(dir)
	cfg.Save()

	// Navigate webview to new server
	runtime.WindowExecJS(a.ctx, fmt.Sprintf(`window.location.href = "%s"`, a.url))

	// Rebuild menu to update recent workspaces list
	appMenu := a.buildMenu()
	runtime.MenuSetApplicationMenu(a.ctx, appMenu)
	runtime.MenuUpdateApplicationMenu(a.ctx)
}

func (a *App) openWorkspaceDialog(_ *menu.CallbackData) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Workspace",
	})
	if err != nil || dir == "" {
		return
	}
	a.switchWorkspace(dir)
}

func (a *App) buildMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// App menu (macOS standard)
	appSubmenu := appMenu.AddSubmenu("LiveBoard")
	appSubmenu.AddText("Settings...", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		runtime.WindowExecJS(a.ctx, `window.location.href = "/settings"`)
	})
	appSubmenu.AddSeparator()
	appSubmenu.Append(menu.AppMenu())

	// File menu
	fileSubmenu := appMenu.AddSubmenu("File")
	fileSubmenu.AddText("Open Workspace...", keys.CmdOrCtrl("o"), a.openWorkspaceDialog)

	// Recent Workspaces submenu
	cfg := defaults.LoadDesktopConfig()
	if len(cfg.RecentWorkspaces) > 0 {
		recentSubmenu := fileSubmenu.AddSubmenu("Recent Workspaces")
		for _, dir := range cfg.RecentWorkspaces {
			d := dir // capture for closure
			label := filepath.Base(d)
			recentSubmenu.AddText(label, nil, func(_ *menu.CallbackData) {
				a.switchWorkspace(d)
			})
		}
	}

	fileSubmenu.AddSeparator()
	fileSubmenu.AddText("Close Window", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
		runtime.Quit(a.ctx)
	})

	// Edit menu (enables Cmd+C, Cmd+V, etc in webview)
	appMenu.Append(menu.EditMenu())

	// Window menu (Minimize, Zoom, Full Screen)
	appMenu.Append(menu.WindowMenu())

	return appMenu
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
