package queue

import (
	"fmt"
	"slices"
	"testing"

	"musicplayer/internal/music"
)

// makeSongs 生成 n 首假歌，路径是 song0.mp3、song1.mp3……
func makeSongs(n int) []music.Song {
	songs := make([]music.Song, n)
	for i := range songs {
		songs[i] = music.Song{Path: fmt.Sprintf("song%d.mp3", i), Title: fmt.Sprintf("歌%d", i)}
	}
	return songs
}

func currentPath(s State) string {
	if s.Current < 0 {
		return ""
	}
	return s.Songs[s.Current].Path
}

func TestSequenceNextAndPreviousWrapAround(t *testing.T) {
	q := NewWithSeed(1)
	s := q.Replace(makeSongs(3), 1)
	if currentPath(s) != "song1.mp3" {
		t.Fatalf("开始应该是 song1，实际 %q", currentPath(s))
	}
	if !slices.Equal(s.Upcoming, []int{2}) {
		t.Errorf("接下来应该只有第 2 首，实际 %v", s.Upcoming)
	}
	s = q.Next(true)
	s = q.Next(true)
	if currentPath(s) != "song0.mp3" {
		t.Errorf("最后一首之后应该回到 song0，实际 %q", currentPath(s))
	}
	s = q.Previous()
	if currentPath(s) != "song2.mp3" {
		t.Errorf("第一首的上一首应该是最后一首，实际 %q", currentPath(s))
	}
}

func TestRepeatOne(t *testing.T) {
	q := NewWithSeed(1)
	q.Replace(makeSongs(3), 0)
	q.SetMode(RepeatOne)
	if s := q.Next(true); currentPath(s) != "song0.mp3" {
		t.Errorf("单曲循环自然播完应该还是 song0，实际 %q", currentPath(s))
	}
	if s := q.Next(false); currentPath(s) != "song1.mp3" {
		t.Errorf("单曲循环手动下一首应该切到 song1，实际 %q", currentPath(s))
	}
}

func TestShuffleEverySongOncePerRound(t *testing.T) {
	q := NewWithSeed(42)
	q.SetMode(Shuffle)
	s := q.Replace(makeSongs(10), 3)
	if currentPath(s) != "song3.mp3" {
		t.Fatalf("随机模式也应该先放点的那首，实际 %q", currentPath(s))
	}
	for round := 0; round < 3; round++ {
		seen := map[string]bool{currentPath(s): true}
		for i := 0; i < 9; i++ {
			s = q.Next(true)
			seen[currentPath(s)] = true
		}
		if len(seen) != 10 {
			t.Fatalf("第 %d 轮应该 10 首各放一次，实际只放到 %d 首", round+1, len(seen))
		}
		s = q.Next(true) // 进入新一轮
	}
}

func TestShuffleNewRoundDoesNotRepeatLastSong(t *testing.T) {
	for seed := uint64(0); seed < 50; seed++ {
		q := NewWithSeed(seed)
		q.SetMode(Shuffle)
		s := q.Replace(makeSongs(4), 0)
		for i := 0; i < 3; i++ {
			s = q.Next(true)
		}
		last := currentPath(s)
		if s = q.Next(true); currentPath(s) == last {
			t.Fatalf("seed %d：新一轮第一首和上一轮最后一首重复了", seed)
		}
	}
}

func TestAddNextInsertsAfterCurrent(t *testing.T) {
	for _, mode := range []Mode{Sequence, Shuffle} {
		q := NewWithSeed(7)
		q.SetMode(mode)
		q.Replace(makeSongs(5), 2)
		extra := music.Song{Path: "new.mp3"}
		s := q.AddNext(extra)
		if len(s.Songs) != 6 || s.Songs[s.Upcoming[0]].Path != "new.mp3" {
			t.Fatalf("%s：new.mp3 应该是下一首，实际 upcoming=%v", mode, s.Upcoming)
		}
		if currentPath(s) != "song2.mp3" {
			t.Errorf("%s：插队不应该改变正在播放的歌，实际 %q", mode, currentPath(s))
		}
		if s = q.Next(false); currentPath(s) != "new.mp3" {
			t.Errorf("%s：下一首应该是 new.mp3，实际 %q", mode, currentPath(s))
		}
	}
}

