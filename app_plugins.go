package main

import (
	"errors"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"musicplayer/internal/music"
	"musicplayer/internal/plugin"
)

// 插件相关的方法：安装、卸载、启用/停用、插件设置、在线搜索、解析播放地址

// errNoPlugins 在插件管理器没能创建时返回（比如找不到配置文件夹）
var errNoPlugins = errors.New("插件功能不可用：找不到保存插件的文件夹")

// startPlugins 创建插件管理器，并在后台加载插件，不拖慢启动
func (a *App) startPlugins(dir string) {
	a.plugins = plugin.NewManager(plugin.Options{
		Dir:        filepath.Join(dir, "plugins"),
		StatePath:  filepath.Join(dir, "plugins.json"),
		AppVersion: Version,
		Logger: func(id, level, message string) {
			log.Printf("[插件 %s] %s: %s", id, level, message)
		},
	})
	// LoadAll 加载时持有管理器的锁，这时调用其它插件方法会等它加载完
	go func() {
		if err := a.plugins.LoadAll(); err != nil {
			log.Printf("加载插件失败：%v", err)
		}
	}()
}

// Plugins 返回所有已安装的插件（包括加载失败的，带着错误原因）
func (a *App) Plugins() ([]plugin.Info, error) {
	if a.plugins == nil {
		return nil, errNoPlugins
	}
	return a.plugins.List(), nil
}

// InstallPluginFromFile 弹出选择文件的窗口，安装选中的 .js 插件。
// 用户取消选择时返回空的 Info（ID 为空字符串）
func (a *App) InstallPluginFromFile() (plugin.Info, error) {
	if a.plugins == nil {
		return plugin.Info{}, errNoPlugins
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "选择插件文件",
		Filters: []runtime.FileFilter{{DisplayName: "插件 (*.js)", Pattern: "*.js"}},
	})
	if err != nil || path == "" {
		return plugin.Info{}, err
	}
	return a.plugins.InstallFile(path)
}

// InstallPluginFromURL 从网址下载并安装插件；网址也可以是一个插件列表 json
func (a *App) InstallPluginFromURL(url string) ([]plugin.Info, error) {
	if a.plugins == nil {
		return nil, errNoPlugins
	}
	return a.plugins.InstallURL(url)
}

// UninstallPlugin 卸载插件（删除插件文件）
func (a *App) UninstallPlugin(id string) error {
	if a.plugins == nil {
		return errNoPlugins
	}
	return a.plugins.Uninstall(id)
}

// SetPluginEnabled 启用或停用插件，停用的插件不会出现在在线搜索里
func (a *App) SetPluginEnabled(id string, enabled bool) error {
	if a.plugins == nil {
		return errNoPlugins
	}
	return a.plugins.SetEnabled(id, enabled)
}

// PluginUserVariables 返回用户给某个插件填过的设置（变量名 -> 值），打开“设置”对话框时用
func (a *App) PluginUserVariables(id string) (map[string]string, error) {
	if a.plugins == nil {
		return nil, errNoPlugins
	}
	return a.plugins.UserVariables(id)
}

// SetPluginUserVariables 保存插件设置，并重新加载这个插件让设置马上生效，返回最新的插件信息
func (a *App) SetPluginUserVariables(id string, values map[string]string) (plugin.Info, error) {
	if a.plugins == nil {
		return plugin.Info{}, errNoPlugins
	}
	return a.plugins.SetUserVariables(id, values)
}

// ReloadPlugins 重新加载插件文件夹，返回最新的插件列表
func (a *App) ReloadPlugins() ([]plugin.Info, error) {
	if a.plugins == nil {
		return nil, errNoPlugins
	}
	if err := a.plugins.LoadAll(); err != nil {
		return nil, err
	}
	return a.plugins.List(), nil
}

// SearchOnline 用某个插件搜索歌曲，page 从 1 开始
func (a *App) SearchOnline(platform, query string, page int) (plugin.SearchResult, error) {
	if a.plugins == nil {
		return plugin.SearchResult{}, errNoPlugins
	}
	if page < 1 {
		page = 1
	}
	return a.plugins.Search(platform, query, page, "music")
}

// ResolveSong 问插件要在线歌曲的播放地址，返回一个本地代理地址（/stream?id=...）给 <audio> 播放。
// 走代理是为了带上插件要求的请求头（比如 Referer），并且支持拖动进度条
func (a *App) ResolveSong(song music.Song) (string, error) {
	if a.plugins == nil {
		return "", errNoPlugins
	}
	if !song.IsOnline() {
		return "", errors.New("这不是在线歌曲")
	}
	source, err := a.plugins.MediaSource(song, "standard")
	if err != nil {
		return "", err
	}
	return a.stream.Register(source.URL, source.Headers)
}
