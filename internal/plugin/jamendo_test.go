package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// 测试 examples/plugins/jamendo.js：把插件里的 API 地址换成本地的假服务器，
// 用假的 Client ID 和占位数据，不会真的访问 Jamendo
const (
	jamendoAPIBase  = `"https://api.jamendo.com/v3.0"`
	jamendoClientID = "test-client-id"
)

// jamendoAPI 是假的 Jamendo API，记录收到的请求参数
type jamendoAPI struct {
	*httptest.Server
	mu       sync.Mutex
	requests []url.Values
}

func (a *jamendoAPI) record(q url.Values) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests = append(a.requests, q)
}

func (a *jamendoAPI) last() url.Values {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.requests) == 0 {
		return nil
	}
	return a.requests[len(a.requests)-1]
}

func (a *jamendoAPI) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.requests)
}

// jamendoTrack 生成一条占位的 track 数据，字段名和 Jamendo API v3.0 一样
func jamendoTrack(id, host, format string, downloadAllowed bool) map[string]any {
	download := ""
	if downloadAllowed {
		download = "https://" + host + "/download/" + id + "/flac/"
	}
	return map[string]any{
		"id": id, "name": "测试歌曲" + id, "duration": 180,
		"artist_id": "9" + id, "artist_name": "测试歌手", "album_id": "8" + id, "album_name": "测试专辑",
		"license_ccurl": "http://creativecommons.org/licenses/by-sa/3.0/",
		"image":         "https://" + host + "/cover/" + id + ".jpg", "album_image": "",
		"audio":                 "https://" + host + "/stream/?trackid=" + id + "&format=" + format,
		"audiodownload":         download,
		"audiodownload_allowed": downloadAllowed,
		"shareurl":              "https://" + host + "/track/" + id,
	}
}

func newJamendoAPI(t *testing.T) *jamendoAPI {
	api := &jamendoAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		api.record(q)
		w.Header().Set("Content-Type", "application/json")
		reply := func(status string, code int, message string, results []map[string]any) {
			if results == nil {
				results = []map[string]any{}
			}
			json.NewEncoder(w).Encode(map[string]any{
				"headers": map[string]any{"status": status, "code": code, "error_message": message,
					"warnings": "", "results_count": len(results)},
				"results": results,
			})
		}
		if r.URL.Path != "/tracks/" {
			http.NotFound(w, r)
			return
		}
		// 和真的 Jamendo 一样：Client ID 不对时也是 HTTP 200，错误写在 headers 里
		if q.Get("client_id") != jamendoClientID {
			reply("failed", 5, "Jamendo Api Invalid Client Id Error: Your credential is not authorized.", nil)
			return
		}
		host := "media.example.com"
		format := q.Get("audioformat")
		if format == "" {
			format = "mp31"
		}
		switch {
		case q.Has("search"):
			// 第 1 页有 20 条（还有下一页），第 2 页只有 1 条
			var results []map[string]any
			n := 20
			if q.Get("offset") != "0" {
				n = 1
			}
			for i := range n {
				results = append(results, jamendoTrack(string(rune('a'+i)), host, format, i == 0))
			}
			reply("success", 0, "", results)
		case q.Get("id") != "":
			id := q.Get("id")
			if id == "missing" || id == "flaky" && api.count()%2 == 1 {
				reply("success", 0, "", nil) // 下架的歌查不到；flaky 模拟 Jamendo 偶尔返回空结果
				return
			}
			track := jamendoTrack(id, host, format, id == "a")
			if q.Get("include") == "lyrics" && id == "a" {
				track["lyrics"] = "第一行<br />第二行\r\n第三行\r\n"
			}
			reply("success", 0, "", []map[string]any{track})
		default:
			reply("failed", 1, "missing parameters", nil)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

// installJamendo 安装指向假服务器的 jamendo.js
func installJamendo(t *testing.T, api *jamendoAPI) (*Manager, Info) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "plugins", "jamendo.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "const API_BASE = "+jamendoAPIBase+";") {
		t.Fatalf("jamendo.js 里应该有 API_BASE = %s", jamendoAPIBase)
	}
	dir := t.TempDir()
	m := newManager(t, dir)
	src := filepath.Join(dir, "jamendo.js")
	writePlugin(t, src, strings.Replace(string(source), jamendoAPIBase, `"`+api.URL+`"`, 1))
	info, err := m.InstallFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return m, info
}

func TestJamendoPluginInfoAndClientID(t *testing.T) {
	api := newJamendoAPI(t)
	m, info := installJamendo(t, api)

	if info.Error != "" || info.Platform != "Jamendo" || !info.CanSearch || !info.CanPlay || !info.CanLyric {
		t.Fatalf("插件信息不对：%+v", info)
	}
	if len(info.UserVariables) != 1 {
		t.Fatalf("应该声明一个用户变量：%+v", info.UserVariables)
	}
	v := info.UserVariables[0]
	if v.Key != "client_id" || v.Name != "Client ID" || !strings.Contains(v.Hint, "devportal.jamendo.com") {
		t.Fatalf("Client ID 用户变量不对：%+v", v)
	}

	// 没填 Client ID：不发请求，直接提示去「设置」里填写
	_, err := m.Search("Jamendo", "测试", 1, "")
	if err == nil || !strings.Contains(err.Error(), "Client ID") || !strings.Contains(err.Error(), "「设置」") {
		t.Fatalf("没填 Client ID 时应该提示去设置里填写：%v", err)
	}
	if api.count() != 0 {
		t.Fatal("没填 Client ID 时不应该发请求")
	}

	// 填错了：把 Jamendo 返回的错误换成中文提示
	if _, err := m.SetUserVariables(info.ID, map[string]string{"client_id": "wrong-id"}); err != nil {
		t.Fatal(err)
	}
	_, err = m.Search("Jamendo", "测试", 1, "")
	if err == nil || !strings.Contains(err.Error(), "Client ID") || !strings.Contains(err.Error(), "「设置」") {
		t.Fatalf("Client ID 不对时应该提示检查设置：%v", err)
	}
}

