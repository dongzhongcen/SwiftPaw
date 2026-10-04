// Package music 负责“找歌”和“读歌曲信息”：扫描文件夹、判断是不是音频文件、读取标签。
package music

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

// LocalSource 是本地歌曲的来源名
const LocalSource = "local"

// Song 表示一首歌。本地歌曲和插件搜到的在线歌曲都用它，这样队列、歌单、收藏、最近播放都能通用。
type Song struct {
	Name   string `json:"name"`
	Path   string `json:"path"` // 本地歌曲的文件路径；在线歌曲为空
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Format string `json:"format"` // 文件格式，比如 MP3、FLAC

	// 下面这些字段是给在线歌曲用的
	Source   string          `json:"source"`          // 来源："local" 或者插件的 platform 名
	ID       string          `json:"id"`              // 在线歌曲在插件里的 id
	Artwork  string          `json:"artwork"`         // 在线歌曲的封面地址
	Duration float64         `json:"duration"`        // 时长（秒），不知道时为 0
	Extra    json.RawMessage `json:"extra,omitempty"` // 插件返回的原始数据，播放时原样还给插件
}

// IsOnline 判断是不是插件提供的在线歌曲。旧数据里没有 source 字段，当作本地歌曲。
func (s Song) IsOnline() bool {
	return s.Source != "" && s.Source != LocalSource
}

// Key 是一首歌的唯一标识：本地歌曲是文件路径，在线歌曲是“平台:id”
func (s Song) Key() string {
	if s.IsOnline() {
		return s.Source + ":" + s.ID
	}
	return s.Path
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
		Source: LocalSource,
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
