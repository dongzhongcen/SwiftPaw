package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"musicplayer/internal/winstate"
)

// windowStatePath 是保存窗口大小的文件：%AppData%/SwiftPaw/window.json
func windowStatePath() string {
	dir, err := dataDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "window.json")
}

// loadWindowSize 在 Wails 启动前读出上次的窗口大小
func loadWindowSize() winstate.Size {
	path := windowStatePath()
	if path == "" {
		return winstate.Default
	}
	return winstate.Load(path)
}

// beforeClose 在窗口关闭前保存窗口大小。最大化或最小化时不保存，
// 这样下次打开还是用户自己拖出来的大小。返回 false 表示不阻止关闭。
func (a *App) beforeClose(ctx context.Context) bool {
	path := windowStatePath()
	if path == "" || runtime.WindowIsMaximised(ctx) || runtime.WindowIsMinimised(ctx) || runtime.WindowIsFullscreen(ctx) {
		return false
	}
	width, height := runtime.WindowGetSize(ctx)
	if err := winstate.Save(path, winstate.Size{Width: width, Height: height}); err != nil {
		log.Printf("保存窗口大小失败：%v", err)
	}
	return false
}
