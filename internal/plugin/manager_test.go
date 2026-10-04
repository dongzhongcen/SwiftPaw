package plugin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"musicplayer/internal/music"
)

// fakePlugin 是测试用插件：搜索结果来自 api 服务器，播放地址带请求头，只有 high 音质
const fakePlugin = `
const axios = require("axios");
module.exports = {
	platform: "测试源",
	version: "1.2.0",
	author: "tester",
	srcUrl: "https://example.com/fake.js",
	supportedSearchType: ["music"],
	async search(query, page, type) {
		const res = await axios.get(API + "/search", { params: { q: query, page } });
		return { isEnd: page >= 2, data: res.data.items };
	},
	async getMediaSource(item, quality) {
		if (quality !== "high") return null;
		return { url: "https://cdn.example.com/" + item.id + ".mp3?secret=" + item.secret,
			headers: { Referer: "https://example.com/" }, userAgent: "FakeUA" };
	},
	async getLyric(item) {
		return { rawLrc: "[00:01.00]" + item.title };
	},
};`

func apiServer(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[
				{"id": 1234567890123, "title":"` + r.URL.Query().Get("q") + `", "artist":"歌手", "album":"专辑",
				 "artwork":"https://img.example.com/1.jpg", "duration": 200, "secret":"s1"},
				{"id":"b2", "title":"第二首", "duration":"3:05", "artwork":"javascript:alert(1)"},
				{"title":"没有 id，应该被跳过"}
			]}`))
		case "/fake.js":
			w.Write([]byte(strings.Replace(fakePlugin, "API", `"`+"http://"+r.Host+`"`, 1)))
		case "/list.json":
			w.Write([]byte(`{"plugins":[{"name":"fake","url":"http://` + r.Host + `/fake.js"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newManager(t *testing.T, dir string) *Manager {
	m := NewManager(Options{Dir: filepath.Join(dir, "plugins"), StatePath: filepath.Join(dir, "plugins.json"),
		AppVersion: "1.0.0", Timeout: 5 * time.Second})
	if err := m.LoadAll(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return m
}

func writePlugin(t *testing.T, path, source string) {
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInstallSearchPlayLyric(t *testing.T) {
	api := apiServer(t)
	dir := t.TempDir()
	m := newManager(t, dir)

	src := filepath.Join(dir, "下载的插件.js")
	writePlugin(t, src, strings.Replace(fakePlugin, "API", `"`+api.URL+`"`, 1))
	info, err := m.InstallFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "测试源" || info.Platform != "测试源" || info.Version != "1.2.0" || !info.Enabled ||
		!info.CanSearch || !info.CanPlay || !info.CanLyric {
		t.Fatalf("插件信息不对：%+v", info)
	}
	if _, err := os.Stat(filepath.Join(dir, "plugins", "测试源.js")); err != nil {
		t.Fatalf("插件文件应该按平台名保存：%v", err)
	}

	res, err := m.Search("测试源", "海边小路", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.IsEnd || len(res.Data) != 2 {
		t.Fatalf("搜索结果不对：%+v", res)
	}
	first, second := res.Data[0], res.Data[1]
	if first.ID != "1234567890123" || first.Title != "海边小路" || first.Source != "测试源" ||
		first.Key() != "测试源:1234567890123" || first.Duration != 200 || first.Artwork == "" {
		t.Fatalf("第一首转换不对：%+v", first)
	}
	if second.Duration != 185 || second.Artwork != "" || second.Artist != "未知歌手" {
		t.Fatalf("第二首转换不对：%+v", second)
	}
	if res2, _ := m.Search("测试源", "海边小路", 2, ""); !res2.IsEnd {
		t.Fatal("第 2 页应该是最后一页")
	}

	// 请求 standard 音质时插件返回 null，会自动试 high
	src2, err := m.MediaSource(first, "standard")
	if err != nil {
		t.Fatal(err)
	}
	if src2.URL != "https://cdn.example.com/1234567890123.mp3?secret=s1" ||
		src2.Headers["Referer"] != "https://example.com/" || src2.Headers["User-Agent"] != "FakeUA" {
		t.Fatalf("播放地址不对：%+v", src2)
	}

	lrc, err := m.Lyric(first)
	if err != nil || lrc != "[00:01.00]海边小路" {
		t.Fatalf("歌词不对：%q %v", lrc, err)
	}
}

func TestEnableDisableUninstall(t *testing.T) {
	dir := t.TempDir()
	m := newManager(t, dir)
	writePlugin(t, filepath.Join(dir, "plugins", "a.js"), `module.exports = { platform: "A", search() { return { isEnd: true, data: [] }; } };`)
	m.LoadAll()

	if err := m.SetEnabled("a", false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Search("A", "x", 1, ""); err == nil || !strings.Contains(err.Error(), "已停用") {
		t.Fatalf("停用后搜索应该报错：%v", err)
	}
	// 停用状态保存在 plugins.json，重新打开还在
	m2 := newManager(t, dir)
	if list := m2.List(); len(list) != 1 || list[0].Enabled {
		t.Fatalf("停用状态没有保存：%+v", list)
	}
	m2.SetEnabled("a", true)
	if _, err := m2.Search("A", "x", 1, ""); err != nil {
		t.Fatalf("启用后应该能搜索：%v", err)
	}

	if err := m2.Uninstall("a"); err != nil {
		t.Fatal(err)
	}
	if len(m2.List()) != 0 {
		t.Fatal("卸载后列表应该是空的")
	}
	if _, err := os.Stat(filepath.Join(dir, "plugins", "a.js")); !os.IsNotExist(err) {
		t.Fatal("卸载后文件应该被删掉")
	}
}

func TestBrokenAndDuplicatePlugins(t *testing.T) {
	dir := t.TempDir()
	m := newManager(t, dir)
	pdir := filepath.Join(dir, "plugins")
	writePlugin(t, filepath.Join(pdir, "1-ok.js"), `module.exports = { platform: "P" };`)
	writePlugin(t, filepath.Join(pdir, "2-dup.js"), `module.exports = { platform: "P" };`)
	writePlugin(t, filepath.Join(pdir, "3-syntax.js"), `module.exports = {`)
	writePlugin(t, filepath.Join(pdir, "4-noplatform.js"), `module.exports = { version: "1" };`)
	writePlugin(t, filepath.Join(pdir, "5-missing.js"), `const w = require("webdav"); module.exports = { platform: "W" };`)
	m.LoadAll()

	list := m.List()
	if len(list) != 5 {
		t.Fatalf("应该列出全部 5 个插件（包括加载失败的）：%+v", list)
	}
	checks := []string{"", "重复", "语法错误", "platform", `暂不支持模块 "webdav"`}
	for i, want := range checks {
		if want == "" && list[i].Error != "" || want != "" && !strings.Contains(list[i].Error, want) {
			t.Errorf("%s 的错误应该包含 %q，实际 %q", list[i].ID, want, list[i].Error)
		}
	}
	if len(list[4].Missing) != 1 || list[4].Missing[0] != "webdav" {
		t.Errorf("缺少的模块应该是 webdav：%v", list[4].Missing)
	}
	if _, err := m.Search("没有这个", "x", 1, ""); err == nil {
		t.Error("没安装的平台应该报错")
	}
}

func TestInstallURL(t *testing.T) {
	api := apiServer(t)
	dir := t.TempDir()
	m := newManager(t, dir)

	infos, err := m.InstallURL(api.URL + "/fake.js")
	if err != nil || len(infos) != 1 || infos[0].Platform != "测试源" {
		t.Fatalf("从网址安装：%+v %v", infos, err)
	}
	// 插件列表 JSON；同一个平台会覆盖旧版本，不会出现两个
	infos, err = m.InstallURL(api.URL + "/list.json")
	if err != nil || len(infos) != 1 || len(m.List()) != 1 {
		t.Fatalf("从插件列表安装：%+v %v", infos, err)
	}
	for _, bad := range []string{"file:///etc/passwd", api.URL + "/404"} {
		if _, err := m.InstallURL(bad); err == nil {
			t.Errorf("%s 应该安装失败", bad)
		}
	}
	notPlugin := filepath.Join(dir, "x.txt")
	writePlugin(t, notPlugin, "hello")
	if _, err := m.InstallFile(notPlugin); err == nil {
		t.Error("不是 .js 的文件应该被拒绝")
	}
}

func TestItemFromSong(t *testing.T) {
	song := music.Song{Source: "P", ID: "1", Title: "标题", Artist: "歌手",
		Extra: []byte(`{"id":1,"songmid":"abc","big":12345678901234567890}`)}
	raw, err := ItemFromSong(song)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{`"songmid":"abc"`, `"big":12345678901234567890`, `"platform":"P"`, `"title":"标题"`, `"id":1`} {
		if !strings.Contains(s, want) {
			t.Errorf("musicItem 里应该有 %s：%s", want, s)
		}
	}
}

func TestSafeFileName(t *testing.T) {
	cases := map[string]string{"Archive.org": "Archive-org", "../../evil": "evil", "音乐 源": "音乐-源", "": "plugin", "a/b\\c": "a-b-c"}
	for in, want := range cases {
		if got := safeFileName(in); got != want {
			t.Errorf("safeFileName(%q) = %q，期望 %q", in, got, want)
		}
	}
}
