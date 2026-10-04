// Package store 用 SQLite 保存歌单、收藏和最近播放。
//
// 用的是 modernc.org/sqlite，它是纯 Go 写的 SQLite，不需要 C 编译器，Windows 上直接编译就行。
// 每首歌整体存成一段 JSON（data 列），以后给歌曲加新字段时不用改表结构。
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite" // 只需要注册驱动

	"musicplayer/internal/music"
)

// FavoritesID 是内置歌单“我喜欢”的 id，不能改名也不能删除
const FavoritesID int64 = 1

// MaxNameLength 是歌单名的最大字数
const MaxNameLength = 40

var (
	ErrEmptyName     = errors.New("歌单名不能为空")
	ErrNameTooLong   = errors.New("歌单名最多 40 个字")
	ErrBuiltin       = errors.New("“我喜欢”是内置歌单，不能改名或删除")
	ErrNotFound      = errors.New("歌单不存在")
	ErrDuplicateName = errors.New("已经有同名的歌单了")
)

// Playlist 是一个歌单
type Playlist struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Count   int    `json:"count"`   // 里面有几首歌
	Builtin bool   `json:"builtin"` // 是否是内置歌单（我喜欢）
}

// Store 封装数据库连接
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS playlists (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT    NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS playlist_songs (
	playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
	song_key    TEXT    NOT NULL,
	data        TEXT    NOT NULL,
	position    INTEGER NOT NULL,
	added_at    INTEGER NOT NULL,
	PRIMARY KEY (playlist_id, song_key)
);
CREATE TABLE IF NOT EXISTS history (
	song_key   TEXT    PRIMARY KEY,
	data       TEXT    NOT NULL,
	played_at  INTEGER NOT NULL,
	play_count INTEGER NOT NULL DEFAULT 1
);
INSERT OR IGNORE INTO playlists (id, name, created_at) VALUES (1, '我喜欢', 0);
`

// Open 打开（不存在就创建）数据库文件
func Open(path string) (*Store, error) {
	// _pragma 参数让每个连接都打开外键约束，并在数据库忙时最多等 5 秒
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // 桌面应用只有一个用户，用一个连接最简单，也不会锁冲突
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}

// ---- 歌单 ----

// Playlists 返回所有歌单，“我喜欢”排在第一个
func (s *Store) Playlists() ([]Playlist, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, COUNT(ps.song_key)
		FROM playlists p LEFT JOIN playlist_songs ps ON ps.playlist_id = p.id
		GROUP BY p.id ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lists := []Playlist{}
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.Count); err != nil {
			return nil, err
		}
		p.Builtin = p.ID == FavoritesID
		lists = append(lists, p)
	}
	return lists, rows.Err()
}

// CreatePlaylist 新建歌单
func (s *Store) CreatePlaylist(name string) (Playlist, error) {
	name, err := s.checkName(name, 0)
	if err != nil {
		return Playlist{}, err
	}
	res, err := s.db.Exec(`INSERT INTO playlists (name, created_at) VALUES (?, ?)`, name, time.Now().Unix())
	if err != nil {
		return Playlist{}, err
	}
	id, err := res.LastInsertId()
	return Playlist{ID: id, Name: name}, err
}

// RenamePlaylist 给歌单改名
func (s *Store) RenamePlaylist(id int64, name string) error {
	if id == FavoritesID {
		return ErrBuiltin
	}
	name, err := s.checkName(name, id)
	if err != nil {
		return err
	}
	return mustAffect(s.db.Exec(`UPDATE playlists SET name = ? WHERE id = ?`, name, id))
}

// DeletePlaylist 删除歌单（里面的歌会因为外键 ON DELETE CASCADE 一起删掉）
func (s *Store) DeletePlaylist(id int64) error {
	if id == FavoritesID {
		return ErrBuiltin
	}
	return mustAffect(s.db.Exec(`DELETE FROM playlists WHERE id = ?`, id))
}

// PlaylistSongs 返回歌单里的歌，按加入顺序
func (s *Store) PlaylistSongs(id int64) ([]music.Song, error) {
	return s.querySongs(`SELECT data FROM playlist_songs WHERE playlist_id = ? ORDER BY position`, id)
}

// AddToPlaylist 把歌加到歌单末尾，已经在歌单里的歌会跳过。返回实际新加了几首。
func (s *Store) AddToPlaylist(id int64, songs ...music.Song) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() // Commit 成功后再 Rollback 不会有任何效果

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM playlists WHERE id = ?`, id).Scan(&exists); err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, ErrNotFound
	}

	var next int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM playlist_songs WHERE playlist_id = ?`, id).Scan(&next); err != nil {
		return 0, err
	}

	added := 0
	now := time.Now().Unix()
	for _, song := range songs {
		data, err := json.Marshal(song)
		if err != nil {
			return 0, err
		}
		res, err := tx.Exec(`INSERT OR IGNORE INTO playlist_songs (playlist_id, song_key, data, position, added_at)
			VALUES (?, ?, ?, ?, ?)`, id, Key(song), string(data), next, now)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			added++
			next++
		}
	}
	return added, tx.Commit()
}

// RemoveFromPlaylist 从歌单里删掉一首歌
func (s *Store) RemoveFromPlaylist(id int64, key string) error {
	_, err := s.db.Exec(`DELETE FROM playlist_songs WHERE playlist_id = ? AND song_key = ?`, id, key)
	return err
}

// ---- 收藏（就是往“我喜欢”里加或删） ----

// ToggleFavorite 收藏或取消收藏，返回操作之后是否处于收藏状态
func (s *Store) ToggleFavorite(song music.Song) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM playlist_songs WHERE playlist_id = ? AND song_key = ?`, FavoritesID, Key(song))
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return false, nil // 原来收藏了，现在删掉了
	}
	_, err = s.AddToPlaylist(FavoritesID, song)
	return err == nil, err
}

