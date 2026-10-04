package core

import (
	"os"
	"path/filepath"
	"testing"

	"musicplayer/internal/music"
	"musicplayer/internal/stream"
)

// demoPlugin 是测试用插件：播放地址固定，第二首歌自己带了 User-Agent
const demoPlugin = `module.exports = {
	platform: "演示源",
	version: "0.1.0",
	async search(query, page) {
		return { isEnd: true, data: [{ id: "1", title: query }, { id: "2", title: query + "（二）" }] };
	},
	async getMediaSource(item) {
		if (item.id === "2") return { url: "https://cdn.example.com/2.mp3", headers: { "user-agent": "DemoUA" } };
		return { url: "https://cdn.example.com/" + item.id + ".mp3", headers: { Referer: "https://example.com/" } };
	},
};`

func TestPluginFlowThroughCore(t *testing.T) {
	c, _ := newCore(t)
	file := filepath.Join(t.TempDir(), "demo.js")
	if err := os.WriteFile(file, []byte(demoPlugin), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := c.InstallPluginFile(file)
	if err != nil || info.Platform != "演示源" {
		t.Fatalf("安装失败：%+v %v", info, err)
	}
	result, err := c.SearchOnline("演示源", "示例", 0)
	if err != nil || len(result.Data) != 2 || result.Data[0].Title != "示例" {
		t.Fatalf("搜索结果不对：%+v %v", result, err)
	}

	song := result.Data[0]
	source, err := c.MediaSource(song)
	if err != nil {
		t.Fatal(err)
	}
	if source.URL != "https://cdn.example.com/1.mp3" || source.Headers["Referer"] != "https://example.com/" {
		t.Fatalf("播放地址不对：%+v", source)
	}
	if source.Headers["User-Agent"] != stream.DefaultUserAgent {
		t.Fatalf("插件没给 User-Agent 时应该补上默认的：%+v", source.Headers)
	}

	source, err = c.MediaSource(result.Data[1])
	if err != nil || source.Headers["user-agent"] != "DemoUA" || source.Headers["User-Agent"] != "" {
		t.Fatalf("插件给了 User-Agent（大小写不同）时不应该再补：%+v %v", source.Headers, err)
	}

	if url, err := c.ResolveSong(song); err != nil || url == "" {
		t.Fatalf("桌面版的代理地址不对：%q %v", url, err)
	}
	if _, err := c.MediaSource(music.Song{Path: "/a.mp3"}); err == nil {
		t.Fatal("本地歌曲不应该问插件要地址")
	}
}
