// 歌曲列表的渲染。所有页面（本地音乐、歌单、最近播放、播放队列）都用同一个列表组件，
// 只是每行显示哪些操作按钮、点了之后做什么不一样。

import { icons } from './icons.js';
import { isFavorite, toggleFavorite, addNext, addToPlaylist } from './actions.js';
import { loadedSongKey } from './player.js';
import { escapeHtml, displayName, songKey, guard } from './util.js';

// 行尾可以出现的操作按钮
const rowActions = {
  favorite: (song) => ({
    icon: isFavorite(song) ? icons.heartFilled : icons.heart,
    title: isFavorite(song) ? '取消收藏' : '收藏',
    className: isFavorite(song) ? 'on' : '',
    run: () => toggleFavorite(song),
  }),
  next: (song) => ({ icon: icons.playNext, title: '下一首播放', run: () => addNext(song) }),
  add: (song) => ({ icon: icons.addTo, title: '加入歌单', run: () => addToPlaylist(song) }),
  remove: (song, item, options) => ({ icon: icons.close, title: '移除', run: () => options.onRemove(item) }),
};

/**
 * renderSongList 把歌曲画到 container 里。
 * options:
 *   groups:  [{ label, items: [{ song, index }] }]，label 为空时不显示分组标题
 *   rowActions: 每行显示的按钮，比如 ['favorite', 'next', 'add', 'remove']
 *   onPlay(item):   点击一行时调用
 *   onRemove(item): 点击“移除”时调用
 *   empty:   列表为空时显示的 HTML
 */
export function renderSongList(container, options) {
  const { groups, rowActions: actions = [], empty = '' } = options;
  const total = groups.reduce((n, g) => n + g.items.length, 0);
  if (total === 0) {
    container.innerHTML = `<div class="empty-state">${empty}</div>`;
    return;
  }

  container.innerHTML = `
    <div class="song-head">
      <span class="col-index">#</span><span>标题</span><span>歌手</span><span>专辑</span><span></span>
    </div>
    <div class="song-rows"></div>`;
  const rows = container.querySelector('.song-rows');

  let number = 0;
  groups.forEach((group) => {
    if (group.label) {
      const label = document.createElement('div');
      label.className = 'group-label';
      label.textContent = group.label;
      rows.appendChild(label);
    }
    group.items.forEach((item) => {
      number++;
      rows.appendChild(createRow(item, number, actions, options));
    });
  });
  markActive(container);
}

function createRow(item, number, actions, options) {
  const { song } = item;
  const row = document.createElement('div');
  row.className = 'song-row';
  row.dataset.key = songKey(song);
  row.title = song.path || '';
  row.innerHTML = `
    <span class="col-index"><span class="num">${String(number).padStart(2, '0')}</span><span class="eq">${icons.play}</span></span>
    <span class="col-title">
      <span class="title-text">${escapeHtml(displayName(song))}</span>
      ${song.format ? `<span class="chip">${escapeHtml(song.format)}</span>` : ''}
    </span>
    <span class="col-artist">${escapeHtml(song.artist || '未知歌手')}</span>
    <span class="col-album">${escapeHtml(song.album || '')}</span>
    <span class="col-actions"></span>`;

  const cell = row.querySelector('.col-actions');
  actions.forEach((name) => {
    const action = rowActions[name](song, item, options);
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `icon-btn small row-btn ${action.className || ''}`;
    button.dataset.action = name;
    button.title = action.title;
    button.setAttribute('aria-label', action.title);
    button.innerHTML = action.icon;
    button.addEventListener('click', guard(async (event) => {
      event.stopPropagation(); // 不要触发整行的“播放”
      await action.run();
    }));
    cell.appendChild(button);
  });

  row.addEventListener('click', guard(() => options.onPlay?.(item)));
  return row;
}

// markActive 高亮正在播放的那一行（按 key 比较，因为不同列表里顺序不一样）
export function markActive(container, scroll = false) {
  const key = loadedSongKey();
  container.querySelectorAll('.song-row').forEach((row) => {
    const active = !!key && row.dataset.key === key;
    row.classList.toggle('active', active);
    if (active && scroll) row.scrollIntoView({ block: 'nearest' });
  });
}

// updateHearts 收藏状态变了以后，只更新红心按钮，不用整个列表重画
export function updateHearts(container, songsByKey) {
  container.querySelectorAll('.row-btn[data-action=favorite]').forEach((button) => {
    const song = songsByKey.get(button.closest('.song-row').dataset.key);
    if (!song) return;
    const liked = isFavorite(song);
    button.classList.toggle('on', liked);
    button.innerHTML = liked ? icons.heartFilled : icons.heart;
    button.title = liked ? '取消收藏' : '收藏';
  });
}
