package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// InstallFile 从本地文件安装插件（复制到插件文件夹）
func (m *Manager) InstallFile(path string) (Info, error) {
	if !strings.EqualFold(filepath.Ext(path), ".js") {
		return Info{}, fmt.Errorf("插件文件必须是 .js 文件")
	}
	source, err := readLimited(path)
	if err != nil {
		return Info{}, err
	}
	return m.install(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), source)
}

// InstallURL 从网址安装插件。网址可以直接指向 .js 文件，
// 也可以是插件列表 JSON：{"plugins":[{"name":"...","url":"https://..."}]}，会逐个安装。
func (m *Manager) InstallURL(rawURL string) ([]Info, error) {
	data, err := m.download(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	var list struct {
		Plugins []struct {
			URL string `json:"url"`
		} `json:"plugins"`
	}
	if json.Unmarshal(data, &list) == nil && len(list.Plugins) > 0 {
		var installed []Info
		var errs []string
		for _, p := range list.Plugins {
			src, err := m.download(p.URL)
			if err == nil {
				var info Info
				if info, err = m.install(nameFromURL(p.URL), src); err == nil {
					installed = append(installed, info)
					continue
				}
			}
			errs = append(errs, fmt.Sprintf("%s：%v", p.URL, err))
		}
		if len(installed) == 0 {
			return nil, fmt.Errorf("插件列表里的插件都没装上：%s", strings.Join(errs, "；"))
		}
		return installed, nil
	}
	info, err := m.install(nameFromURL(rawURL), data)
	if err != nil {
		return nil, err
	}
	return []Info{info}, nil
}

// nameFromURL 取网址里的文件名（不含 .js）当插件的临时名字，安装失败时错误信息里用
func nameFromURL(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "新插件"
	}
	name := strings.TrimSuffix(path.Base(u.Path), ".js")
	if name == "" || name == "." || name == "/" {
		return "新插件"
	}
	return name
}

// install 先在临时的运行环境里试运行插件，确认没问题后再保存到插件文件夹。
// name 只用在错误信息里（这时还不知道插件的平台名）
func (m *Manager) install(name string, source []byte) (Info, error) {
	rt, info, err := m.evaluate(name, source, nil) // 试运行时还不知道插件 id，不传设置
	if rt != nil {
		rt.Close()
	}
	if err != nil {
		return Info{}, fmt.Errorf("安装失败：%v", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.MkdirAll(m.opts.Dir, 0o755); err != nil {
		return Info{}, err
	}
	// 同一个平台的旧版本插件先删掉（相当于更新）
	id := safeFileName(info.Platform)
	moved := false
	for _, p := range m.plugins {
		if p.info.Platform == info.Platform {
			p.close()
			_ = os.Remove(p.file)
			// 旧文件名和新的不一样时（比如手动复制进来的），把用户填的设置搬到新 id 下，更新后不用重新填
			if vars, ok := m.values[p.info.ID]; ok && p.info.ID != id {
				if _, has := m.values[id]; !has {
					m.values[id] = vars
				}
				delete(m.values, p.info.ID)
				moved = true
			}
		}
	}
	if moved {
		if err := m.writeState(); err != nil {
			return Info{}, err
		}
	}
	target := filepath.Join(m.opts.Dir, id+".js")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, source, 0o644); err != nil {
		return Info{}, err
	}
	if err := os.Rename(tmp, target); err != nil {
		return Info{}, err
	}
	if err := m.loadAllLocked(); err != nil {
		return Info{}, err
	}
	if p := m.findByID(id); p != nil {
		if p.info.Error != "" {
			return p.info, fmt.Errorf("插件已保存，但加载失败：%s", p.info.Error)
		}
		return p.info, nil
	}
	return Info{}, fmt.Errorf("插件保存失败")
}

// download 下载一个 http/https 地址的内容（最大 5 MB）
func (m *Manager) download(rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("只支持 http/https 网址")
	}
	resp, err := m.opts.HTTPClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("下载失败：%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败：服务器返回 %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPluginSize+1))
	if err != nil {
		return nil, fmt.Errorf("下载失败：%v", err)
	}
	if len(data) > maxPluginSize {
		return nil, fmt.Errorf("文件太大（超过 %d MB）", maxPluginSize>>20)
	}
	return data, nil
}

// safeFileName 把平台名变成安全的文件名：只保留文字、数字、- 和 _
func safeFileName(platform string) string {
	name := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, platform)
	name = strings.Trim(name, "-")
	if name == "" {
		name = "plugin"
	}
	return name
}
