// 在线搜索页：选一个插件，输入关键词搜索，结果可以直接播放、收藏、加入歌单。
// 搜索结果保存在这个模块里，切到别的页面再回来还在。

import { Plugins, SearchOnline } from '../../wailsjs/go/main/App';
import { icons } from './icons.js';
import { playSongs } from './player.js';
import { renderSongList, updateHearts } from './songlist.js';
import { registerView, showView } from './views.js';
import { on } from './state.js';
import { escapeHtml, errorText, songKey } from './util.js';

const search = {
  platform: '', // 选中的插件平台名
  query: '',
  page: 0, // 已经加载到第几页，0 表示还没搜索
  songs: [],
  isEnd: true,
  loading: false,
  error: '',
};

let token = 0; // 每次新搜索加 1，丢掉过期的结果
let container = null; // 当前画在哪个元素里（离开页面后为 null）

export function initOnline() {
  registerView('online', loadOnlineView);
  on('favorites-changed', () => {
    if (container) updateHearts(container, new Map(search.songs.map((s) => [songKey(s), s])));
  });
  on('view-changed', (view) => {
    if (view !== 'online') container = null;
  });
}

async function loadOnlineView() {
  const plugins = ((await Plugins()) || []).filter((p) => p.enabled && p.canSearch && !p.error);
  if (!plugins.some((p) => p.platform === search.platform)) {
    search.platform = plugins[0]?.platform || '';
  }
  return {
    title: '在线搜索',
    subtitle: plugins.length ? `用插件搜索歌曲 · ${plugins.length} 个可用插件` : '用插件搜索歌曲',
    actions: [{ label: '管理插件', icon: icons.plugin, run: () => showView('plugins') }],
    render: (body) => renderOnline(body, plugins),
  };
}

function renderOnline(body, plugins) {
  if (!plugins.length) {
    container = null;
    body.innerHTML = `
      <div class="empty-state">
        <p>还没有可以搜索的插件</p>
        <p class="muted">在线歌曲来自插件，先去插件页安装一个吧</p>
        <button class="btn primary" type="button" id="go-plugins">去安装插件</button>
      </div>`;
    body.querySelector('#go-plugins').addEventListener('click', () => showView('plugins'));
    return;
  }

  body.innerHTML = `
    <form class="online-form" id="online-form">
      <select class="select" id="online-platform" aria-label="插件">
        ${plugins
          .map((p) => `<option value="${escapeHtml(p.platform)}" ${p.platform === search.platform ? 'selected' : ''}>${escapeHtml(p.platform)}</option>`)
          .join('')}
      </select>
      <input class="text-input" id="online-query" type="search" placeholder="输入歌名或歌手" autocomplete="off"
        value="${escapeHtml(search.query)}" maxlength="100" />
      <button class="btn primary" type="submit">${icons.search}<span>搜索</span></button>
    </form>
    <div class="online-results" id="online-results"></div>`;

  container = body.querySelector('#online-results');
  body.querySelector('#online-platform').addEventListener('change', (event) => {
    search.platform = event.target.value;
  });
  body.querySelector('#online-form').addEventListener('submit', (event) => {
    event.preventDefault();
    const query = body.querySelector('#online-query').value.trim();
    if (query) startSearch(query);
  });
  renderResults();
  if (!search.query) body.querySelector('#online-query').focus();
}

// startSearch 开始一次新的搜索（从第 1 页开始）
function startSearch(query) {
  search.query = query;
  search.page = 0;
  search.songs = [];
  search.isEnd = false;
  search.error = '';
  token++;
  loadMore();
}

// loadMore 加载下一页
async function loadMore() {
  const myToken = token;
  search.loading = true;
  search.error = '';
  renderResults();
  try {
    const result = await SearchOnline(search.platform, search.query, search.page + 1);
    if (myToken !== token) return;
    search.page++;
    search.songs = search.songs.concat(result?.data || []);
    search.isEnd = !!result?.isEnd || !(result?.data || []).length;
  } catch (err) {
    if (myToken !== token) return;
    search.error = errorText(err);
  } finally {
    if (myToken === token) {
      search.loading = false;
      renderResults();
    }
  }
}

function renderResults() {
  if (!container) return;
  const scrollBox = container.closest('.view-body');
  const scrollTop = scrollBox?.scrollTop || 0;

  if (!search.query) {
    container.innerHTML = '<div class="empty-state"><p>输入关键词，按回车搜索</p></div>';
    return;
  }
  if (!search.songs.length) {
    let text = `<p>没有找到和“${escapeHtml(search.query)}”有关的歌</p>`;
    if (search.loading) text = '<p>正在搜索…</p>';
    if (search.error) text = `<p>搜索失败</p><p class="muted">${escapeHtml(search.error)}</p>`;
    container.innerHTML = `<div class="empty-state">${text}</div>`;
    return;
  }

  const songs = search.songs;
  renderSongList(container, {
    groups: [{ items: songs.map((song, index) => ({ song, index })) }],
    rowActions: ['favorite', 'next', 'add'],
    onPlay: (item) => playSongs(songs, item.index),
  });

  const footer = document.createElement('div');
  footer.className = 'online-footer';
  if (search.error) {
    footer.innerHTML = `<span class="muted">加载失败：${escapeHtml(search.error)}</span>`;
  }
  if (search.loading) {
    footer.insertAdjacentHTML('beforeend', '<span class="muted">正在加载…</span>');
  } else if (!search.isEnd) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'btn';
    button.textContent = search.error ? '重试' : '加载更多';
    button.addEventListener('click', loadMore);
    footer.appendChild(button);
  } else {
    footer.insertAdjacentHTML('beforeend', `<span class="muted">共 ${songs.length} 首，没有更多了</span>`);
  }
  container.appendChild(footer);
  if (scrollBox) scrollBox.scrollTop = scrollTop;
}
