package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"musicplayer/internal/queue"
)

// AppConfig 保存播放器需要记住的用户状态
type AppConfig struct {
	LastFolder string  `json:"lastFolder"`
	LastSong   string  `json:"lastSong"`
	PlayMode   string  `json:"playMode"` // sequence / repeat-one / shuffle
	Volume     float64 `json:"volume"`   // 0 到 1
	Theme      string  `json:"theme"`    // dark / light / system
	// Style 是风格主题：空字符串表示不用风格，按 Theme 显示深色/浅色；
	// vinyl（唱片行）/ metronome（节拍器）/ neon（霓虹）时 Theme 不起作用
	Style string `json:"style"`
	// ReduceBlur 是“减少模糊效果”：空字符串表示自动（Android 上减少，桌面版不减少），on / off 是用户选的
	ReduceBlur string           `json:"reduceBlur"`
	Background BackgroundConfig `json:"background"`
}

// BackgroundConfig 是自定义背景图片的设置
type BackgroundConfig struct {
	Image string `json:"image"` // 复制到数据文件夹里的图片文件名（见 SetBackgroundImage），空表示不用背景图
	Blur  int    `json:"blur"`  // 模糊程度，0 到 30（像素）
	Dim   int    `json:"dim"`   // 遮罩浓度，0 到 80（百分比）
}

// 可以选的主题和风格
var (
	themes     = map[string]bool{"dark": true, "light": true, "system": true}
	styles     = map[string]bool{"": true, "vinyl": true, "metronome": true, "neon": true}
	reduceBlur = map[string]bool{"": true, "on": true, "off": true}
)

const (
	maxBackgroundBlur = 30
	maxBackgroundDim  = 80
)

// defaultConfig 是第一次运行（或者旧版本的配置文件里没有某个字段）时的配置
func defaultConfig() AppConfig {
	return AppConfig{PlayMode: string(queue.Sequence), Volume: 1, Theme: "dark", Background: BackgroundConfig{Dim: 40}}
}

// normalize 把不认识的值换成默认值、超出范围的数字限制到范围里，免得手动改坏的配置文件让界面出错
func (config *AppConfig) normalize() {
	if !themes[config.Theme] {
		config.Theme = "dark"
	}
	if !styles[config.Style] {
		config.Style = ""
	}
	if !reduceBlur[config.ReduceBlur] {
		config.ReduceBlur = ""
	}
	if config.Background.Image != "" && !validBackgroundName(config.Background.Image) {
		config.Background.Image = ""
	}
	config.Background.Blur = clamp(config.Background.Blur, 0, maxBackgroundBlur)
	config.Background.Dim = clamp(config.Background.Dim, 0, maxBackgroundDim)
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}

// LoadConfig 读取上次保存的播放器状态
func (c *Core) LoadConfig() (AppConfig, error) {
	// 先填默认值，旧版本的配置文件里没有这些字段时就用默认值
	config := defaultConfig()
	path, err := c.configFilePath()
	if err != nil {
		return config, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return config, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return defaultConfig(), err
	}
	config.normalize()
	return config, nil
}

// SaveConfig 保存播放器状态，供下次启动时恢复
func (c *Core) SaveConfig(config AppConfig) error {
	path, err := c.configFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	config.normalize()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (c *Core) configFilePath() (string, error) {
	if c.dataDir == "" {
		return "", errNoDataDir
	}
	return filepath.Join(c.dataDir, "config.json"), nil
}
