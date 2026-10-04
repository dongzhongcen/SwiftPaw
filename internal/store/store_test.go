package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"musicplayer/internal/music"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func song(n int) music.Song {
	return music.Song{Path: fmt.Sprintf(`D:\音乐\歌%d.mp3`, n), Title: fmt.Sprintf("歌%d", n), Artist: "歌手"}
}

func TestFavoritesExistsByDefault(t *testing.T) {
	s := openTemp(t)
	lists, err := s.Playlists()
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || lists[0].ID != FavoritesID || lists[0].Name != "我喜欢" || !lists[0].Builtin {
		t.Fatalf("新数据库应该只有“我喜欢”，实际 %+v", lists)
	}
	if err := s.RenamePlaylist(FavoritesID, "x"); !errors.Is(err, ErrBuiltin) {
		t.Errorf("“我喜欢”不能改名，实际 err=%v", err)
	}
	if err := s.DeletePlaylist(FavoritesID); !errors.Is(err, ErrBuiltin) {
		t.Errorf("“我喜欢”不能删除，实际 err=%v", err)
	}
}

func TestPlaylistCRUD(t *testing.T) {
	s := openTemp(t)
	p, err := s.CreatePlaylist("  跑步  ")
	if err != nil || p.Name != "跑步" {
		t.Fatalf("新建歌单应该去掉首尾空格：%+v %v", p, err)
	}
	if _, err := s.CreatePlaylist("跑步"); !errors.Is(err, ErrDuplicateName) {
		t.Errorf("重名应该报错，实际 %v", err)
	}
	if _, err := s.CreatePlaylist("   "); !errors.Is(err, ErrEmptyName) {
		t.Errorf("空名字应该报错，实际 %v", err)
	}
	if _, err := s.CreatePlaylist(strings.Repeat("长", 41)); !errors.Is(err, ErrNameTooLong) {
		t.Errorf("41 个字应该报错，实际 %v", err)
	}

	if err := s.RenamePlaylist(p.ID, "跑步"); err != nil {
		t.Errorf("改成自己原来的名字不算重名，实际 %v", err)
	}
	if err := s.RenamePlaylist(p.ID, "夜跑"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenamePlaylist(999, "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("改不存在的歌单应该报 ErrNotFound，实际 %v", err)
	}

	added, err := s.AddToPlaylist(p.ID, song(1), song(2), song(1))
	if err != nil || added != 2 {
		t.Fatalf("应该加进 2 首（重复的跳过），实际 %d %v", added, err)
	}
	added, _ = s.AddToPlaylist(p.ID, song(3), song(2))
	if added != 1 {
		t.Errorf("第二次应该只新加 1 首，实际 %d", added)
	}
	songs, _ := s.PlaylistSongs(p.ID)
	if len(songs) != 3 || songs[0].Title != "歌1" || songs[2].Title != "歌3" {
		t.Fatalf("歌单内容或顺序不对：%+v", songs)
	}

	if err := s.RemoveFromPlaylist(p.ID, Key(song(2))); err != nil {
		t.Fatal(err)
	}
	lists, _ := s.Playlists()
	if len(lists) != 2 || lists[1].Name != "夜跑" || lists[1].Count != 2 {
		t.Fatalf("歌单列表不对：%+v", lists)
	}

	if _, err := s.AddToPlaylist(999, song(1)); !errors.Is(err, ErrNotFound) {
		t.Errorf("加到不存在的歌单应该报错，实际 %v", err)
	}

	if err := s.DeletePlaylist(p.ID); err != nil {
		t.Fatal(err)
	}
	var left int
	s.db.QueryRow(`SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = ?`, p.ID).Scan(&left)
	if left != 0 {
		t.Errorf("删除歌单后里面的歌也应该被删掉（外键级联），还剩 %d 首", left)
	}
}

func TestToggleFavorite(t *testing.T) {
	s := openTemp(t)
	on, err := s.ToggleFavorite(song(1))
	if err != nil || !on {
		t.Fatalf("第一次应该是收藏，实际 %v %v", on, err)
	}
	s.ToggleFavorite(song(2))
	keys, _ := s.FavoriteKeys()
	if len(keys) != 2 {
		t.Fatalf("应该有 2 首收藏，实际 %v", keys)
	}
	if on, _ = s.ToggleFavorite(song(1)); on {
		t.Error("第二次应该是取消收藏")
	}
	keys, _ = s.FavoriteKeys()
	if len(keys) != 1 || keys[0] != Key(song(2)) {
		t.Errorf("取消后应该只剩歌2，实际 %v", keys)
	}
}

func TestHistory(t *testing.T) {
	s := openTemp(t)
	s.RecordPlay(song(1))
	s.RecordPlay(song(2))
	s.RecordPlay(song(1)) // 再放一次歌1，它应该排到最前面，而且不会重复

	songs, err := s.RecentPlays(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 2 || songs[0].Title != "歌1" || songs[1].Title != "歌2" {
		t.Fatalf("最近播放顺序不对：%+v", songs)
	}
	var count int
	s.db.QueryRow(`SELECT play_count FROM history WHERE song_key = ?`, Key(song(1))).Scan(&count)
	if count != 2 {
		t.Errorf("歌1 应该记 2 次，实际 %d", count)
	}

	for i := 0; i < historyLimit+10; i++ {
		s.RecordPlay(song(100 + i))
	}
	all, _ := s.RecentPlays(0)
	if len(all) != historyLimit {
		t.Errorf("最近播放最多保留 %d 首，实际 %d", historyLimit, len(all))
	}

	if err := s.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	if songs, _ := s.RecentPlays(10); len(songs) != 0 {
		t.Errorf("清空后应该没有记录")
	}
}

func TestDataSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.CreatePlaylist("通勤")
	s.AddToPlaylist(p.ID, song(1))
	s.Close()

	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	songs, _ := s.PlaylistSongs(p.ID)
	if len(songs) != 1 || songs[0].Path != song(1).Path {
		t.Fatalf("重新打开后数据应该还在：%+v", songs)
	}
}
