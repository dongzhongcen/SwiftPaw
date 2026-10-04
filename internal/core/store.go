package core

import (
	"musicplayer/internal/music"
	"musicplayer/internal/store"
)

// ---- 歌单、收藏、最近播放 ----

// Playlists 返回所有歌单，第一个是“我喜欢”
func (c *Core) Playlists() ([]store.Playlist, error) {
	db, err := c.db()
	if err != nil {
		return nil, err
	}
	return db.Playlists()
}

// CreatePlaylist 新建歌单
func (c *Core) CreatePlaylist(name string) (store.Playlist, error) {
	db, err := c.db()
	if err != nil {
		return store.Playlist{}, err
	}
	return db.CreatePlaylist(name)
}

// RenamePlaylist 歌单改名
func (c *Core) RenamePlaylist(id int64, name string) error {
	db, err := c.db()
	if err != nil {
		return err
	}
	return db.RenamePlaylist(id, name)
}

// DeletePlaylist 删除歌单
func (c *Core) DeletePlaylist(id int64) error {
	db, err := c.db()
	if err != nil {
		return err
	}
	return db.DeletePlaylist(id)
}

// PlaylistSongs 返回歌单里的歌
func (c *Core) PlaylistSongs(id int64) ([]music.Song, error) {
	db, err := c.db()
	if err != nil {
		return nil, err
	}
	return db.PlaylistSongs(id)
}

// AddToPlaylist 把一首歌加到歌单，返回实际新加了几首（已经在里面就是 0）
func (c *Core) AddToPlaylist(id int64, song music.Song) (int, error) {
	db, err := c.db()
	if err != nil {
		return 0, err
	}
	return db.AddToPlaylist(id, song)
}

// RemoveFromPlaylist 从歌单里删掉一首歌
func (c *Core) RemoveFromPlaylist(id int64, key string) error {
	db, err := c.db()
	if err != nil {
		return err
	}
	return db.RemoveFromPlaylist(id, key)
}

// ToggleFavorite 收藏或取消收藏，返回之后是否是收藏状态
func (c *Core) ToggleFavorite(song music.Song) (bool, error) {
	db, err := c.db()
	if err != nil {
		return false, err
	}
	return db.ToggleFavorite(song)
}

// FavoriteKeys 返回所有收藏歌曲的 key（本地歌曲就是路径）
func (c *Core) FavoriteKeys() ([]string, error) {
	db, err := c.db()
	if err != nil {
		return nil, err
	}
	return db.FavoriteKeys()
}

// RecordPlay 记录一次播放
func (c *Core) RecordPlay(song music.Song) error {
	db, err := c.db()
	if err != nil {
		return err
	}
	return db.RecordPlay(song)
}

// RecentPlays 返回最近播放的歌，最新的在前
func (c *Core) RecentPlays() ([]music.Song, error) {
	db, err := c.db()
	if err != nil {
		return nil, err
	}
	return db.RecentPlays(0)
}

// ClearHistory 清空最近播放
func (c *Core) ClearHistory() error {
	db, err := c.db()
	if err != nil {
		return err
	}
	return db.ClearHistory()
}
