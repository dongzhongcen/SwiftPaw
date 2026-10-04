package stream

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 源站：要求带 Referer，支持 Range（http.ServeContent 会处理 Range）
func origin(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://music.example.com/" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/song.mp3" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		http.ServeContent(w, r, "x", time.Time{}, strings.NewReader("0123456789"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, p *Proxy, target string, rangeHeader string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	return rec.Result()
}

func TestProxyHeadersAndRange(t *testing.T) {
	src := origin(t)
	p := New(nil)
	local, err := p.Register(src.URL+"/song.mp3", map[string]string{"Referer": "https://music.example.com/"})
	if err != nil || !strings.HasPrefix(local, "/stream?id=") {
		t.Fatalf("Register：%q %v", local, err)
	}

	resp := get(t, p, local, "")
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "0123456789" {
		t.Fatalf("完整请求：%d %q", resp.StatusCode, body)
	}
	if resp.Header.Get("Content-Type") != "audio/mpeg" {
		t.Errorf("octet-stream 应该按扩展名换成 audio/mpeg，实际 %q", resp.Header.Get("Content-Type"))
	}

	resp = get(t, p, local, "bytes=2-5")
	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusPartialContent || string(body) != "2345" ||
		resp.Header.Get("Content-Range") != "bytes 2-5/10" {
		t.Fatalf("Range 请求：%d %q %q", resp.StatusCode, body, resp.Header.Get("Content-Range"))
	}
}

func TestProxyUserAgent(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Header.Get("User-Agent")+"|"+r.Header.Get("Cookie"))
	}))
	defer src.Close()
	p := New(nil)

	withUA, _ := p.Register(src.URL, map[string]string{"User-Agent": "PluginUA", "Cookie": "k=v"})
	body, _ := io.ReadAll(get(t, p, withUA, "").Body)
	if string(body) != "PluginUA|k=v" {
		t.Errorf("插件的请求头没有带上：%q", body)
	}

	noUA, _ := p.Register(src.URL, nil)
	body, _ = io.ReadAll(get(t, p, noUA, "").Body)
	if !strings.HasPrefix(string(body), "Mozilla/5.0") {
		t.Errorf("没指定时应该用默认 User-Agent：%q", body)
	}
}

func TestProxyNeedsReferer(t *testing.T) {
	src := origin(t)
	p := New(nil)
	local, _ := p.Register(src.URL+"/a", nil)
	resp := get(t, p, local, "")
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("源站拒绝时应该返回 502，实际 %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "音乐服务器返回 403") {
		t.Fatalf("错误信息里要有源站的状态码，实际 %q", body)
	}
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "audio/") {
		t.Fatalf("错误页面不能标成音频：%q", ct)
	}
}

func TestProxyRejects(t *testing.T) {
	p := New(nil)
	for _, bad := range []string{"file:///etc/passwd", "ftp://x/a.mp3", "javascript:alert(1)", "/relative", ""} {
		if _, err := p.Register(bad, nil); err != ErrBadURL {
			t.Errorf("%q 应该被拒绝，实际 %v", bad, err)
		}
	}
	if resp := get(t, p, "/stream?id=unknown", ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("没登记过的 id 应该 404，实际 %d", resp.StatusCode)
	}
}

func TestProxyEvictsOldEntries(t *testing.T) {
	p := New(nil)
	first, _ := p.Register("https://example.com/0.mp3", nil)
	for i := 0; i < maxEntries; i++ {
		p.Register("https://example.com/x.mp3", nil)
	}
	if resp := get(t, p, first, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("最早的地址应该被清掉，实际 %d", resp.StatusCode)
	}
}
