package core

import (
	"musicplayer/internal/music"
	"musicplayer/internal/queue"
)

// ---- 歌曲库 ----

// ScanMusic 扫描文件夹（包括子文件夹）里所有支持的音频文件，结果作为歌曲库（桌面版用）
func (c *Core) ScanMusic(dir string) ([]music.Song, error) {
	songs, err := music.Scan(dir)
	if err != nil {
		return nil, err
	}
	c.setLibrary(songs)
	return songs, nil
}

// SetLibrary 直接设置歌曲库（Android 版用：歌曲由系统的媒体库扫描出来，再交给这里）。
// 不支持的格式会被去掉，没写来源的当作本地歌曲。返回实际保存的歌曲库
func (c *Core) SetLibrary(songs []music.Song) []music.Song {
	kept := make([]music.Song, 0, len(songs))
	for _, song := range songs {
		if !music.IsAudio(song.Path) {
			continue
		}
		if song.Source == "" {
			song.Source = music.LocalSource
		}
		if song.Name == "" {
			song.Name = song.Title
		}
		kept = append(kept, song)
	}
	c.setLibrary(kept)
	return kept
}

func (c *Core) setLibrary(songs []music.Song) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.library = songs
}

// ---- 播放队列：每个方法都返回最新的队列快照，前端拿到后刷新界面 ----

// PlayLibrary 用整个歌曲库替换播放队列，从第 index 首开始
func (c *Core) PlayLibrary(index int) queue.State {
	c.mu.Lock()
	library := c.library
	c.mu.Unlock()
	return c.queue.Replace(library, index)
}

// PlaySongs 用一组歌（比如某个歌单）替换播放队列，从第 index 首开始
func (c *Core) PlaySongs(songs []music.Song, index int) queue.State {
	return c.queue.Replace(songs, index)
}

// QueueState 返回当前播放队列
func (c *Core) QueueState() queue.State {
	return c.queue.State()
}

// QueuePlayAt 切到队列里的第 index 首
func (c *Core) QueuePlayAt(index int) queue.State {
	return c.queue.PlayAt(index)
}

// QueueNext 下一首；auto 为 true 表示歌曲自然播完
func (c *Core) QueueNext(auto bool) queue.State {
	return c.queue.Next(auto)
}

// QueuePrevious 上一首
func (c *Core) QueuePrevious() queue.State {
	return c.queue.Previous()
}

// QueueSetMode 切换播放模式：sequence / repeat-one / shuffle
func (c *Core) QueueSetMode(mode string) queue.State {
	return c.queue.SetMode(queue.Mode(mode))
}

// QueueAddNext 把一首歌设为下一首播放
func (c *Core) QueueAddNext(song music.Song) queue.State {
	return c.queue.AddNext(song)
}

// QueueRemove 从队列里移除第 index 首
func (c *Core) QueueRemove(index int) queue.State {
	return c.queue.Remove(index)
}
