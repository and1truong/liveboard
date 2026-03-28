package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/and1truong/liveboard/internal/defaults"
)

var version = "dev"

//go:embed all:placeholder
var placeholder embed.FS

func main() {
	app := NewApp(version)

	width, height := 1280, 860
	if cfg := defaults.LoadDesktopConfig(); cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		width, height = cfg.WindowWidth, cfg.WindowHeight
	}

	if err := wails.Run(&options.App{
		Title:     "LiveBoard",
		Width:     width,
		Height:    height,
		MinWidth:  800,
		MinHeight: 600,
		Menu:      app.buildMenu(),
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
