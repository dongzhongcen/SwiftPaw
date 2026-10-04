package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"musicplayer/internal/jsrt"
	"musicplayer/internal/music"
)

// maxPluginSize 是插件文件的大小上限
const maxPluginSize = 5 << 20

// Options 是创建 Manager 的参数
type Options struct {
	Dir        string        // 插件文件夹
	StatePath  string        // 保存启用状态的 json 文件
	AppVersion string        // 通过 env.appVersion 告诉插件
	Timeout    time.Duration // 每次调用插件的超时，0 表示默认 15 秒
	HTTPClient *http.Client  // 从网址安装插件时用的客户端，nil 用默认的
	Logger     func(pluginID, level, message string)
}

// Manager 管理所有插件。所有公开方法都可以在多个 goroutine 里同时调用。
type Manager struct {
	opts Options

	mu       sync.Mutex
	plugins  []*loaded
	disabled map[string]bool              // 被停用的插件 id
	values   map[string]map[string]string // 用户填的设置：插件 id -> (变量名 -> 值)
}

// loaded 是一个已加载（或者加载失败）的插件
type loaded struct {
	info Info
	file string
	rt   *jsrt.Runtime // 加载失败时为 nil
}

// state 是 plugins.json 的内容
type state struct {
	Disabled []string `json:"disabled"`
	// UserVariables 是用户在“插件设置”里填的值：插件 id -> (变量名 -> 值)。
	// 旧版本的 plugins.json 里没有这一项，读出来就是空的，不需要额外迁移
	UserVariables map[string]map[string]string `json:"userVariables,omitempty"`
}

// maxValueLength 是每个设置值的最大长度（字符数），防止误粘贴一大段内容
const maxValueLength = 2000

// NewManager 创建插件管理器，还不会加载插件，需要调用 LoadAll
func NewManager(opts Options) *Manager {
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Manager{opts: opts, disabled: map[string]bool{}, values: map[string]map[string]string{}}
}

// LoadAll 重新加载插件文件夹里所有的 .js 文件
func (m *Manager) LoadAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadAllLocked()
}

