package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"musicplayer/internal/music"
)

func newCore(t *testing.T) (*Core, string) {
	dir := t.TempDir()
	c := New(Options{DataDir: dir, AppVersion: "9.9.9"})
	t.Cleanup(c.Close)
	return c, dir
}

func TestConfigRoundTrip(t *testing.T) {
	c, dir := newCore(t)
	config, err := c.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.PlayMode != "sequence" || config.Volume != 1 || config.Theme != "dark" {
		t.Fatalf("默认配置不对：%+v", config)
	}
	config.LastFolder = "/music"
	config.Theme = "light"
	if err := c.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatalf("配置文件应该保存在数据文件夹里：%v", err)
	}
	got, err := c.LoadConfig()
	if err != nil || got != config {
		t.Fatalf("读回来的配置不一样：%+v %v", got, err)
	}
	if c.AppVersion() != "9.9.9" {
		t.Fatalf("版本号不对：%s", c.AppVersion())
	}
}

func TestNoDataDir(t *testing.T) {
	c := New(Options{})
	defer c.Close()
	if _, err := c.Playlists(); err == nil {
		t.Fatal("没有数据文件夹时歌单应该返回错误")
	}
	if _, err := c.Plugins(); !errors.Is(err, errNoPlugins) {
		t.Fatalf("没有数据文件夹时插件应该不可用：%v", err)
	}
	if c.PluginsAvailable() {
		t.Fatal("PluginsAvailable 应该是 false")
	}
	config, err := c.LoadConfig()
	if err == nil || config.Volume != 1 {
		t.Fatalf("没有数据文件夹时应该返回默认配置和错误：%+v %v", config, err)
	}
	// 队列不依赖数据文件夹
	state := c.PlaySongs([]music.Song{{Path: "/a/一.mp3", Title: "一"}}, 0)
	if state.Current != 0 {
		t.Fatalf("队列应该还能用：%+v", state)
	}
}

func TestSetLibrary(t *testing.T) {
	c, _ := newCore(t)
	kept := c.SetLibrary([]music.Song{
		{Path: "/sdcard/Music/示例一.mp3", Title: "示例一"},
		{Path: "/sdcard/Music/铃声.amr", Title: "不支持的格式"},
		{Path: "/sdcard/Music/示例二.FLAC", Title: "示例二", Source: "local", Name: "二"},
	})
	if len(kept) != 2 {
		t.Fatalf("应该只留下支持的格式：%+v", kept)
	}
	if kept[0].Source != music.LocalSource || kept[0].Name != "示例一" {
		t.Fatalf("应该补上来源和名字：%+v", kept[0])
	}
	state := c.PlayLibrary(1)
	if len(state.Songs) != 2 || state.Current != 1 || state.Songs[1].Title != "示例二" {
		t.Fatalf("PlayLibrary 应该用设置好的歌曲库：%+v", state)
	}
}

func TestScanMusic(t *testing.T) {
	c, _ := newCore(t)
	music := t.TempDir()
	for _, name := range []string{"示例一.mp3", "说明.txt"} {
		if err := os.WriteFile(filepath.Join(music, name), []byte("not audio"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	songs, err := c.ScanMusic(music)
	if err != nil || len(songs) != 1 {
		t.Fatalf("扫描结果不对：%+v %v", songs, err)
	}
	if state := c.PlayLibrary(0); len(state.Songs) != 1 {
		t.Fatalf("扫描后歌曲库应该有一首歌：%+v", state)
	}
}

func TestPlaylistsThroughCore(t *testing.T) {
	c, _ := newCore(t)
	song := music.Song{Path: "/music/示例.mp3", Title: "示例", Source: "local"}
	liked, err := c.ToggleFavorite(song)
	if err != nil || !liked {
		t.Fatalf("收藏失败：%v", err)
	}
	keys, err := c.FavoriteKeys()
	if err != nil || len(keys) != 1 || keys[0] != song.Path {
		t.Fatalf("收藏列表不对：%v %v", keys, err)
	}
	p, err := c.CreatePlaylist("跑步")
	if err != nil {
		t.Fatal(err)
	}
	if n, err := c.AddToPlaylist(p.ID, song); err != nil || n != 1 {
		t.Fatalf("加入歌单失败：%d %v", n, err)
	}
	if err := c.RecordPlay(song); err != nil {
		t.Fatal(err)
	}
	recent, err := c.RecentPlays()
	if err != nil || len(recent) != 1 {
		t.Fatalf("最近播放不对：%v %v", recent, err)
	}
}

func TestCoverAndLyricsOfMissingFile(t *testing.T) {
	c, _ := newCore(t)
	if _, err := c.Cover("/没有这个文件.mp3"); err == nil {
		t.Fatal("文件不存在时应该返回错误")
	}
	if _, err := c.Cover("/etc/passwd"); err == nil {
		t.Fatal("不是音频文件时不应该读")
	}
	l, err := c.GetLyrics(music.Song{Path: "/没有这个文件.mp3"})
	if err != nil || len(l.Lines) != 0 {
		t.Fatalf("找不到歌词时应该返回空列表：%+v %v", l, err)
	}
}
