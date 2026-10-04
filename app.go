package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"musicplayer/internal/music"
	"musicplayer/internal/queue"
)

// AppConfig 保存播放器需要记住的用户状态
type AppConfig struct {
	LastFolder string  `json:"lastFolder"`
	LastSong   string  `json:"lastSong"`
	PlayMode   string  `json:"playMode"` // sequence / repeat-one / shuffle
	Volume     float64 `json:"volume"`   // 0 到 1
}

// App 是绑定给前端调用的对象，前端通过 wailsjs/go/main/App.js 调用这里的公开方法
type App struct {
	ctx     context.Context
	library []music.Song // 最近一次扫描到的歌曲库
	queue   *queue.Queue // 播放队列
}

func NewApp() *App {
	return &App{queue: queue.New()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

// ---- 配置 ----

// LoadConfig 读取上次保存的播放器状态
func (a *App) LoadConfig() (AppConfig, error) {
	// 先填默认值，旧版本的配置文件里没有这些字段时就用默认值
	config := AppConfig{PlayMode: string(queue.Sequence), Volume: 1}
	path, err := configFilePath()
	if err != nil {
		return config, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}

// SaveConfig 保存播放器状态，供下次启动时恢复
func (a *App) SaveConfig(config AppConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func configFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "SwiftPaw", "config.json"), nil
}
