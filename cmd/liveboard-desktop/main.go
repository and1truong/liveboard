package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	version = "dev"
	commit  = "none"
)

//go:embed all:placeholder
var placeholder embed.FS

func main() {
	app := NewApp(version)

	appMenu := menu.NewMenu()

	// App menu (macOS standard)
	appSubmenu := appMenu.AddSubmenu("LiveBoard")
	appSubmenu.AddText("Settings...", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		runtime.WindowExecJS(app.ctx, `window.location.href = "/settings"`)
	})
	appSubmenu.AddSeparator()
	appSubmenu.Append(menu.AppMenu())

	// Edit menu (enables Cmd+C, Cmd+V, etc in webview)
	appMenu.Append(menu.EditMenu())

	if err := wails.Run(&options.App{
		Title:     "LiveBoard",
		Width:     1280,
		Height:    860,
		MinWidth:  800,
		MinHeight: 600,
		Menu:      appMenu,
		AssetServer: &assetserver.Options{
			Assets: placeholder,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "LiveBoard",
				Message: "Markdown-powered Kanban board\nVersion " + version,
			},
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		OnDomReady: app.domReady,
	}); err != nil {
		panic(err)
	}
}
