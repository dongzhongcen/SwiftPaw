// Package plugin 管理音乐源插件：安装、卸载、启用/停用，以及调用插件的搜索、播放地址、歌词接口。
//
// 插件是一个 CommonJS 格式的 JS 文件，导出：
//
//	module.exports = {
//	  platform, version, author, srcUrl, supportedSearchType,
//	  search(query, page, type),        // 返回 { isEnd, data: [musicItem] }
//	  getMediaSource(musicItem, quality), // 返回 { url, headers?, userAgent? }
//	  getLyric(musicItem),               // 返回 { rawLrc }
//	}
//
// 插件文件放在 %AppData%/SwiftPaw/plugins/*.js，启用状态保存在 plugins.json。
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
