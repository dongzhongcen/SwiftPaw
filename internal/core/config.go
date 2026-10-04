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
}

// LoadConfig 读取上次保存的播放器状态
func (c *Core) LoadConfig() (AppConfig, error) {
	// 先填默认值，旧版本的配置文件里没有这些字段时就用默认值
	config := AppConfig{PlayMode: string(queue.Sequence), Volume: 1, Theme: "dark"}
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

	err = json.Unmarshal(data, &config)
	return config, err
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
