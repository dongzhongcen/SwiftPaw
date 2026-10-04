// 中间内容区的各个页面。每个页面用一个 load 函数描述：标题、副标题、右上角按钮、歌曲列表、行操作。
// renderView 拿到描述后统一画出来，所以新增页面只需要再写一个 load 函数。

import {
  ClearHistory,
  PlaySongs,
  PlaylistSongs,
  QueuePlayAt,
  QueueRemove,
  RecentPlays,
  RemoveFromPlaylist,
} from '../../wailsjs/go/main/App';
import { icons } from './icons.js';
import { FAVORITES_ID, deletePlaylist, refreshFavorites, refreshPlaylists, renamePlaylist } from './actions.js';
import { confirmDialog } from './dialog.js';
import { pickFolder, rescan } from './library.js';
import { applyQueue, audio, playSongs } from './player.js';
import { renderSongList, markActive, updateHearts } from './songlist.js';
import { state, on, emit } from './state.js';
import { $, escapeHtml, songKey, toast, guard } from './util.js';

// 把一组歌变成列表需要的 items（index 是它在原数组里的位置）
const toItems = (songs) => songs.map((song, index) => ({ song, index }));

const playAllAction = (songs) => ({ label: '播放全部', icon: icons.play, primary: true, run: () => playSongs(songs, 0) });

// 每个页面的定义。param 是 view 名里冒号后面的部分，比如 playlist:3 里的 3。
// load 函数返回页面描述；返回值里有 render(body) 时表示这是自定义页面（歌词、设置等），不画歌曲列表。
const views = {
  async library() {
    const songs = state.library;
    return {
      title: '本地音乐',
      subtitle: state.folder ? `${songs.length} 首歌 · ${state.folder}` : '还没有选择音乐文件夹',
      actions: [
        playAllAction(songs),
        { label: state.folder ? '更换文件夹' : '选择文件夹', icon: icons.folder, run: pickFolder },
        state.folder && { label: '重新扫描', icon: icons.refresh, run: rescan },
      ],
      groups: [{ items: toItems(songs) }],
      rowActions: ['favorite', 'next', 'add'],
      onPlay: (item) => playSongs(songs, item.index),
      empty: state.folder
        ? '<p>这个文件夹里没有找到支持的音频文件</p>'
        : '<p>还没有选择音乐文件夹</p><button class="btn primary" data-empty-action>选择音乐文件夹</button>',
      emptyAction: pickFolder,
    };
  },

  async playlist(param) {
    const playlist = state.playlists.find((p) => p.id === Number(param));
    if (!playlist) return null; // 歌单被删了，回到本地音乐
    const songs = (await PlaylistSongs(playlist.id)) || [];
    const isFavorites = playlist.id === FAVORITES_ID;
    return {
      title: playlist.name,
      subtitle: isFavorites ? `${songs.length} 首收藏的歌` : `歌单 · ${songs.length} 首歌`,
      actions: [
        playAllAction(songs),
        !playlist.builtin && { label: '重命名', icon: icons.edit, run: () => renamePlaylist(playlist) },
        !playlist.builtin && {
          label: '删除',
          icon: icons.trash,
          danger: true,
          run: async () => {
            if (await deletePlaylist(playlist)) showView('library');
          },
        },
      ],
      groups: [{ items: toItems(songs) }],
      rowActions: ['favorite', 'next', 'add', 'remove'],
      onPlay: (item) => playSongs(songs, item.index),
      onRemove: async (item) => {
        await RemoveFromPlaylist(playlist.id, songKey(item.song));
        if (isFavorites) await refreshFavorites();
        await refreshPlaylists();
        toast('已从歌单移除');
      },
      empty: isFavorites
        ? '<p>还没有收藏的歌</p><p class="muted">点歌曲后面的红心就能收藏</p>'
        : '<p>歌单里还没有歌</p><p class="muted">在本地音乐里点歌曲后面的“加入歌单”</p>',
    };
  },

  async recent() {
    const songs = (await RecentPlays()) || [];
    return {
      title: '最近播放',
      subtitle: `${songs.length} 首歌`,
      actions: [
        playAllAction(songs),
        songs.length > 0 && {
          label: '清空',
          icon: icons.trash,
          run: async () => {
            const ok = await confirmDialog({ title: '清空最近播放', message: '确定要清空最近播放记录吗？', okText: '清空', danger: true });
            if (!ok) return;
            await ClearHistory();
            emit('recent-changed');
            toast('已清空最近播放');
          },
        },
      ],
      groups: [{ items: toItems(songs) }],
      rowActions: ['favorite', 'next', 'add'],
      onPlay: (item) => playSongs(songs, item.index),
      empty: '<p>还没有播放记录</p>',
    };
  },

  async queue() {
    const q = state.queue;
    const groups = [];
    if (q.current >= 0) groups.push({ label: '正在播放', items: [{ song: q.songs[q.current], index: q.current }] });
    if (q.upcoming.length) groups.push({ label: '接下来', items: q.upcoming.map((index) => ({ song: q.songs[index], index })) });
    const shown = groups.reduce((n, g) => n + g.items.length, 0);
    const hidden = q.songs.length - shown;
    return {
      title: '播放队列',
      subtitle: `${q.songs.length} 首歌` + (hidden > 0 ? ` · 还有 ${hidden} 首已经放过或排得比较远` : ''),
      actions: [
        playAllAction(q.songs),
        q.songs.length > 0 && {
          label: '清空队列',
          icon: icons.trash,
          run: async () => {
            await applyQueue(await PlaySongs([], -1), false);
            toast('已清空播放队列');
          },
        },
      ],
      groups,
      rowActions: ['favorite', 'add', 'remove'],
      onPlay: async (item) => applyQueue(await QueuePlayAt(item.index), true),
      onRemove: async (item) => applyQueue(await QueueRemove(item.index), !audio().paused),
      empty: '<p>播放队列是空的</p><p class="muted">在任意列表里点一首歌开始播放</p>',
    };
  },
};