func (m *Manager) loadAllLocked() error {
	m.closeAllLocked()
	m.readState()
	if err := os.MkdirAll(m.opts.Dir, 0o755); err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(m.opts.Dir, "*.js"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	seen := map[string]string{} // platform -> 先加载的插件 id
	for _, file := range files {
		p := m.loadFile(file)
		if p.info.Error == "" {
			if other, dup := seen[p.info.Platform]; dup {
				p.info.Error = fmt.Sprintf("平台名「%s」和插件 %s 重复了", p.info.Platform, other)
				p.close()
			} else {
				seen[p.info.Platform] = p.info.ID
			}
		}
		m.plugins = append(m.plugins, p)
	}
	return nil
}

// loadFile 加载一个插件文件。出错时不返回 error，而是记在 info.Error 里，界面上能看到原因。
func (m *Manager) loadFile(file string) *loaded {
	id := strings.TrimSuffix(filepath.Base(file), ".js")
	p := &loaded{file: file, info: Info{ID: id, Enabled: !m.disabled[id]}}
	source, err := readLimited(file)
	if err != nil {
		p.info.Error = err.Error()
		return p
	}
	rt, info, err := m.evaluate(id, source, m.values[id])
	if rt != nil {
		info.Missing = rt.MissingModules()
	}
	info.ID, info.Enabled = id, p.info.Enabled
	p.info = info
	if err != nil {
		p.info.Error = err.Error()
		if rt != nil {
			rt.Close()
		}
		return p
	}
	p.rt = rt
	return p
}

// evaluate 在新的运行环境里执行插件代码，读出插件信息。
// vars 是用户给这个插件填的设置，插件通过 env.getUserVariables() 读到；没有时传 nil
func (m *Manager) evaluate(name string, source []byte, vars map[string]string) (*jsrt.Runtime, Info, error) {
	rt, err := jsrt.New(jsrt.Options{
		Name:    name,
		Timeout: m.opts.Timeout,
		// 复制一份再交给运行环境，之后修改 m.values 不会影响正在运行的插件
		Env: jsrt.Env{AppVersion: m.opts.AppVersion, UserVariables: maps.Clone(vars)},
		Logger: func(level, msg string) {
			if m.opts.Logger != nil {
				m.opts.Logger(name, level, msg)
			}
		},
	})
	if err != nil {
		return nil, Info{}, err
	}
	if err := rt.Load(string(source)); err != nil {
		return rt, Info{}, err
	}
	metaJSON, funcs, err := rt.Meta()
	if err != nil {
		return rt, Info{}, err
	}
	var meta struct {
		Platform            any    `json:"platform"`
		Version             any    `json:"version"`
		Author              any    `json:"author"`
		SrcURL              string `json:"srcUrl"`
		SupportedSearchType []any  `json:"supportedSearchType"`
		UserVariables       []any  `json:"userVariables"`
	}
	_ = json.Unmarshal(metaJSON, &meta)
	info := Info{
		Platform:  strings.TrimSpace(fmt.Sprint(nilToEmpty(meta.Platform))),
		Version:   fmt.Sprint(nilToEmpty(meta.Version)),
		Author:    fmt.Sprint(nilToEmpty(meta.Author)),
		SrcURL:    meta.SrcURL,
		CanSearch: slices.Contains(funcs, "search"),
		CanPlay:   slices.Contains(funcs, "getMediaSource"),
		CanLyric:  slices.Contains(funcs, "getLyric"),
	}
	for _, t := range meta.SupportedSearchType {
		info.SupportedSearchType = append(info.SupportedSearchType, fmt.Sprint(t))
	}
	info.UserVariables = parseUserVariables(meta.UserVariables)
	if info.Platform == "" {
		return rt, info, errors.New("插件没有设置 platform（平台名）")
	}
	return rt, info, nil
}

// parseUserVariables 读出插件声明的用户变量。每一项要有 key，name 和 hint 可以不写：
// 没有 name 时用 key 当名字。格式不对的项和重复的 key 会被跳过，不影响插件加载
func parseUserVariables(list []any) []UserVariable {
	vars := []UserVariable{}
	seen := map[string]bool{}
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.TrimSpace(textOf(obj["key"]))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		v := UserVariable{Key: key, Name: strings.TrimSpace(textOf(obj["name"])), Hint: strings.TrimSpace(textOf(obj["hint"]))}
		if v.Name == "" {
			v.Name = key
		}
		vars = append(vars, v)
	}
	return vars
}

// textOf 把 JSON 里的字符串或数字变成文字，其它类型（对象、数组等）当作没写
func textOf(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64, bool:
		return fmt.Sprint(x)
	}
	return ""
}

func nilToEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// List 返回所有插件的信息
func (m *Manager) List() []Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := make([]Info, 0, len(m.plugins))
	for _, p := range m.plugins {
		info := p.info
		if p.rt != nil {
			info.Missing = p.rt.MissingModules() // 运行中 require 失败的模块也算上
		}
		list = append(list, info)
	}
	return list
}

// SetEnabled 启用或停用插件
func (m *Manager) SetEnabled(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.findByID(id)
	if p == nil {
		return fmt.Errorf("找不到插件 %s", id)
	}
	p.info.Enabled = enabled
	if enabled {
		delete(m.disabled, id)
	} else {
		m.disabled[id] = true
	}
	return m.writeState()
}

// Uninstall 删除插件文件
func (m *Manager) Uninstall(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.findByID(id)
	if p == nil {
		return fmt.Errorf("找不到插件 %s", id)
	}
	p.close()
	if err := os.Remove(p.file); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(m.disabled, id)
	delete(m.values, id) // 卸载时把用户填的设置也删掉
	m.plugins = slices.DeleteFunc(m.plugins, func(x *loaded) bool { return x == p })
	return m.writeState()
}

// ---- 插件设置（用户变量） ----

// UserVariables 返回用户给某个插件填的设置（变量名 -> 值），没填过时返回空的 map
func (m *Manager) UserVariables(id string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findByID(id) == nil {
		return nil, fmt.Errorf("找不到插件 %s", id)
	}
	values := maps.Clone(m.values[id])
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}

