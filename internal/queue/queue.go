// Package queue 是播放队列：记住要放哪些歌、现在放到哪一首、下一首放谁。
//
// 思路：songs 按加入顺序存歌，order 存“播放顺序”（songs 的下标），pos 指向 order 里正在播放的位置。
// 顺序播放时 order 就是 0,1,2,...；随机播放时 order 是打乱后的顺序，每首歌放完一轮才会重复。
package queue

import (
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"musicplayer/internal/music"
)

// Mode 是播放模式
type Mode string

const (
	Sequence  Mode = "sequence"   // 顺序播放，放完最后一首回到第一首
	RepeatOne Mode = "repeat-one" // 单曲循环，自然播完会重播；手动点下一首仍然切歌
	Shuffle   Mode = "shuffle"    // 随机播放，一轮里每首歌只放一次
)

// upcomingLimit 是“接下来播放”最多显示几首
const upcomingLimit = 50

// State 是返回给前端的队列快照
type State struct {
	Songs    []music.Song `json:"songs"`    // 队列里的歌
	Current  int          `json:"current"`  // 正在播放的是 Songs 里的第几首，-1 表示没有
	Mode     Mode         `json:"mode"`     // 播放模式
	Upcoming []int        `json:"upcoming"` // 接下来要播放的歌（Songs 的下标），按播放顺序
}

// Queue 是播放队列。前端可能同时调用好几个方法，所以用互斥锁保护。
type Queue struct {
	mu    sync.Mutex
	songs []music.Song
	order []int
	pos   int
	mode  Mode
	rng   *rand.Rand
}

// New 创建一个空队列
func New() *Queue {
	return NewWithSeed(uint64(time.Now().UnixNano()))
}

// NewWithSeed 用固定的随机种子创建队列，测试里用它让随机结果可以重现
func NewWithSeed(seed uint64) *Queue {
	return &Queue{pos: -1, mode: Sequence, rng: rand.New(rand.NewPCG(seed, 2026))}
}

// State 返回当前队列快照
func (q *Queue) State() State {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.state()
}

// Replace 用一组歌替换整个队列，并把第 start 首设为当前歌曲（start 为 -1 表示先不选）
func (q *Queue) Replace(songs []music.Song, start int) State {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.songs = slices.Clone(songs)
	q.rebuild(start)
	return q.state()
}

// PlayAt 切到队列里的第 index 首
func (q *Queue) PlayAt(index int) State {
	q.mu.Lock()
	defer q.mu.Unlock()
	if index < 0 || index >= len(q.songs) {
		return q.state()
	}
	if q.mode == Shuffle {
		q.rebuild(index) // 随机模式下从这首开始重新洗牌，其余的歌这一轮都还能放到
	} else {
		q.pos = slices.Index(q.order, index)
	}
	return q.state()
}

// Next 切到下一首。auto 为 true 表示是歌曲自然播完触发的。
func (q *Queue) Next(auto bool) State {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := len(q.order)
	if n == 0 {
		return q.state()
	}
	if auto && q.mode == RepeatOne && q.pos >= 0 {
		return q.state() // 单曲循环：还是这首
	}

	last := q.current()
	q.pos++
	if q.pos >= n {
		if q.mode == Shuffle {
			// 一轮放完了，重新洗牌；避免新一轮第一首刚好是刚放完的那首
			q.rebuild(-1)
			if n > 1 && q.order[0] == last {
				q.order[0], q.order[n-1] = q.order[n-1], q.order[0]
			}
		}
		q.pos = 0
	}
	return q.state()
}

// Previous 切到上一首，已经是第一首时跳到最后一首
func (q *Queue) Previous() State {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := len(q.order)
	if n == 0 {
		return q.state()
	}
	if q.pos <= 0 {
		q.pos = n - 1
	} else {
		q.pos--
	}
	return q.state()
}

// SetMode 切换播放模式，正在播放的歌保持不变
func (q *Queue) SetMode(mode Mode) State {
	q.mu.Lock()
	defer q.mu.Unlock()
	if mode != Sequence && mode != RepeatOne && mode != Shuffle {
		return q.state()
	}
	cur := q.current()
	q.mode = mode
	q.rebuild(cur)
	return q.state()
}

// AddNext 把一首歌插到“下一首播放”。如果它已经在队列里，就把它挪过来。
func (q *Queue) AddNext(song music.Song) State {
	q.mu.Lock()
	defer q.mu.Unlock()

	if i := slices.IndexFunc(q.songs, func(s music.Song) bool { return s.Key() == song.Key() }); i >= 0 {
		if i == q.current() {
			return q.state() // 正在放的就是它，不用动
		}
		q.removeAt(i)
	}

	insertAt := q.current() + 1 // 在 songs 里紧跟在当前这首后面；没有当前歌曲时就是 0
	q.songs = slices.Insert(q.songs, insertAt, song)
	for k, idx := range q.order {
		if idx >= insertAt {
			q.order[k]++ // 插入后，后面的歌下标都要加一
		}
	}
	q.order = slices.Insert(q.order, q.pos+1, insertAt)
	return q.state()
}

// Remove 从队列里删掉第 index 首。删的是正在播放的歌时，当前歌曲变成它后面那首。
func (q *Queue) Remove(index int) State {
	q.mu.Lock()
	defer q.mu.Unlock()
	if index >= 0 && index < len(q.songs) {
		q.removeAt(index)
	}
	return q.state()
}

// ---- 下面是内部方法，调用前必须已经拿到锁 ----

func (q *Queue) current() int {
	if q.pos < 0 || q.pos >= len(q.order) {
		return -1
	}
	return q.order[q.pos]
}

// rebuild 按当前模式重新生成播放顺序，并让 first 成为当前歌曲（-1 表示没有当前歌曲）
func (q *Queue) rebuild(first int) {
	n := len(q.songs)
	q.order = make([]int, n)
	for i := range q.order {
		q.order[i] = i
	}
	if first < 0 || first >= n {
		first = -1
	}
	if q.mode != Shuffle {
		q.pos = first
		return
	}

	q.rng.Shuffle(n, func(i, j int) { q.order[i], q.order[j] = q.order[j], q.order[i] })
	if first < 0 {
		q.pos = -1
		return
	}
	// 把要播放的这首换到最前面
	i := slices.Index(q.order, first)
	q.order[0], q.order[i] = q.order[i], q.order[0]
	q.pos = 0
}

func (q *Queue) removeAt(i int) {
	p := slices.Index(q.order, i)
	q.songs = slices.Delete(q.songs, i, i+1)
	q.order = slices.Delete(q.order, p, p+1)
	for k, idx := range q.order {
		if idx > i {
			q.order[k]-- // 删除后，后面的歌下标都要减一
		}
	}

	switch {
	case len(q.order) == 0:
		q.pos = -1
	case p < q.pos:
		q.pos-- // 删的是已经放过的歌，当前位置往前挪一格
	case p == q.pos && q.pos >= len(q.order):
		q.pos = 0 // 删的是最后一首而且正在放，回到开头
	}
}

func (q *Queue) state() State {
	s := State{
		Songs:    slices.Clone(q.songs),
		Current:  q.current(),
		Mode:     q.mode,
		Upcoming: []int{},
	}
	if s.Songs == nil {
		s.Songs = []music.Song{}
	}
	for p := q.pos + 1; p < len(q.order) && len(s.Upcoming) < upcomingLimit; p++ {
		s.Upcoming = append(s.Upcoming, q.order[p])
	}
	return s
}
