// Package winstate 记住窗口大小：关闭时保存到 window.json，下次启动时按这个大小打开。
//
// 单独放一个文件而不是放进 config.json，是因为窗口大小要在 Wails 启动之前就读出来，
// 那时前端还没加载，config.json 由前端负责读写，两边分开更简单。
package winstate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Size 是窗口的宽和高（单位和 Wails 的 options.App.Width / Height 一样）
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// 窗口大小的范围：不能比最小尺寸还小，也不能大得离谱（比如文件被改坏了）
const (
	MinWidth  = 860
	MinHeight = 600
	maxSide   = 10000
)

// Default 是第一次启动时的窗口大小
var Default = Size{Width: 1100, Height: 760}

// Load 读取保存的窗口大小；文件不存在或者内容不对时返回 Default
func Load(path string) Size {
	data, err := os.ReadFile(path)
	if err != nil {
		return Default
	}
	var size Size
	if json.Unmarshal(data, &size) != nil {
		return Default
	}
	return Clamp(size)
}

// Save 保存窗口大小（会先 Clamp）
func Save(path string, size Size) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Clamp(size), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Clamp 把窗口大小限制在合理范围内；宽或高不是正数时整个用 Default
func Clamp(size Size) Size {
	if size.Width <= 0 || size.Height <= 0 || size.Width > maxSide || size.Height > maxSide {
		return Default
	}
	size.Width = max(size.Width, MinWidth)
	size.Height = max(size.Height, MinHeight)
	return size
}
