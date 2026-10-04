// Package plugin 管理音乐源插件：安装、卸载、启用/停用，以及调用插件的搜索、播放地址、歌词接口。
//
// 插件是一个 CommonJS 格式的 JS 文件，导出：
//
//	module.exports = {
//	  platform, version, author, srcUrl, supportedSearchType,
//	  search(query, page, type),        // 返回 { isEnd, data: [musicItem] }
//	  getMediaSource(musicItem, quality), // 返回 { url, headers?, userAgent? }
//	  getLyric(musicItem),               // 返回 { rawLrc }
//	  userVariables: [{ key, name, hint }], // 可选：需要用户填写的设置（比如 API Key）
//	}
//
// 插件文件放在 %AppData%/SwiftPaw/plugins/*.js，启用状态和用户填的设置都保存在 plugins.json。
// JS 的执行由 internal/jsrt 负责，这个包只关心“插件”这一层的逻辑。
package plugin

import "musicplayer/internal/music"

// Info 是插件的信息，插件管理页面显示用
type Info struct {
	ID                  string   `json:"id"`       // 文件名（不含 .js），卸载、启用/停用时用
	Platform            string   `json:"platform"` // 插件的平台名，在线歌曲的 source 就是它
	Version             string   `json:"version"`
	Author              string   `json:"author"`
	SrcURL              string   `json:"srcUrl"` // 插件的更新地址
	SupportedSearchType []string `json:"supportedSearchType"`
	Enabled             bool     `json:"enabled"`
	CanSearch           bool     `json:"canSearch"` // 实现了 search
	CanPlay             bool     `json:"canPlay"`   // 实现了 getMediaSource
	CanLyric            bool     `json:"canLyric"`  // 实现了 getLyric
	Error               string   `json:"error"`     // 加载失败的原因，空字符串表示正常
	Missing             []string `json:"missing"`   // 插件需要但还不支持的模块
	// UserVariables 是插件声明的“用户变量”，也就是插件设置里要用户填的项（比如 API Key）。
	// 没有声明时是空列表，界面上就不显示“设置”按钮
	UserVariables []UserVariable `json:"userVariables"`
}

// UserVariable 是插件声明的一个用户变量。插件这样写：
//
//	userVariables: [{ key: "key", name: "API Key", hint: "在某某网站申请" }]
//
// 插件运行时用 env.getUserVariables().key 读到用户填的值
type UserVariable struct {
	Key  string `json:"key"`  // 变量名，插件按这个名字读取
	Name string `json:"name"` // 显示给用户的名字，插件没写时用 key
	Hint string `json:"hint"` // 输入框里的提示文字，可以为空
}

// SearchResult 是一页搜索结果
type SearchResult struct {
	IsEnd bool         `json:"isEnd"` // 没有下一页了
	Data  []music.Song `json:"data"`
}

// MediaSource 是插件返回的播放地址
type MediaSource struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}
