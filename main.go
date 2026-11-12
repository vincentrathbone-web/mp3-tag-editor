package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "MP3 Tag Editor",
		Width:     1320,
		Height:    840,
		MinWidth:  900,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 16, B: 13, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:           windows.RGB(26, 23, 19),
				DarkModeTitleBarInactive:   windows.RGB(18, 16, 13),
				DarkModeTitleText:          windows.RGB(239, 232, 221),
				DarkModeTitleTextInactive:  windows.RGB(160, 150, 138),
				DarkModeBorder:             windows.RGB(53, 47, 39),
				DarkModeBorderInactive:     windows.RGB(38, 34, 28),
				LightModeTitleBar:          windows.RGB(26, 23, 19),
				LightModeTitleBarInactive:  windows.RGB(18, 16, 13),
				LightModeTitleText:         windows.RGB(239, 232, 221),
				LightModeTitleTextInactive: windows.RGB(160, 150, 138),
				LightModeBorder:            windows.RGB(53, 47, 39),
				LightModeBorderInactive:    windows.RGB(38, 34, 28),
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
