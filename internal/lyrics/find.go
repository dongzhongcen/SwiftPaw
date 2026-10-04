package lyrics

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

// maxLrcSize 是歌词文件的大小上限，正常的 LRC 只有几 KB
const maxLrcSize = 1 << 20

// Find 找一首本地歌曲的歌词：
//  1. 先找同一个文件夹里同名的 .lrc 文件（扩展名不区分大小写）
//  2. 再找音频文件里内嵌的歌词（mp3 的 USLT、flac/ogg 的 LYRICS、m4a 的 ©lyr）
//
// 都没有时返回空的 Lyrics（Source 为空），不算错误。
func Find(audioPath string) (Lyrics, error) {
	if lrcPath := findLrcFile(audioPath); lrcPath != "" {
		info, err := os.Stat(lrcPath)
		if err == nil && info.Size() <= maxLrcSize {
			data, err := os.ReadFile(lrcPath)
			if err != nil {
				return Lyrics{}, err
			}
			if l := Parse(Decode(data)); len(l.Lines) > 0 {
				l.Source = "lrc"
				return l, nil
			}
		}
	}

	if text := embeddedLyrics(audioPath); text != "" {
		if l := Parse(text); len(l.Lines) > 0 {
			l.Source = "embedded"
			return l, nil
		}
	}
	return Lyrics{Lines: []Line{}}, nil
}

// findLrcFile 返回同名 .lrc 文件的路径，找不到返回空字符串
func findLrcFile(audioPath string) string {
	dir := filepath.Dir(audioPath)
	base := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))

	// 最常见的情况：直接拼出来就存在
	direct := filepath.Join(dir, base+".lrc")
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	// Linux 上文件名区分大小写，再扫一遍文件夹找 .LRC / .Lrc
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), base+".lrc") {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

func embeddedLyrics(audioPath string) string {
	file, err := os.Open(audioPath)
	if err != nil {
		return ""
	}
	defer file.Close()
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(metadata.Lyrics())
}
