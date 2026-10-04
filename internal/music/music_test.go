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
	for _, p := range []string{filepath.Join(dir, "晴天.mp3"), filepath.Join(sub, "稻香.flac"), filepath.Join(dir, "封面.jpg")} {
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
	if s, ok := byTitle["稻香"]; !ok || s.Format != "FLAC" || s.Artist != "未知歌手" {
		t.Errorf("子文件夹里的 flac 没读对：%+v", s)
	}
	if s, ok := byTitle["晴天"]; !ok || s.Format != "MP3" {
		t.Errorf("mp3 没读对：%+v", s)
	}
}
