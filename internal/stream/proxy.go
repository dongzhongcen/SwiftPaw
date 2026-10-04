// Package stream 是在线歌曲的播放代理。
//
// 为什么需要代理：很多音乐地址要求带上特定的 Referer、User-Agent 才能访问，
// 但网页里的 <audio> 没法自己设置这些请求头。所以前端播放 /stream?id=xxx，
// 由 Go 带上插件给的请求头去请求真正的地址，再把数据转发给前端。
//
// 安全起见，代理只转发“插件解析出来、登记过”的地址（用随机 id 对应），不是任意网址都能代理。
package stream

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"musicplayer/internal/music"
)

// maxEntries 是最多记住多少个登记过的地址，超过时删掉最早的
const maxEntries = 200

// DefaultUserAgent 是插件没指定 User-Agent 时用的
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

// ErrBadURL 表示地址不是 http/https
var ErrBadURL = errors.New("只支持 http/https 的播放地址")

type entry struct {
	url     string
	headers map[string]string
}

// Proxy 是播放代理，实现了 http.Handler
type Proxy struct {
	client *http.Client

	mu      sync.Mutex
	entries map[string]entry
	order   []string
}

// New 创建代理。client 为 nil 时用默认客户端（只允许重定向到 http/https）。
func New(client *http.Client) *Proxy {
	if client == nil {
		client = &http.Client{
			// 不设置总超时：一首歌可能要播好几分钟；连接和响应头有单独的超时
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				ForceAttemptHTTP2:     true,
				TLSHandshakeTimeout:   15 * time.Second,
				ResponseHeaderTimeout: 20 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("重定向次数太多")
				}
				if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
					return ErrBadURL
				}
				return nil
			},
		}
	}
	return &Proxy{client: client, entries: map[string]entry{}}
}

// Register 登记一个播放地址和它需要的请求头，返回前端可以直接播放的地址（/stream?id=...）
func (p *Proxy) Register(rawURL string, headers map[string]string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", ErrBadURL
	}
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf)

	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries[id] = entry{url: u.String(), headers: headers}
	p.order = append(p.order, id)
	for len(p.order) > maxEntries {
		delete(p.entries, p.order[0])
		p.order = p.order[1:]
	}
	return "/stream?id=" + id, nil
}

// passHeaders 是从前端请求里转发过去的请求头（Range 让拖动进度条时只取需要的部分）
var passHeaders = []string{"Range", "If-Range"}

// copyHeaders 是从源站响应里转发回前端的响应头
var copyHeaders = []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges",
	"Last-Modified", "ETag", "Cache-Control"}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	e, ok := p.entries[r.URL.Query().Get("id")]
	p.mu.Unlock()
	if !ok {
		http.Error(w, "播放地址已过期，请重新播放", http.StatusNotFound)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, e.url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, h := range passHeaders {
		if v := r.Header.Get(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	for k, v := range e.headers {
		req.Header.Set(k, v) // 插件给的请求头（Referer、User-Agent、Cookie 等）优先
	}

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("连接音乐地址失败：%v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 音乐服务器出错（比如 403、404、500）时，不把它的错误页面当成音频转发
	if resp.StatusCode >= 400 {
		http.Error(w, fmt.Sprintf("音乐服务器返回 %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	for _, h := range copyHeaders {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	// 有些服务器返回 application/octet-stream，WebView 可能认不出来，按扩展名补一个
	if ct := resp.Header.Get("Content-Type"); ct == "" || strings.HasPrefix(ct, "application/octet-stream") {
		if guess := music.ContentType(path.Base(resp.Request.URL.Path)); guess != "" {
			w.Header().Set("Content-Type", guess)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
