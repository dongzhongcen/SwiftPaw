package music

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsAudio(t *testing.T) {
	cases := map[string]bool{
		"a.mp3": true, "B.FLAC": true, "c.m4a": true, "d.ogg": true, "e.opus": true, "f.wav": true,
		"g.txt": false, "h.jpg": false, "noext": false,
	}
	for path, want := range cases {
		if got := IsAudio(path); got != want {
			t.Errorf("IsAudio(%q) = %v, want %v", path, got, want)
		}
	}
	if ContentType("x.FLAC") != "audio/flac" || ContentType("x.txt") != "" {
		t.Error("ContentType 返回值不对")
	}
}

func TestScanFallsBackToFileName(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "子文件夹")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// 内容是假的，读不出标签，应该用文件名当歌名
	for _, p := range []string{filepath.Join(dir, "海边小路.mp3"), filepath.Join(sub, "午后小调.flac"), filepath.Join(dir, "封面.jpg")} {
		if err := os.WriteFile(p, []byte("not really audio"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	songs, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 2 {
		t.Fatalf("应该找到 2 首歌，实际 %d 首", len(songs))
	}
	byTitle := map[string]Song{}
	for _, s := range songs {
		byTitle[s.Title] = s
	}
	if s, ok := byTitle["午后小调"]; !ok || s.Format != "FLAC" || s.Artist != "未知歌手" {
		t.Errorf("子文件夹里的 flac 没读对：%+v", s)
	}
	if s, ok := byTitle["海边小路"]; !ok || s.Format != "MP3" {
		t.Errorf("mp3 没读对：%+v", s)
	}
}

func TestSongKey(t *testing.T) {
	local := Song{Path: `D:\音乐\海边小路.mp3`, Source: LocalSource}
	old := Song{Path: `D:\音乐\旧数据.mp3`} // 旧版本保存的歌没有 source 字段
	online := Song{Source: "archive", ID: "abc123", Path: ""}
	if local.IsOnline() || old.IsOnline() || !online.IsOnline() {
		t.Fatal("IsOnline 判断不对")
	}
	if local.Key() != `D:\音乐\海边小路.mp3` || old.Key() != `D:\音乐\旧数据.mp3` || online.Key() != "archive:abc123" {
		t.Fatalf("Key 不对：%q %q %q", local.Key(), old.Key(), online.Key())
	}
}