// FavoriteKeys 返回所有收藏歌曲的 key，前端用它来显示红心
func (s *Store) FavoriteKeys() ([]string, error) {
	rows, err := s.db.Query(`SELECT song_key FROM playlist_songs WHERE playlist_id = ?`, FavoritesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// ---- 最近播放 ----

// historyLimit 是最近播放最多保留几首
const historyLimit = 200

// RecordPlay 记录播放了一首歌；同一首歌只保留一条，更新时间和次数
func (s *Store) RecordPlay(song music.Song) error {
	data, err := json.Marshal(song)
	if err != nil {
		return err
	}
	now := time.Now().UnixNano()
	_, err = s.db.Exec(`
		INSERT INTO history (song_key, data, played_at) VALUES (?, ?, ?)
		ON CONFLICT(song_key) DO UPDATE SET data = excluded.data, played_at = excluded.played_at, play_count = play_count + 1`,
		Key(song), string(data), now)
	if err != nil {
		return err
	}
	// 超过上限的旧记录删掉
	_, err = s.db.Exec(`DELETE FROM history WHERE song_key NOT IN (
		SELECT song_key FROM history ORDER BY played_at DESC LIMIT ?)`, historyLimit)
	return err
}

// RecentPlays 返回最近播放的歌，最新的在前
func (s *Store) RecentPlays(limit int) ([]music.Song, error) {
	if limit <= 0 || limit > historyLimit {
		limit = historyLimit
	}
	return s.querySongs(`SELECT data FROM history ORDER BY played_at DESC LIMIT ?`, limit)
}

// ClearHistory 清空最近播放
func (s *Store) ClearHistory() error {
	_, err := s.db.Exec(`DELETE FROM history`)
	return err
}

// ---- 工具函数 ----

// Key 是一首歌在数据库里的唯一标识。本地歌曲就是文件路径。
func Key(song music.Song) string {
	return song.Path
}

func (s *Store) querySongs(query string, args ...any) ([]music.Song, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	songs := []music.Song{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var song music.Song
		if err := json.Unmarshal([]byte(data), &song); err != nil {
			continue // 一条坏数据不影响其他歌
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

// checkName 检查歌单名：去掉首尾空格、不能为空、不能太长、不能和别的歌单重名（exceptID 是正在改名的歌单自己）
func (s *Store) checkName(name string, exceptID int64) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return "", ErrNameTooLong
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM playlists WHERE name = ? AND id != ?`, name, exceptID).Scan(&n); err != nil {
		return "", err
	}
	if n > 0 {
		return "", ErrDuplicateName
	}
	return name, nil
}

func mustAffect(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
