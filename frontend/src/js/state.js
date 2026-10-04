// 全局状态和一个很小的事件中心。
// 各个模块（播放器、侧边栏、列表）互相不直接调用渲染函数，而是改完状态后 emit 一个事件，
// 关心这个事件的模块自己去刷新。这样模块之间耦合少，比较好读。

export const state = {
  library: [], // 本地音乐：当前文件夹扫描到的歌
  folder: '', // 当前音乐文件夹
  queue: { songs: [], current: -1, mode: 'sequence', upcoming: [] }, // Go 返回的队列快照
  playlists: [], // 所有歌单，第一个是“我喜欢”
  favorites: new Set(), // 收藏歌曲的 key
  view: 'library', // 当前显示的页面：library / recent / queue / lyrics / settings / playlist:<id>
  previousView: 'library', // 上一个页面（从歌词页返回时用）
};

const listeners = {};

// on 订阅事件
export function on(name, fn) {
  (listeners[name] ||= []).push(fn);
}

// emit 发出事件，所有订阅者都会被调用
export function emit(name, data) {
  (listeners[name] || []).forEach((fn) => fn(data));
}

// currentSong 返回队列里正在播放的歌，没有就返回 null
export function currentSong() {
  const q = state.queue;
  return q.current >= 0 ? q.songs[q.current] : null;
}
