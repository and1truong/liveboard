package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

var (
	version = "dev"
	commit  = "none"
)

//go:embed all:placeholder
var placeholder embed.FS

func main() {
	app := NewApp(version)

	if err := wails.Run(&options.App{
		Title:     "LiveBoard",
		Width:     1280,
		Height:    860,
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
