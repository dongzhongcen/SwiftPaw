// Package music 负责“找歌”和“读歌曲信息”：扫描文件夹、判断是不是音频文件、读取标签。
package music

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

// Song 表示一首歌
type Song struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Format string `json:"format"` // 文件格式，比如 MP3、FLAC
}

// audioTypes 列出支持的扩展名，以及播放时告诉浏览器的 Content-Type。
// Windows 上的 WebView2 和 Chrome 内核一样，这些格式都能直接播放。
var audioTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".wav":  "audio/wav",
}

// IsAudio 判断一个路径是不是支持的音频文件（只看扩展名，不区分大小写）
func IsAudio(path string) bool {
	_, ok := audioTypes[strings.ToLower(filepath.Ext(path))]
	return ok
}

// ContentType 返回音频文件对应的 Content-Type，不支持的格式返回空字符串
func ContentType(path string) string {
	return audioTypes[strings.ToLower(filepath.Ext(path))]
}

// Scan 扫描文件夹（包括子文件夹）里所有支持的音频文件
func Scan(dir string) ([]Song, error) {
	songs := []Song{}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 某个子文件夹读不了就跳过，不影响其他文件
		}
		if d.IsDir() || !IsAudio(path) {
			return nil
		}
		songs = append(songs, ReadInfo(path))
		return nil
	})

	return songs, err
}

// ReadInfo 读取一首歌的标签；读不到标签时用文件名当歌名
func ReadInfo(path string) Song {
	fileName := filepath.Base(path)
	ext := filepath.Ext(fileName)
	fallbackName := strings.TrimSuffix(fileName, ext)
	song := Song{
		Name:   fallbackName,
		Path:   path,
		Title:  fallbackName,
		Artist: "未知歌手",
		Format: strings.ToUpper(strings.TrimPrefix(ext, ".")),
	}

	file, err := os.Open(path)
	if err != nil {
		return song
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return song // 比如 wav 一般没有标签，直接用文件名
	}

	if title := strings.TrimSpace(metadata.Title()); title != "" {
		song.Title = title
		song.Name = title
	}
	if artist := strings.TrimSpace(metadata.Artist()); artist != "" {
		song.Artist = artist
	}
	song.Album = strings.TrimSpace(metadata.Album())

	return song
}
