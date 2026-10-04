package main

import (
	"musicplayer/internal/lyrics"
	"musicplayer/internal/music"
)

// GetLyrics 返回一首歌的歌词。
// 本地歌曲：先找同名 .lrc 文件，再找文件里内嵌的歌词；在线歌曲：问对应的插件要。
// 找不到时返回空列表（不是错误），前端显示“暂无歌词”。
func (a *App) GetLyrics(song music.Song) (lyrics.Lyrics, error) {
	return a.core.GetLyrics(song)
}
