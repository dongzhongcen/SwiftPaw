package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Song 表示一首歌
type Song struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// AppConfig 保存播放器需要记住的用户状态
type AppConfig struct {
	LastFolder string `json:"lastFolder"`
	LastSong   string `json:"lastSong"`
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SelectFolder 弹出选择文件夹的窗口，返回选中的路径
func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择音乐文件夹",
	})
}

// ScanMusic 扫描文件夹（包括子文件夹）里的所有 mp3
func (a *App) ScanMusic(dir string) ([]Song, error) {
	songs := []Song{}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".mp3" {
			songs = append(songs, readSongInfo(path, d.Name()))
		}
		return nil
	})

	return songs, err
}

// LoadConfig 读取上次保存的播放器状态
func (a *App) LoadConfig() (AppConfig, error) {
	config := AppConfig{}
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

func configFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "SwiftPaw", "config.json"), nil
}

func readSongInfo(path string, fileName string) Song {
	fallbackName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	song := Song{
		Name:   fallbackName,
		Path:   path,
		Title:  fallbackName,
		Artist: "未知歌手",
	}

	file, err := os.Open(path)
	if err != nil {
		return song
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return song
	}

	if title := strings.TrimSpace(metadata.Title()); title != "" {
		song.Title = title
		song.Name = title
	}
	if artist := strings.TrimSpace(metadata.Artist()); artist != "" {
		song.Artist = artist
	}

	return song
}
