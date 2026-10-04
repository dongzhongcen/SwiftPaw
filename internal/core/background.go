package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 自定义背景图片：用户选的图片复制一份到数据文件夹的 backgrounds 里，原来的文件删了、挪了都不影响。
// 文件名是 background-<内容的哈希>.<扩展名>，换了图片文件名就会变，前端不会显示缓存里的旧图。

// MaxBackgroundSize 是背景图片的大小上限
const MaxBackgroundSize = 20 << 20 // 20 MB

const backgroundDirName = "backgrounds"

// ErrNotImage 表示选的文件不是支持的图片
var ErrNotImage = errors.New("只支持 JPG、PNG、WebP、GIF 格式的图片")

// ErrImageTooLarge 表示图片超过了大小上限
var ErrImageTooLarge = fmt.Errorf("图片太大了（超过 %d MB）", MaxBackgroundSize>>20)

var backgroundName = regexp.MustCompile(`^background-[0-9a-f]{16}\.(jpg|png|webp|gif)$`)

// backgroundTypes 是扩展名对应的 MIME 类型
var backgroundTypes = map[string]string{
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

// SetBackgroundImage 把 src 这张图片复制到数据文件夹，返回保存后的文件名（存进配置的 background.image）。
// 图片格式按文件内容判断，不看扩展名（Android 的文件选择器给的文件名不一定带扩展名）。
// 复制成功后会删掉以前的背景图片。
func (c *Core) SetBackgroundImage(src string) (string, error) {
	c.bgMu.Lock()
	defer c.bgMu.Unlock()
	dir, err := c.backgroundDir()
	if err != nil {
		return "", err
	}
	f, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("打不开图片：%w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", ErrNotImage
	}
	if info.Size() > MaxBackgroundSize {
		return "", ErrImageTooLarge
	}

	// 整个读进内存（最多 20 MB）：既要判断格式，又要算哈希
	data, err := io.ReadAll(io.LimitReader(f, MaxBackgroundSize+1))
	if err != nil {
		return "", err
	}
	if len(data) > MaxBackgroundSize {
		return "", ErrImageTooLarge
	}
	ext := imageExt(data)
	if ext == "" {
		return "", ErrNotImage
	}
	sum := sha256.Sum256(data)
	name := "background-" + hex.EncodeToString(sum[:8]) + ext

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, name)
	if _, err := os.Stat(target); err != nil {
		// 先写临时文件再改名，复制到一半出错时不会留下坏掉的图片
		tmp, err := os.CreateTemp(dir, "tmp-*")
		if err != nil {
			return "", err
		}
		_, werr := tmp.Write(data)
		cerr := tmp.Close()
		if werr == nil {
			werr = cerr
		}
		if werr == nil {
			werr = os.Rename(tmp.Name(), target)
		}
		if werr != nil {
			_ = os.Remove(tmp.Name())
			return "", fmt.Errorf("保存图片失败：%w", werr)
		}
	}
	c.removeBackgrounds(dir, name)
	return name, nil
}

// RemoveBackgroundImage 删掉保存的背景图片
func (c *Core) RemoveBackgroundImage() error {
	c.bgMu.Lock()
	defer c.bgMu.Unlock()
	dir, err := c.backgroundDir()
	if err != nil {
		return err
	}
	return c.removeBackgrounds(dir, "")
}

// BackgroundImagePath 返回保存的背景图片的完整路径，给桌面版的 /background 路由和 Android 的 WebView 用。
// name 必须是 SetBackgroundImage 返回的文件名，不能带路径，所以不能拿它读别的文件
func (c *Core) BackgroundImagePath(name string) (string, error) {
	if !validBackgroundName(name) {
		return "", ErrNotImage
	}
	dir, err := c.backgroundDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

// BackgroundContentType 返回背景图片文件名对应的 MIME 类型
func BackgroundContentType(name string) string {
	return backgroundTypes[strings.ToLower(filepath.Ext(name))]
}

func validBackgroundName(name string) bool {
	return backgroundName.MatchString(name)
}

func (c *Core) backgroundDir() (string, error) {
	if c.dataDir == "" {
		return "", errNoDataDir
	}
	return filepath.Join(c.dataDir, backgroundDirName), nil
}

// removeBackgrounds 删掉文件夹里除了 keep 以外的背景图片（和没写完的临时文件）
func (c *Core) removeBackgrounds(dir, keep string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var firstErr error
	for _, e := range entries {
		name := e.Name()
		if name == keep || e.IsDir() {
			continue
		}
		if validBackgroundName(name) || strings.HasPrefix(name, "tmp-") {
			if err := os.Remove(filepath.Join(dir, name)); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// imageExt 按文件开头的几个字节判断图片格式，返回扩展名；不是支持的格式时返回空字符串
func imageExt(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return ".jpg"
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return ".png"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return ".gif"
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return ".webp"
	}
	return ""
}

// BackgroundHandler 提供背景图片：GET /background?name=<文件名>。
// 桌面版挂在 Wails 的资源服务上（见根目录的 handler.go），Android 版由 WebView 拦截后调用 BackgroundImagePath
func (c *Core) BackgroundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		path, err := c.BackgroundImagePath(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", BackgroundContentType(name))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// 文件名里带着内容的哈希，内容变了文件名也会变，可以一直缓存
		w.Header().Set("Cache-Control", "max-age=31536000, immutable")
		http.ServeContent(w, r, name, info.ModTime(), f)
	})
}
