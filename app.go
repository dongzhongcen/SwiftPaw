package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"musicplayer/internal/music"
	"musicplayer/internal/plugin"
	"musicplayer/internal/queue"
	"musicplayer/internal/store"
	"musicplayer/internal/stream"
)

// App 是绑定给前端调用的对象，前端通过 wailsjs/go/main/App.js 调用这里的公开方法
type App struct {
	ctx     context.Context
	library []music.Song // 最近一次扫描到的歌曲库
	queue   *queue.Queue // 播放队列
	store   *store.Store // 歌单、收藏、最近播放（SQLite）
	dbErr   error        // 数据库打不开时记下原因，调用歌单相关方法时返回给前端

	plugins *plugin.Manager // 插件管理器；找不到配置文件夹时为 nil
	stream  *stream.Proxy   // 在线歌曲的播放代理，前端用 /stream?id=... 播放
}

func NewApp() *App {
	return &App{queue: queue.New(), stream: stream.New(nil)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	dir, err := dataDir()
	if err == nil {
		err = os.MkdirAll(dir, 0755)
	}
	if err == nil {
		a.startPlugins(dir)
		a.store, err = store.Open(filepath.Join(dir, "swiftpaw.db"))
	}
	a.dbErr = err
}

func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		a.store.Close()
	}
	if a.plugins != nil {
		a.plugins.Close()
	}
}

// SelectFolder 弹出选择文件夹的窗口，返回选中的路径
func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择音乐文件夹",
	})
}

// ScanMusic 扫描文件夹（包括子文件夹）里所有支持的音频文件，结果作为歌曲库
func (a *App) ScanMusic(dir string) ([]music.Song, error) {
	songs, err := music.Scan(dir)
	if err != nil {
		return nil, err
	}
	a.library = songs
	return songs, nil
}

// ---- 播放队列：每个方法都返回最新的队列快照，前端拿到后刷新界面 ----

// PlayLibrary 用整个歌曲库替换播放队列，从第 index 首开始
func (a *App) PlayLibrary(index int) queue.State {
	return a.queue.Replace(a.library, index)
}

// PlaySongs 用一组歌（比如某个歌单）替换播放队列，从第 index 首开始
func (a *App) PlaySongs(songs []music.Song, index int) queue.State {
	return a.queue.Replace(songs, index)
}

// QueueState 返回当前播放队列
func (a *App) QueueState() queue.State {
	return a.queue.State()
}

// QueuePlayAt 切到队列里的第 index 首
func (a *App) QueuePlayAt(index int) queue.State {
	return a.queue.PlayAt(index)
}

// QueueNext 下一首；auto 为 true 表示歌曲自然播完
func (a *App) QueueNext(auto bool) queue.State {
	return a.queue.Next(auto)
}

// QueuePrevious 上一首
func (a *App) QueuePrevious() queue.State {
	return a.queue.Previous()
}

// QueueSetMode 切换播放模式：sequence / repeat-one / shuffle
func (a *App) QueueSetMode(mode string) queue.State {
	return a.queue.SetMode(queue.Mode(mode))
}

// QueueAddNext 把一首歌设为下一首播放
func (a *App) QueueAddNext(song music.Song) queue.State {
	return a.queue.AddNext(song)
}

// QueueRemove 从队列里移除第 index 首
func (a *App) QueueRemove(index int) queue.State {
	return a.queue.Remove(index)
}
