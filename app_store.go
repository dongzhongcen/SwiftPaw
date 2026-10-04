package main

import (
	"musicplayer/internal/music"
	"musicplayer/internal/store"
)

// ---- 歌单、收藏、最近播放 ----

// Playlists 返回所有歌单，第一个是“我喜欢”
func (a *App) Playlists() ([]store.Playlist, error) {
	return a.core.Playlists()
}

// CreatePlaylist 新建歌单
func (a *App) CreatePlaylist(name string) (store.Playlist, error) {
	return a.core.CreatePlaylist(name)
}

// RenamePlaylist 歌单改名
func (a *App) RenamePlaylist(id int64, name string) error {
	return a.core.RenamePlaylist(id, name)
}

// DeletePlaylist 删除歌单
func (a *App) DeletePlaylist(id int64) error {
	return a.core.DeletePlaylist(id)
}

// PlaylistSongs 返回歌单里的歌
func (a *App) PlaylistSongs(id int64) ([]music.Song, error) {
	return a.core.PlaylistSongs(id)
}

// AddToPlaylist 把一首歌加到歌单，返回实际新加了几首（已经在里面就是 0）
func (a *App) AddToPlaylist(id int64, song music.Song) (int, error) {
	return a.core.AddToPlaylist(id, song)
}

// RemoveFromPlaylist 从歌单里删掉一首歌
func (a *App) RemoveFromPlaylist(id int64, key string) error {
	return a.core.RemoveFromPlaylist(id, key)
}

// ToggleFavorite 收藏或取消收藏，返回之后是否是收藏状态
func (a *App) ToggleFavorite(song music.Song) (bool, error) {
	return a.core.ToggleFavorite(song)
}

// FavoriteKeys 返回所有收藏歌曲的 key（本地歌曲就是路径）
func (a *App) FavoriteKeys() ([]string, error) {
	return a.core.FavoriteKeys()
}

// RecordPlay 记录一次播放
func (a *App) RecordPlay(song music.Song) error {
	return a.core.RecordPlay(song)
}

// RecentPlays 返回最近播放的歌，最新的在前
func (a *App) RecentPlays() ([]music.Song, error) {
	return a.core.RecentPlays()
}

// ClearHistory 清空最近播放
func (a *App) ClearHistory() error {
	return a.core.ClearHistory()
}