func TestAddNextMovesExistingSong(t *testing.T) {
	q := NewWithSeed(1)
	songs := makeSongs(5)
	q.Replace(songs, 0)
	s := q.AddNext(songs[4])
	if len(s.Songs) != 5 {
		t.Fatalf("已经在队列里的歌不应该重复加入，实际 %d 首", len(s.Songs))
	}
	if s = q.Next(false); currentPath(s) != "song4.mp3" {
		t.Errorf("下一首应该是挪过来的 song4，实际 %q", currentPath(s))
	}
	// 再加正在播放的这首，不应该有变化
	if s2 := q.AddNext(songs[4]); len(s2.Songs) != 5 || currentPath(s2) != "song4.mp3" {
		t.Errorf("加入正在播放的歌不应该改变队列")
	}
}

func TestAddNextToEmptyQueue(t *testing.T) {
	q := NewWithSeed(1)
	s := q.AddNext(music.Song{Path: "a.mp3"})
	if s.Current != -1 || len(s.Upcoming) != 1 {
		t.Fatalf("空队列插入后应该还没开始放、下一首是它，实际 %+v", s)
	}
	if s = q.Next(false); currentPath(s) != "a.mp3" {
		t.Errorf("下一首应该是 a.mp3，实际 %q", currentPath(s))
	}
}

func TestRemove(t *testing.T) {
	q := NewWithSeed(1)
	q.Replace(makeSongs(5), 2)

	s := q.Remove(0) // 删已经放过的
	if currentPath(s) != "song2.mp3" || len(s.Songs) != 4 {
		t.Errorf("删前面的歌不应该影响当前歌曲，实际 %q", currentPath(s))
	}
	s = q.Remove(s.Current) // 删正在放的
	if currentPath(s) != "song3.mp3" {
		t.Errorf("删掉正在放的歌后应该接着放后面那首，实际 %q", currentPath(s))
	}
	s = q.Remove(s.Upcoming[0]) // 删下一首 song4
	if len(s.Upcoming) != 0 || currentPath(s) != "song3.mp3" {
		t.Errorf("删后面的歌后队列不对：%+v", s)
	}
	s = q.Remove(s.Current) // 删最后一首而且正在放，回到开头
	if currentPath(s) != "song1.mp3" {
		t.Errorf("删掉最后一首后应该回到开头，实际 %q", currentPath(s))
	}
	q.Remove(0)
	if s = q.State(); s.Current != -1 || len(s.Songs) != 0 {
		t.Errorf("全删完应该是空队列，实际 %+v", s)
	}
}

func TestSetModeKeepsCurrentSong(t *testing.T) {
	q := NewWithSeed(3)
	q.Replace(makeSongs(6), 4)
	for _, mode := range []Mode{Shuffle, RepeatOne, Sequence} {
		if s := q.SetMode(mode); currentPath(s) != "song4.mp3" || s.Mode != mode {
			t.Errorf("切到 %s 后当前歌曲应该还是 song4，实际 %q", mode, currentPath(s))
		}
	}
	if s := q.SetMode("bad"); s.Mode != Sequence {
		t.Errorf("无效模式应该被忽略")
	}
}

func TestPlayAtAndEmptyQueue(t *testing.T) {
	q := NewWithSeed(1)
	if s := q.Next(false); s.Current != -1 {
		t.Error("空队列下一首应该什么都不做")
	}
	if s := q.Previous(); s.Current != -1 {
		t.Error("空队列上一首应该什么都不做")
	}
	q.Replace(makeSongs(3), -1)
	if s := q.PlayAt(9); s.Current != -1 {
		t.Error("越界下标应该被忽略")
	}
	if s := q.PlayAt(2); currentPath(s) != "song2.mp3" {
		t.Errorf("PlayAt(2) 应该放 song2，实际 %q", currentPath(s))
	}
}

func TestAddNextOnlineSong(t *testing.T) {
	q := NewWithSeed(1)
	a := music.Song{Source: "p", ID: "1", Title: "在线1"}
	b := music.Song{Source: "p", ID: "2", Title: "在线2"}
	c := music.Song{Source: "p", ID: "3", Title: "在线3"}
	q.Replace([]music.Song{a, b, c}, 0)
	// 在线歌曲没有 Path，要按 Key 判断是不是同一首，不能把所有在线歌曲当成一首
	st := q.AddNext(c)
	if len(st.Songs) != 3 || st.Songs[st.Upcoming[0]].ID != "3" {
		t.Fatalf("AddNext 在线歌曲不对：%+v", st)
	}
	st = q.AddNext(music.Song{Source: "p", ID: "4", Title: "新的"})
	if len(st.Songs) != 4 || st.Songs[st.Upcoming[0]].ID != "4" {
		t.Fatalf("AddNext 新的在线歌曲不对：%+v", st)
	}
}
