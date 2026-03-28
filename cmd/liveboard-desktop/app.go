// LiveBoard desktop application using Wails.
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
	"github.com/and1truong/liveboard/internal/web"
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
	if dir == "" {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "No Workspace Found",
			Message: "Could not determine a workspace directory. Please select one.",
		})
		a.openWorkspaceDialog(nil)
		if a.srv == nil {
			runtime.Quit(a.ctx)
		}
		return
	}

	// Record initial workspace in recent list
	cfg := defaults.LoadDesktopConfig()
	cfg.AddRecent(dir)
	if err := cfg.Save(); err != nil {
		log.Printf("failed to save desktop config: %v", err)
	}

	if err := a.startServer(dir); err != nil {
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Server Error",
			Message: fmt.Sprintf("Failed to start server: %v", err),
		})
		runtime.Quit(a.ctx)
		return
	}

	// Restore last viewed board
	if s := web.LoadSettingsFromDir(dir); s.LastBoard != "" {
		a.url = a.url + "/board/" + s.LastBoard
	}
}

func (a *App) startServer(dir string) error {
	ws := workspace.Open(dir)
	eng := board.New()

	a.srv = api.NewServer(ws, eng, false, false, a.version)

	addr, err := a.srv.ListenAndServe("127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on 127.0.0.1:0: %w", err)
	}
	a.url = fmt.Sprintf("http://%s", addr.String())
	log.Printf("LiveBoard server listening on %s (workspace: %s)", a.url, dir)
	return nil
}

func (a *App) switchWorkspace(dir string) {
	// Run shutdown + restart off the main thread to avoid blocking the UI.
	go func() {
		// Shutdown current server
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.srv.Shutdown(shutCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}

		// Start new server with the selected workspace
		if err := a.startServer(dir); err != nil {
			_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Server Error",
				Message: fmt.Sprintf("Failed to switch workspace: %v", err),
			})
			return
		}

		// Persist to config
		cfg := defaults.LoadDesktopConfig()
		cfg.AddRecent(dir)
		if err := cfg.Save(); err != nil {
			log.Printf("failed to save desktop config: %v", err)
		}

		// Navigate webview to new server, restoring last board if available
		target := a.url
		if s := web.LoadSettingsFromDir(dir); s.LastBoard != "" {
			target = a.url + "/board/" + s.LastBoard
		}
		runtime.WindowExecJS(a.ctx, fmt.Sprintf(`window.location.href = "%s"`, target))

		// Rebuild menu to update recent workspaces list
		appMenu := a.buildMenu()
		runtime.MenuSetApplicationMenu(a.ctx, appMenu)
		runtime.MenuUpdateApplicationMenu(a.ctx)
	}()
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

	// Recent Workspaces submenu (clean stale entries first)
	cfg := defaults.LoadDesktopConfig()
	cfg.CleanStale()
	if err := cfg.Save(); err != nil {
		log.Printf("failed to save desktop config: %v", err)
	}
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
	// After navigating to the server, mark as desktop for CSS adjustments
	// and inject window drag handler (Wails runtime JS isn't present on external URLs).
	runtime.WindowExecJS(ctx, `
		document.documentElement.classList.add("desktop-app");
		(function() {
			var prop = "--wails-draggable";
			var val = "drag";
			var invoke = window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.external
				? function(m) { window.webkit.messageHandlers.external.postMessage(m); }
				: (window.WailsInvoke || function(){});
			var shouldDrag = false;
			window.addEventListener("mousedown", function(e) {
				var v = window.getComputedStyle(e.target).getPropertyValue(prop);
				if (v && v.trim() === val && e.buttons === 1 && e.detail === 1) {
					if (e.offsetX > e.target.clientWidth || e.offsetY > e.target.clientHeight) return;
					shouldDrag = true;
				} else {
					shouldDrag = false;
				}
			});
			window.addEventListener("mousemove", function(e) {
				if (shouldDrag) {
					shouldDrag = false;
					if (e.buttons > 0) invoke("drag");
				}
			});
			window.addEventListener("mouseup", function() {
				shouldDrag = false;
			});
		})();
	`)
}

func (a *App) shutdown(_ context.Context) {
	// Persist window size for next launch
	w, h := runtime.WindowGetSize(a.ctx)
	if w > 0 && h > 0 {
		cfg := defaults.LoadDesktopConfig()
		cfg.WindowWidth = w
		cfg.WindowHeight = h
		if err := cfg.Save(); err != nil {
			log.Printf("failed to save window size: %v", err)
		}
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