// SetUserVariables 保存用户给插件填的设置，然后重新加载这个插件，马上生效，不用重启。
// 只保存插件声明过的变量；值的前后空格会去掉，空值表示“没填”。返回重新加载后的插件信息
func (m *Manager) SetUserVariables(id string, values map[string]string) (Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.findByID(id)
	if p == nil {
		return Info{}, fmt.Errorf("找不到插件 %s", id)
	}
	if len(p.info.UserVariables) == 0 {
		return Info{}, fmt.Errorf("插件「%s」没有需要填写的设置", p.info.Platform)
	}
	saved := map[string]string{}
	for _, v := range p.info.UserVariables {
		value := strings.TrimSpace(values[v.Key])
		if utf8.RuneCountInString(value) > maxValueLength {
			return Info{}, fmt.Errorf("「%s」太长了（最多 %d 个字符）", v.Name, maxValueLength)
		}
		if value != "" {
			saved[v.Key] = value
		}
	}
	if len(saved) == 0 {
		delete(m.values, id)
	} else {
		m.values[id] = saved
	}
	if err := m.writeState(); err != nil {
		return Info{}, err
	}
	return m.reloadLocked(p).info, nil
}

// reloadLocked 重新加载一个插件（用新的设置创建新的运行环境），替换掉列表里的旧插件。
// 调用前要先拿到 m.mu 锁
func (m *Manager) reloadLocked(old *loaded) *loaded {
	old.close()
	p := m.loadFile(old.file)
	// 和 loadAllLocked 一样：同一个平台名只能有一个插件在用
	if p.info.Error == "" {
		for _, other := range m.plugins {
			if other != old && other.info.Error == "" && other.info.Platform == p.info.Platform {
				p.info.Error = fmt.Sprintf("平台名「%s」和插件 %s 重复了", p.info.Platform, other.info.ID)
				p.close()
				break
			}
		}
	}
	for i, x := range m.plugins {
		if x == old {
			m.plugins[i] = p
		}
	}
	return p
}

// Close 停止所有插件
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeAllLocked()
}

func (m *Manager) closeAllLocked() {
	for _, p := range m.plugins {
		p.close()
	}
	m.plugins = nil
}

func (p *loaded) close() {
	if p.rt != nil {
		p.rt.Close()
		p.rt = nil
	}
}

func (m *Manager) findByID(id string) *loaded {
	for _, p := range m.plugins {
		if p.info.ID == id {
			return p
		}
	}
	return nil
}

// runtimeFor 找到某个平台可用的插件运行环境
func (m *Manager) runtimeFor(platform string) (*jsrt.Runtime, Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.plugins {
		if p.info.Platform != platform || p.info.Error != "" {
			continue
		}
		if !p.info.Enabled {
			return nil, p.info, fmt.Errorf("插件「%s」已停用，请在插件页面启用", platform)
		}
		return p.rt, p.info, nil
	}
	return nil, Info{}, fmt.Errorf("没有安装「%s」插件，或者插件加载失败", platform)
}

// ---- 调用插件 ----

// Search 用某个插件搜索。page 从 1 开始，searchType 默认是 music。
func (m *Manager) Search(platform, query string, page int, searchType string) (SearchResult, error) {
	rt, info, err := m.runtimeFor(platform)
	if err != nil {
		return SearchResult{}, err
	}
	if !info.CanSearch {
		return SearchResult{}, fmt.Errorf("插件「%s」不支持搜索", platform)
	}
	if page < 1 {
		page = 1
	}
	if searchType == "" {
		searchType = "music"
	}
	raw, err := rt.Call("search", query, page, searchType)
	if err != nil {
		return SearchResult{}, err
	}
	var result struct {
		IsEnd *bool             `json:"isEnd"`
		Data  []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return SearchResult{}, fmt.Errorf("插件「%s」返回的搜索结果格式不对", platform)
	}
	out := SearchResult{IsEnd: result.IsEnd == nil || *result.IsEnd, Data: []music.Song{}}
	for _, item := range result.Data {
		song, err := SongFromItem(platform, item)
		if err != nil {
			continue // 跳过格式不对的条目
		}
		out.Data = append(out.Data, song)
	}
	if len(result.Data) == 0 {
		out.IsEnd = true
	}
	return out, nil
}

