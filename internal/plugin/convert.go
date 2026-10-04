package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"musicplayer/internal/music"
)

// SongFromItem 把插件返回的 musicItem 变成 music.Song。
// 原始数据整个保存在 Song.Extra 里，以后调用 getMediaSource 时原样传回插件（插件可能放了自己需要的字段）。
func SongFromItem(platform string, raw json.RawMessage) (music.Song, error) {
	item, err := decodeObject(raw)
	if err != nil {
		return music.Song{}, err
	}
	id := scalarString(item["id"])
	if id == "" {
		return music.Song{}, fmt.Errorf("歌曲缺少 id")
	}
	title := scalarString(item["title"])
	if title == "" {
		title = "未知歌曲"
	}
	artist := scalarString(item["artist"])
	if artist == "" {
		artist = "未知歌手"
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return music.Song{}, err
	}
	return music.Song{
		Name:     title,
		Title:    title,
		Artist:   artist,
		Album:    scalarString(item["album"]),
		Source:   platform,
		ID:       id,
		Artwork:  httpURL(scalarString(item["artwork"])),
		Duration: parseDuration(item["duration"]),
		Extra:    compact.Bytes(),
	}, nil
}

// ItemFromSong 把 Song 还原成插件认识的 musicItem（JSON）
func ItemFromSong(song music.Song) (json.RawMessage, error) {
	item := map[string]any{}
	if len(song.Extra) > 0 {
		decoded, err := decodeObject(song.Extra)
		if err != nil {
			return nil, err
		}
		item = decoded
	}
	// 插件原始数据里没有的字段，用 Song 里的补上
	setDefault := func(key string, value any) {
		if _, ok := item[key]; !ok {
			item[key] = value
		}
	}
	setDefault("id", song.ID)
	setDefault("title", song.Title)
	setDefault("artist", song.Artist)
	setDefault("album", song.Album)
	setDefault("artwork", song.Artwork)
	setDefault("duration", song.Duration)
	item["platform"] = song.Source
	return json.Marshal(item)
}

// decodeObject 解析 JSON 对象；数字保留成 json.Number，避免很长的 id 丢精度
func decodeObject(raw json.RawMessage) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var item map[string]any
	if err := dec.Decode(&item); err != nil {
		return nil, fmt.Errorf("歌曲数据格式不对：%v", err)
	}
	if item == nil {
		return nil, fmt.Errorf("歌曲数据是空的")
	}
	return item, nil
}

// scalarString 把字符串、数字、布尔值变成字符串；对象和数组返回空字符串
func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case json.Number:
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	}
	return ""
}

// parseDuration 支持数字（秒）和 "3:45" 这样的字符串
func parseDuration(v any) float64 {
	switch x := v.(type) {
	case json.Number:
		f, _ := x.Float64()
		return max(0, f)
	case string:
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return max(0, f)
		}
		total := 0.0
		for _, part := range strings.Split(x, ":") {
			n, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil {
				return 0
			}
			total = total*60 + n
		}
		return total
	}
	return 0
}

// httpURL 只保留 http/https 地址，别的（比如 javascript:）一律丢掉
func httpURL(s string) string {
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		return s
	}
	if strings.HasPrefix(s, "//") {
		return "https:" + s
	}
	return ""
}