func TestJamendoPluginSearchPlayLyric(t *testing.T) {
	api := newJamendoAPI(t)
	m, info := installJamendo(t, api)
	if _, err := m.SetUserVariables(info.ID, map[string]string{"client_id": " " + jamendoClientID + " "}); err != nil {
		t.Fatal(err)
	}

	res, err := m.Search("Jamendo", "测试", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	q := api.last()
	for key, want := range map[string]string{"client_id": jamendoClientID, "format": "json", "search": "测试",
		"limit": "20", "offset": "0", "type": "single albumtrack", "audioformat": "mp32"} {
		if got := q.Get(key); got != want {
			t.Errorf("搜索参数 %s = %q，期望 %q", key, got, want)
		}
	}
	if res.IsEnd || len(res.Data) != 20 {
		t.Fatalf("第 1 页应该有 20 条而且还有下一页：isEnd=%v len=%d", res.IsEnd, len(res.Data))
	}
	first, second := res.Data[0], res.Data[1]
	if first.ID != "a" || first.Title != "测试歌曲a" || first.Artist != "测试歌手" || first.Album != "测试专辑" ||
		first.Source != "Jamendo" || first.Duration != 180 || first.Artwork != "https://media.example.com/cover/a.jpg" {
		t.Fatalf("第一首转换不对：%+v", first)
	}
	if !strings.Contains(string(first.Extra), `"license":"http://creativecommons.org/licenses/by-sa/3.0/"`) {
		t.Errorf("应该保存许可证地址：%s", first.Extra)
	}

	res2, err := m.Search("Jamendo", "测试", 2, "")
	if err != nil || !res2.IsEnd || len(res2.Data) != 1 || api.last().Get("offset") != "20" {
		t.Fatalf("第 2 页不对：%+v %v %v", res2, err, api.last())
	}

	// 标准音质：直接用搜索结果里的 mp32 地址，不再请求
	before := api.count()
	src, err := m.MediaSource(first, "standard")
	if err != nil || src.URL != "https://media.example.com/stream/?trackid=a&format=mp32" {
		t.Fatalf("标准音质播放地址不对：%+v %v", src, err)
	}
	if api.count() != before {
		t.Error("标准音质不应该再请求接口")
	}

	// 低音质：重新请求 mp31 格式
	src, err = m.MediaSource(first, "low")
	if err != nil || src.URL != "https://media.example.com/stream/?trackid=a&format=mp31" ||
		api.last().Get("id") != "a" || api.last().Get("audioformat") != "mp31" {
		t.Fatalf("低音质播放地址不对：%+v %v %v", src, err, api.last())
	}

	// 无损：允许下载时用 flac 格式的在线播放地址（不用 audiodownload 下载地址）
	src, err = m.MediaSource(first, "super")
	if err != nil || src.URL != "https://media.example.com/stream/?trackid=a&format=flac" || api.last().Get("audioformat") != "flac" {
		t.Fatalf("允许下载时无损应该用 flac 播放地址：%+v %v %v", src, err, api.last())
	}
	// 不允许下载：不提供无损，返回 null，SwiftPaw 自动换成标准音质
	before = api.count()
	src, err = m.MediaSource(second, "super")
	if err != nil || src.URL != "https://media.example.com/stream/?trackid=b&format=mp32" {
		t.Fatalf("不允许下载时应该换成在线播放地址：%+v %v", src, err)
	}
	if api.count() != before {
		t.Error("不允许下载时不应该去请求无损地址")
	}

	// 歌词：纯文本，<br> 和 \r\n 换成换行
	lrc, err := m.Lyric(first)
	if err != nil || lrc != "第一行\n第二行\n第三行" || api.last().Get("include") != "lyrics" {
		t.Fatalf("歌词不对：%q %v %v", lrc, err, api.last())
	}
	if lrc, err := m.Lyric(second); err != nil || lrc != "" {
		t.Fatalf("没有歌词时应该是空的：%q %v", lrc, err)
	}

	// Jamendo 偶尔返回空结果：再查一次
	flaky := first
	flaky.ID, flaky.Extra = "flaky", []byte(`{"id":"flaky","title":"x"}`)
	for range 2 {
		if src, err := m.MediaSource(flaky, "low"); err != nil || src.URL != "https://media.example.com/stream/?trackid=flaky&format=mp31" {
			t.Fatalf("空结果时应该重试：%+v %v", src, err)
		}
	}

	// 歌曲已经下架（接口查不到）：没有播放地址
	gone := first
	gone.ID, gone.Extra = "missing", []byte(`{"id":"missing","title":"x"}`)
	if _, err := m.MediaSource(gone, "low"); err == nil {
		t.Error("查不到的歌曲应该报没有播放地址")
	}
}