// qualities 是音质，按优先级排列；插件不支持请求的音质时会依次尝试别的
var qualities = []string{"standard", "high", "low", "super"}

// MediaSource 让插件解析一首在线歌曲的播放地址
func (m *Manager) MediaSource(song music.Song, quality string) (MediaSource, error) {
	rt, info, err := m.runtimeFor(song.Source)
	if err != nil {
		return MediaSource{}, err
	}
	item, err := ItemFromSong(song)
	if err != nil {
		return MediaSource{}, err
	}
	if !info.CanPlay {
		// 没有 getMediaSource 的插件，搜索结果里直接带了 url
		var direct struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(item, &direct)
		if direct.URL == "" {
			return MediaSource{}, fmt.Errorf("插件「%s」没有提供播放地址", song.Source)
		}
		return MediaSource{URL: direct.URL}, nil
	}

	if quality == "" {
		quality = "standard"
	}
	order := append([]string{quality}, slices.DeleteFunc(slices.Clone(qualities), func(q string) bool { return q == quality })...)
	for _, q := range order {
		raw, err := rt.Call("getMediaSource", item, q)
		if err != nil {
			return MediaSource{}, err
		}
		var src struct {
			URL       string         `json:"url"`
			Headers   map[string]any `json:"headers"`
			UserAgent string         `json:"userAgent"`
		}
		if json.Unmarshal(raw, &src) != nil || src.URL == "" {
			continue // 这个音质没有，试下一个
		}
		out := MediaSource{URL: src.URL, Headers: map[string]string{}}
		for k, v := range src.Headers {
			out.Headers[k] = fmt.Sprint(v)
		}
		if src.UserAgent != "" {
			out.Headers["User-Agent"] = src.UserAgent
		}
		return out, nil
	}
	return MediaSource{}, fmt.Errorf("插件「%s」没有返回这首歌的播放地址", song.Source)
}

// Lyric 让插件返回在线歌曲的歌词（LRC 文本）。插件不支持歌词时返回空字符串。
func (m *Manager) Lyric(song music.Song) (string, error) {
	rt, info, err := m.runtimeFor(song.Source)
	if err != nil {
		return "", err
	}
	if !info.CanLyric {
		return "", nil
	}
	item, err := ItemFromSong(song)
	if err != nil {
		return "", err
	}
	raw, err := rt.Call("getLyric", item)
	if err != nil {
		return "", err
	}
	var lyric struct {
		RawLrc string `json:"rawLrc"`
		Lrc    string `json:"lrc"` // 有的插件给的是歌词文件地址
	}
	_ = json.Unmarshal(raw, &lyric)
	if lyric.RawLrc == "" && httpURL(lyric.Lrc) != "" {
		data, err := m.download(lyric.Lrc)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return lyric.RawLrc, nil
}

// ---- 状态文件 ----

func (m *Manager) readState() {
	m.disabled = map[string]bool{}
	m.values = map[string]map[string]string{}
	data, err := os.ReadFile(m.opts.StatePath)
	if err != nil {
		return
	}
	var s state
	if json.Unmarshal(data, &s) == nil {
		for _, id := range s.Disabled {
			m.disabled[id] = true
		}
		for id, vars := range s.UserVariables {
			if len(vars) > 0 {
				m.values[id] = vars
			}
		}
	}
}

func (m *Manager) writeState() error {
	s := state{Disabled: []string{}, UserVariables: m.values}
	for id := range m.disabled {
		s.Disabled = append(s.Disabled, id)
	}
	sort.Strings(s.Disabled)
	data, _ := json.MarshalIndent(s, "", "  ")
	if err := os.MkdirAll(filepath.Dir(m.opts.StatePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.opts.StatePath, data, 0o644)
}

func readLimited(file string) ([]byte, error) {
	info, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxPluginSize {
		return nil, fmt.Errorf("插件文件太大（超过 %d MB）", maxPluginSize>>20)
	}
	return os.ReadFile(file)
}
