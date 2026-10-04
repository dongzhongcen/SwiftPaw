// 左边的侧边栏：导航、歌曲数量、我的歌单列表

import { icons } from './icons.js';
import { createPlaylist } from './actions.js';
import { showView } from './views.js';
import { state, on } from './state.js';
import { $, escapeHtml, guard } from './util.js';

export function initSidebar() {
  $('nav').addEventListener('click', onNavClick);
  $('playlist-nav').addEventListener('click', onNavClick);
  $('new-playlist').addEventListener('click', guard(async () => {
    const playlist = await createPlaylist();
    if (playlist) showView('playlist:' + playlist.id);
  }));

  on('view-changed', markActive);
  on('playlists-changed', renderPlaylists);
  on('library-changed', () => {
    $('count-library').textContent = state.library.length || '';
  });
  on('queue-changed', () => {
    $('count-queue').textContent = String(state.queue.songs.length);
  });
}

function onNavClick(event) {
  const item = event.target.closest('[data-view]');
  if (item) showView(item.dataset.view);
}

function renderPlaylists() {
  const favorites = state.playlists.find((p) => p.builtin);
  $('count-favorites').textContent = favorites?.count || '';

  const own = state.playlists.filter((p) => !p.builtin);
  $('playlist-nav').innerHTML = own.length
    ? own
        .map(
          (p) => `
      <button class="nav-item" type="button" data-view="playlist:${p.id}" title="${escapeHtml(p.name)}">
        ${icons.list}<span class="nav-label">${escapeHtml(p.name)}</span><span class="nav-count">${p.count || ''}</span>
      </button>`,
        )
        .join('')
    : '<p class="nav-empty">点 + 新建一个歌单</p>';
  markActive();
}

function markActive() {
  document.querySelectorAll('.sidebar [data-view]').forEach((item) => {
    item.classList.toggle('active', item.dataset.view === state.view);
  });
}
