package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"musicplayer/internal/winstate"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	size := loadWindowSize() // 上次关闭时的窗口大小

	err := wails.Run(&options.App{
		Title:     "极拍 SwiftPaw",
		Width:     size.Width,
		Height:    size.Height,
		MinWidth:  winstate.MinWidth,
		MinHeight: winstate.MinHeight,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: NewMusicHandler(app.stream),
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
