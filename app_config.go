package main

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

// ---- 配置 ----

// LoadConfig 读取上次保存的播放器状态
func (a *App) LoadConfig() (AppConfig, error) {
	// 先填默认值，旧版本的配置文件里没有这些字段时就用默认值
	config := AppConfig{PlayMode: string(queue.Sequence), Volume: 1, Theme: "dark"}
	path, err := configFilePath()
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
func (a *App) SaveConfig(config AppConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// dataDir 是保存配置和数据库的文件夹，Windows 上是 %AppData%\SwiftPaw
func dataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "SwiftPaw"), nil
}

func configFilePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
