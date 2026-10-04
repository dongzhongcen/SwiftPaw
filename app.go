package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"musicplayer/internal/core"
	"musicplayer/internal/music"
	"musicplayer/internal/queue"
)

// App 是绑定给前端调用的对象，前端通过 wailsjs/go/main/App.js 调用这里的公开方法。
//
// 播放器的功能（歌曲库、队列、歌单、歌词、插件、配置）都在 internal/core 里，桌面版和 Android 版共用；
// 这里的方法大多只是转发给 core。只有和 Windows 窗口有关的事情（弹出选择文件夹/文件的窗口、
// 记住窗口大小）写在 main 包里。
type App struct {
	ctx  context.Context
	core *core.Core
}

func NewApp() *App {
	dir, err := dataDir()
	if err != nil {
		log.Printf("找不到保存数据的文件夹：%v", err)
		dir = ""
	}
	return &App{core: core.New(core.Options{DataDir: dir, AppVersion: Version})}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.core.Close()
}

// dataDir 是保存配置和数据库的文件夹，Windows 上是 %AppData%\SwiftPaw
func dataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "SwiftPaw"), nil
}

// SelectFolder 弹出选择文件夹的窗口，返回选中的路径
func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择音乐文件夹",
	})
}

// ScanMusic 扫描文件夹（包括子文件夹）里所有支持的音频文件，结果作为歌曲库
func (a *App) ScanMusic(dir string) ([]music.Song, error) {
	return a.core.ScanMusic(dir)
}

// ---- 播放队列：每个方法都返回最新的队列快照，前端拿到后刷新界面 ----

// PlayLibrary 用整个歌曲库替换播放队列，从第 index 首开始
func (a *App) PlayLibrary(index int) queue.State {
	return a.core.PlayLibrary(index)
}

// PlaySongs 用一组歌（比如某个歌单）替换播放队列，从第 index 首开始
func (a *App) PlaySongs(songs []music.Song, index int) queue.State {
	return a.core.PlaySongs(songs, index)
}

// QueueState 返回当前播放队列
func (a *App) QueueState() queue.State {
	return a.core.QueueState()
}

// QueuePlayAt 切到队列里的第 index 首
func (a *App) QueuePlayAt(index int) queue.State {
	return a.core.QueuePlayAt(index)
}

// QueueNext 下一首；auto 为 true 表示歌曲自然播完
func (a *App) QueueNext(auto bool) queue.State {
	return a.core.QueueNext(auto)
}

// QueuePrevious 上一首
func (a *App) QueuePrevious() queue.State {
	return a.core.QueuePrevious()
}

// QueueSetMode 切换播放模式：sequence / repeat-one / shuffle
func (a *App) QueueSetMode(mode string) queue.State {
	return a.core.QueueSetMode(mode)
}

// QueueAddNext 把一首歌设为下一首播放
func (a *App) QueueAddNext(song music.Song) queue.State {
	return a.core.QueueAddNext(song)
}

// QueueRemove 从队列里移除第 index 首
func (a *App) QueueRemove(index int) queue.State {
	return a.core.QueueRemove(index)
}
