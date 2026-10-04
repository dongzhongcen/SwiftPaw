// Package lyrics 负责歌词：解析 LRC 格式、找到一首歌对应的歌词。
//
// LRC 是最常见的歌词格式，每一行前面用方括号写时间，比如：
//
//	[ti:海边小路]
//	[00:12.30][01:40.00]第一句测试歌词
//
// 一行可以有多个时间（副歌重复时常见），[ti:] [ar:] 这类是标题、歌手等信息，不算歌词。
package lyrics

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Line 是一行歌词
type Line struct {
	Time float64 `json:"time"` // 从第几秒开始显示；没有时间轴的歌词都是 0
	Text string  `json:"text"`
}

// Lyrics 是一首歌的歌词
type Lyrics struct {
	Lines  []Line `json:"lines"`  // 按时间排好序
	Plain  bool   `json:"plain"`  // true 表示没有时间轴，只是纯文本
	Source string `json:"source"` // 歌词从哪来：lrc（同名文件）/ embedded（内嵌）/ plugin（插件）；空字符串表示没找到
}

var (
	// timeTag 匹配 [mm:ss]、[mm:ss.xx]、[mm:ss.xxx]，也兼容 [mm:ss:xx]
	timeTag = regexp.MustCompile(`^(\d{1,3}):(\d{1,2})(?:[.:](\d{1,3}))?$`)
	// wordTag 是“逐字歌词”里的 <mm:ss.xx>，显示时去掉
	wordTag = regexp.MustCompile(`<\d{1,3}:\d{1,2}(?:[.:]\d{1,3})?>`)
)

// Parse 解析 LRC 文本。没有任何时间标签时，当成纯文本歌词（Plain = true）。
func Parse(text string) Lyrics {
	text = strings.TrimPrefix(text, "\uFEFF") // 去掉 UTF-8 BOM
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var timed []Line
	var plain []string
	offset := 0.0 // [offset:+500] 表示整体提前 0.5 秒

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		times, rest, isMeta := splitTags(line, &offset)
		rest = strings.TrimSpace(wordTag.ReplaceAllString(rest, ""))
		switch {
		case len(times) > 0:
			for _, t := range times {
				timed = append(timed, Line{Time: t, Text: rest})
			}
		case isMeta:
			// [ti:xxx] 这样的信息行，跳过
		case rest != "":
			plain = append(plain, rest)
		}
	}

	if len(timed) == 0 {
		lines := make([]Line, 0, len(plain))
		for _, t := range plain {
			lines = append(lines, Line{Text: t})
		}
		return Lyrics{Lines: lines, Plain: len(lines) > 0}
	}

	for i := range timed {
		timed[i].Time = max(0, round3(timed[i].Time-offset))
	}
	// SliceStable：时间相同的行保持原来的先后顺序（比如原文和翻译）
	sort.SliceStable(timed, func(i, j int) bool { return timed[i].Time < timed[j].Time })
	return Lyrics{Lines: timed}
}

// splitTags 把一行开头的所有 [..] 标签拆出来。
// 返回：时间列表（秒）、标签后面的文字、这一行是不是信息标签（ti/ar/offset 等）。
func splitTags(line string, offset *float64) (times []float64, rest string, isMeta bool) {
	rest = line
	for strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]")
		if end < 0 {
			break
		}
		tag := strings.TrimSpace(rest[1:end])
		if t, ok := parseTime(tag); ok {
			times = append(times, t)
		} else if key, value, found := strings.Cut(tag, ":"); found && isMetaKey(key) {
			isMeta = true
			if strings.EqualFold(strings.TrimSpace(key), "offset") {
				if ms, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
					*offset = float64(ms) / 1000
				}
			}
		} else {
			break // 不认识的方括号，当成歌词正文的一部分
		}
		rest = rest[end+1:]
	}
	return times, rest, isMeta
}

// isMetaKey 判断是不是 [ti:] [ar:] 这种信息标签：冒号前面全是英文字母
func isMetaKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

// parseTime 把 "01:02.50" 变成 62.5 秒
func parseTime(tag string) (float64, bool) {
	m := timeTag.FindStringSubmatch(tag)
	if m == nil {
		return 0, false
	}
	minutes, _ := strconv.Atoi(m[1])
	seconds, _ := strconv.Atoi(m[2])
	t := float64(minutes*60 + seconds)
	if frac := m[3]; frac != "" {
		// .5 是 0.5 秒，.50 是 0.50 秒，.500 也是 0.5 秒：按位数换算
		n, _ := strconv.Atoi(frac)
		divisor := 1.0
		for range len(frac) {
			divisor *= 10
		}
		t += float64(n) / divisor
	}
	return round3(t), true
}

// round3 保留三位小数，避免 0.1+0.2 这类浮点误差出现在 JSON 里
func round3(f float64) float64 {
	return float64(int64(f*1000+0.5)) / 1000
}
