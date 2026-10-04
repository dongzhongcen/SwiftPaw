package main

import "musicplayer/internal/core"

// ---- 配置（保存在数据文件夹的 config.json）----

// LoadConfig 读取上次保存的播放器状态
func (a *App) LoadConfig() (core.AppConfig, error) {
	return a.core.LoadConfig()
}

// SaveConfig 保存播放器状态，供下次启动时恢复
func (a *App) SaveConfig(config core.AppConfig) error {
	return a.core.SaveConfig(config)
}
