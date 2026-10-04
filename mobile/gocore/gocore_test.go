package gocore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestCore(t *testing.T) *Core {
	c := NewCore(t.TempDir(), "1.2.3")
	t.Cleanup(c.Close)
	return c
}

func TestCallBasic(t *testing.T) {
	c := newTestCore(t)
	out, err := c.Call("AppVersion", "")
	if err != nil || out != `"1.2.3"` {
		t.Fatalf("AppVersion = %s %v", out, err)
	}

	// 参数和返回值都是 JSON，字段名和桌面版前端用的一样
	songs := `[[{"path":"/sdcard/Music/示例一.mp3","title":"示例一"},{"path":"/sdcard/铃声.amr"}]]`
	out, err = c.Call("SetLibrary", songs)
	if err != nil {
		t.Fatal(err)
	}
	var kept []map[string]any
	if err := json.Unmarshal([]byte(out), &kept); err != nil || len(kept) != 1 || kept[0]["source"] != "local" {
		t.Fatalf("SetLibrary 返回值不对：%s %v", out, err)
	}

	out, err = c.Call("PlayLibrary", "[0]")
	if err != nil || !strings.Contains(out, `"current":0`) {
		t.Fatalf("PlayLibrary = %s %v", out, err)
	}
	// 少传参数时用零值：QueueNext() 等于 QueueNext(false)
	if _, err := c.Call("QueueNext", "[]"); err != nil {
		t.Fatal(err)
	}
	// 没有返回值的方法返回 null
	if out, err := c.Call("ClearHistory", "null"); err != nil || out != "null" {
		t.Fatalf("ClearHistory = %s %v", out, err)
	}
}

func TestCallErrors(t *testing.T) {
	c := newTestCore(t)
	cases := map[string][2]string{
		"不存在的方法":                 {"NoSuchMethod", "[]"},
		"不能调用 Close":             {"Close", "[]"},
		"不能调用 Stream":            {"Stream", "[]"},
		"不能调用 BackgroundHandler": {"BackgroundHandler", "[]"},
		"参数不是数组":                 {"PlayLibrary", `{"index":1}`},
		"参数类型不对":                 {"PlayLibrary", `["一"]`},
		"参数太多":                   {"AppVersion", `[1]`},
		"方法返回错误":                 {"RenamePlaylist", `[1, "新名字"]`}, // “我喜欢”不能改名
	}
	for name, tc := range cases {
		if _, err := c.Call(tc[0], tc[1]); err == nil {
			t.Errorf("%s：应该返回错误", name)
		}
	}
	if strings.Contains(c.Methods(), `"Close"`) || !strings.Contains(c.Methods(), `"QueueNext"`) {
		t.Errorf("方法列表不对：%s", c.Methods())
	}
}

func TestCoverMissing(t *testing.T) {
	c := newTestCore(t)
	if _, err := c.Cover("/没有这个文件.mp3"); err == nil {
		t.Fatal("文件不存在时应该返回错误")
	}
}

func TestBackground(t *testing.T) {
	c := newTestCore(t)
	// 最小的 GIF 文件头就能通过格式检查
	src := filepath.Join(t.TempDir(), "bg")
	if err := os.WriteFile(src, []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 前端通过 Call 设置背景（Android 上 Java 先把选中的图片复制到缓存文件夹，再把路径传进来）
	args, _ := json.Marshal([]string{src})
	out, err := c.Call("SetBackgroundImage", string(args))
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := json.Unmarshal([]byte(out), &name); err != nil || !strings.HasSuffix(name, ".gif") {
		t.Fatalf("SetBackgroundImage = %s %v", out, err)
	}
	bg, err := c.Background(name)
	if err != nil || bg.MIME != "image/gif" {
		t.Fatalf("Background = %+v %v", bg, err)
	}
	if _, err := os.Stat(bg.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Background("../config.json"); err == nil {
		t.Fatal("不应该能读别的文件")
	}
	if _, err := c.Call("RemoveBackgroundImage", "[]"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Background(name); err == nil {
		t.Fatal("删除后应该找不到")
	}
}
