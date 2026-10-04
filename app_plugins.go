package main

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"musicplayer/internal/music"
	"musicplayer/internal/plugin"
)

// 插件相关的方法：安装、卸载、启用/停用、插件设置、在线搜索、解析播放地址

// Plugins 返回所有已安装的插件（包括加载失败的，带着错误原因）
func (a *App) Plugins() ([]plugin.Info, error) {
	return a.core.Plugins()
}

// InstallPluginFromFile 弹出选择文件的窗口，安装选中的 .js 插件。
// 用户取消选择时返回空的 Info（ID 为空字符串）
func (a *App) InstallPluginFromFile() (plugin.Info, error) {
	if !a.core.PluginsAvailable() {
		return a.core.InstallPluginFile("") // 插件功能不可用：不用弹窗，直接返回原因
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "选择插件文件",
		Filters: []runtime.FileFilter{{DisplayName: "插件 (*.js)", Pattern: "*.js"}},
	})
	if err != nil || path == "" {
		return plugin.Info{}, err
	}
	return a.core.InstallPluginFile(path)
}

// InstallPluginFromURL 从网址下载并安装插件；网址也可以是一个插件列表 json
func (a *App) InstallPluginFromURL(url string) ([]plugin.Info, error) {
	return a.core.InstallPluginFromURL(url)
}

// UninstallPlugin 卸载插件（删除插件文件）
func (a *App) UninstallPlugin(id string) error {
	return a.core.UninstallPlugin(id)
}

// SetPluginEnabled 启用或停用插件，停用的插件不会出现在在线搜索里
func (a *App) SetPluginEnabled(id string, enabled bool) error {
	return a.core.SetPluginEnabled(id, enabled)
}

// PluginUserVariables 返回用户给某个插件填过的设置（变量名 -> 值），打开“设置”对话框时用
func (a *App) PluginUserVariables(id string) (map[string]string, error) {
	return a.core.PluginUserVariables(id)
}

// SetPluginUserVariables 保存插件设置，并重新加载这个插件让设置马上生效，返回最新的插件信息
func (a *App) SetPluginUserVariables(id string, values map[string]string) (plugin.Info, error) {
	return a.core.SetPluginUserVariables(id, values)
}

// ReloadPlugins 重新加载插件文件夹，返回最新的插件列表
func (a *App) ReloadPlugins() ([]plugin.Info, error) {
	return a.core.ReloadPlugins()
}

// SearchOnline 用某个插件搜索歌曲，page 从 1 开始
func (a *App) SearchOnline(platform, query string, page int) (plugin.SearchResult, error) {
	return a.core.SearchOnline(platform, query, page)
}

// ResolveSong 问插件要在线歌曲的播放地址，返回一个本地代理地址（/stream?id=...）给 <audio> 播放。
// 走代理是为了带上插件要求的请求头（比如 Referer），并且支持拖动进度条
func (a *App) ResolveSong(song music.Song) (string, error) {
	return a.core.ResolveSong(song)
}
