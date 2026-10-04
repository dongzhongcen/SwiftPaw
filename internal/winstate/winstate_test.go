package winstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	if got := Load(filepath.Join(t.TempDir(), "window.json")); got != Default {
		t.Fatalf("文件不存在时应该用默认大小，实际 %+v", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "window.json")
	if err := Save(path, Size{Width: 1280, Height: 900}); err != nil {
		t.Fatal(err)
	}
	if got := Load(path); got != (Size{Width: 1280, Height: 900}) {
		t.Fatalf("读回来的大小不对：%+v", got)
	}
}

func TestLoadBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.json")
	os.WriteFile(path, []byte("{坏掉的 json"), 0644)
	if got := Load(path); got != Default {
		t.Fatalf("文件坏了应该用默认大小，实际 %+v", got)
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		in, want Size
	}{
		{Size{1200, 800}, Size{1200, 800}},
		{Size{500, 300}, Size{MinWidth, MinHeight}}, // 太小：放大到最小尺寸
		{Size{900, 100}, Size{900, MinHeight}},
		{Size{0, 800}, Default}, // 不是正数
		{Size{-5, -5}, Default},
		{Size{50000, 800}, Default}, // 大得离谱
	}
	for _, c := range cases {
		if got := Clamp(c.in); got != c.want {
			t.Errorf("Clamp(%+v) = %+v，应该是 %+v", c.in, got, c.want)
		}
	}
}