// registerView 让其他模块注册自己的页面，比如 registerView('lyrics', loadLyricsView)
export function registerView(kind, load) {
  views[kind] = load;
}

let renderToken = 0;
let songsByKey = new Map(); // 当前页面里的歌，更新红心时用
let lastData = null; // 当前页面的描述，搜索时直接用它重新过滤，不用再去 Go 那边取

// showView 切换页面
export async function showView(name) {
  const changed = state.view !== name;
  if (changed) {
    state.previousView = state.view;
    $('search').value = ''; // 换页面时清空搜索
  }
  state.view = name;
  emit('view-changed', name);
  await renderView(!changed);
}

// renderView 重新画当前页面；keepScroll 为 true 时保持滚动位置（比如删掉一行之后）
export async function renderView(keepScroll = true) {
  const token = ++renderToken;
  const [kind, param] = state.view.split(':');
  let data;
  try {
    data = await views[kind]?.(param);
  } catch (err) {
    data = { title: '出错了', subtitle: String(err), actions: [], groups: [], empty: `<p>${escapeHtml(String(err))}</p>` };
  }
  if (token !== renderToken) return; // 期间又切换了页面，丢掉这次结果
  if (!data) {
    showView('library');
    return;
  }

  lastData = data;
  $('view-title').textContent = data.title;
  $('view-subtitle').textContent = data.subtitle || '';
  renderActions((data.actions || []).filter(Boolean));

  const body = $('view-body');
  const scrollTop = body.scrollTop;
  const searchable = !data.render && data.searchable !== false;
  $('search-box').hidden = !searchable;
  document.querySelector('.view-header').classList.toggle('searchable', searchable);
  body.classList.toggle('custom', !!data.render);

  if (data.render) {
    await data.render(body);
    return;
  }
  renderList();
  body.scrollTop = keepScroll ? scrollTop : 0;
  if (!keepScroll) markActive(body, true);
}

// renderList 按搜索框里的文字过滤后画出歌曲列表
function renderList() {
  const data = lastData;
  const body = $('view-body');
  const query = $('search').value.trim().toLowerCase();
  const groups = query ? filterGroups(data.groups, query) : data.groups;

  songsByKey = new Map();
  groups.forEach((g) => g.items.forEach((item) => songsByKey.set(songKey(item.song), item.song)));
  const empty = query ? `<p>没有找到和“${escapeHtml(query)}”有关的歌</p>` : data.empty;
  renderSongList(body, { ...data, groups, empty });
  body.querySelector('[data-empty-action]')?.addEventListener('click', guard(data.emptyAction));
}

// filterGroups 按标题、歌手、专辑、文件名过滤（不区分大小写），空的分组去掉
export function filterGroups(groups, query) {
  return groups
    .map((g) => ({ ...g, items: g.items.filter((item) => matchSong(item.song, query)) }))
    .filter((g) => g.items.length > 0);
}

function matchSong(song, query) {
  const fileName = (song.path || '').split(/[\\/]/).pop();
  return [song.title, song.artist, song.album, song.name, fileName]
    .some((field) => (field || '').toLowerCase().includes(query));
}

function renderActions(actions) {
  const box = $('view-actions');
  box.innerHTML = '';
  actions.forEach((action) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'btn' + (action.primary ? ' primary' : '') + (action.danger ? ' danger' : '');
    button.innerHTML = `${action.icon || ''}<span>${escapeHtml(action.label)}</span>`;
    button.setAttribute('aria-label', action.label); // 手机上按钮可能只显示图标
    button.addEventListener('click', guard(action.run));
    box.appendChild(button);
  });
}

// initViews 订阅各种数据变化，相关页面自动刷新
export function initViews() {
  $('search').addEventListener('input', () => lastData && !lastData.render && renderList());
  $('search').addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      event.target.value = '';
      renderList();
    }
  });
  const when = (test) => () => test() && renderView();
  on('library-changed', when(() => state.view === 'library'));
  on('playlists-changed', when(() => state.view.startsWith('playlist:')));
  on('recent-changed', when(() => state.view === 'recent'));
  on('queue-changed', when(() => state.view === 'queue'));
  on('song-changed', () => markActive($('view-body')));
  on('favorites-changed', () => updateHearts($('view-body'), songsByKey));
}
