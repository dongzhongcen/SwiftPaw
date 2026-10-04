package core

import (
	"errors"
	"net/http"

	"musicplayer/internal/music"
	"musicplayer/internal/plugin"
	"musicplayer/internal/stream"
)

// 插件相关的方法：安装、卸载、启用/停用、插件设置、在线搜索、解析播放地址

// Plugins 返回所有已安装的插件（包括加载失败的，带着错误原因）
func (c *Core) Plugins() ([]plugin.Info, error) {
	if c.plugins == nil {
		return nil, errNoPlugins
	}
	return c.plugins.List(), nil
}

// PluginsAvailable 返回插件功能能不能用（没有数据文件夹时不能用）
func (c *Core) PluginsAvailable() bool {
	return c.plugins != nil
}

// InstallPluginFile 安装一个本地的 .js 插件文件（文件会被复制到插件文件夹）。
// 选择文件的窗口由各个平台自己弹出
func (c *Core) InstallPluginFile(path string) (plugin.Info, error) {
	if c.plugins == nil {
		return plugin.Info{}, errNoPlugins
	}
	return c.plugins.InstallFile(path)
}

// InstallPluginFromURL 从网址下载并安装插件；网址也可以是一个插件列表 json
func (c *Core) InstallPluginFromURL(url string) ([]plugin.Info, error) {
	if c.plugins == nil {
		return nil, errNoPlugins
	}
	return c.plugins.InstallURL(url)
}

// UninstallPlugin 卸载插件（删除插件文件）
func (c *Core) UninstallPlugin(id string) error {
	if c.plugins == nil {
		return errNoPlugins
	}
	return c.plugins.Uninstall(id)
}

// SetPluginEnabled 启用或停用插件，停用的插件不会出现在在线搜索里
func (c *Core) SetPluginEnabled(id string, enabled bool) error {
	if c.plugins == nil {
		return errNoPlugins
	}
	return c.plugins.SetEnabled(id, enabled)
}

// PluginUserVariables 返回用户给某个插件填过的设置（变量名 -> 值），打开“设置”对话框时用
func (c *Core) PluginUserVariables(id string) (map[string]string, error) {
	if c.plugins == nil {
		return nil, errNoPlugins
	}
	return c.plugins.UserVariables(id)
}

// SetPluginUserVariables 保存插件设置，并重新加载这个插件让设置马上生效，返回最新的插件信息
func (c *Core) SetPluginUserVariables(id string, values map[string]string) (plugin.Info, error) {
	if c.plugins == nil {
		return plugin.Info{}, errNoPlugins
	}
	return c.plugins.SetUserVariables(id, values)
}

// ReloadPlugins 重新加载插件文件夹，返回最新的插件列表
func (c *Core) ReloadPlugins() ([]plugin.Info, error) {
	if c.plugins == nil {
		return nil, errNoPlugins
	}
	if err := c.plugins.LoadAll(); err != nil {
		return nil, err
	}
	return c.plugins.List(), nil
}

// SearchOnline 用某个插件搜索歌曲，page 从 1 开始
func (c *Core) SearchOnline(platform, query string, page int) (plugin.SearchResult, error) {
	if c.plugins == nil {
		return plugin.SearchResult{}, errNoPlugins
	}
	if page < 1 {
		page = 1
	}
	return c.plugins.Search(platform, query, page, "music")
}

// MediaSource 问插件要在线歌曲的真实播放地址和需要带上的请求头。
// Android 版的原生播放器直接用它播放；插件没指定 User-Agent 时补上默认的
func (c *Core) MediaSource(song music.Song) (plugin.MediaSource, error) {
	if c.plugins == nil {
		return plugin.MediaSource{}, errNoPlugins
	}
	if !song.IsOnline() {
		return plugin.MediaSource{}, errors.New("这不是在线歌曲")
	}
	source, err := c.plugins.MediaSource(song, "standard")
	if err != nil {
		return source, err
	}
	headers := map[string]string{}
	for k, v := range source.Headers {
		headers[k] = v
	}
	if !hasHeader(headers, "User-Agent") {
		headers["User-Agent"] = stream.DefaultUserAgent
	}
	source.Headers = headers
	return source, nil
}

// ResolveSong 问插件要在线歌曲的播放地址，返回一个本地代理地址（/stream?id=...）给 <audio> 播放（桌面版用）。
// 走代理是为了带上插件要求的请求头（比如 Referer），并且支持拖动进度条
func (c *Core) ResolveSong(song music.Song) (string, error) {
	if c.plugins == nil {
		return "", errNoPlugins
	}
	if !song.IsOnline() {
		return "", errors.New("这不是在线歌曲")
	}
	source, err := c.plugins.MediaSource(song, "standard")
	if err != nil {
		return "", err
	}
	return c.stream.Register(source.URL, source.Headers)
}

// hasHeader 不区分大小写地判断有没有某个请求头
func hasHeader(headers map[string]string, name string) bool {
	for k := range headers {
		if http.CanonicalHeaderKey(k) == http.CanonicalHeaderKey(name) {
			return true
		}
	}
	return false
}
