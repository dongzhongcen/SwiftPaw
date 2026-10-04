// 手机布局下的外壳：标签栏、页面左上角的返回按钮、“我的”页面、全屏播放页，以及 Android 的返回键。
// 桌面布局下这些元素都是隐藏的，这里的代码也基本不起作用（见 responsive.js）。

import { icons } from './icons.js';
import { createPlaylist } from './actions.js';
import { registerView, renderView, showView } from './views.js';
import { mobileLayout, narrowScreen } from './responsive.js';
import { isAndroid } from './platform.js';
import { nativeCall, onNative } from './native/android.js';
import { state, on } from './state.js';
import { $, escapeHtml, guard } from './util.js';

// 标签栏上的 5 个标签，其他页面都算在“我的”下面
const tabs = ['library', 'mine', 'online', 'plugins', 'settings'];

const app = () => document.querySelector('.app');

export function initShell() {
  registerView('mine', loadMineView);

  $('tabbar').addEventListener('click', (event) => {
    const item = event.target.closest('[data-tab]');
    if (item) showView(item.dataset.tab);
  });
  $('view-back').addEventListener('click', goBack);

  initFullPlayer();

  on('view-changed', updateNavigation);
  const rerenderMine = () => state.view === 'mine' && renderView();
  on('playlists-changed', rerenderMine);
  on('favorites-changed', rerenderMine);
  on('queue-changed', rerenderMine);

  mobileLayout.addEventListener('change', () => {
    if (!mobileLayout.matches) closePlayer(); // 窗口拉宽回到桌面布局时，收起全屏播放页
    updateNavigation();
  });
  // 宽窄切换时歌曲列表的行尾按钮不一样，重画一下
  narrowScreen.addEventListener('change', () => renderView());

  if (isAndroid) onNative('backButton', guard(handleBack));
  updateNavigation();
}

// parentOf 返回页面的上一级（左上角返回按钮回到哪里），标签页本身没有上一级
function parentOf(view) {
  if (tabs.includes(view)) return '';
  if (view === 'lyrics') {
    const previous = state.previousView;
    return previous && previous !== 'lyrics' ? previous : 'mine';
  }
  return 'mine';
}

function tabOf(view) {
  if (tabs.includes(view)) return view;
  if (view === 'lyrics' && tabs.includes(state.previousView)) return state.previousView;
  return 'mine';
}

function updateNavigation() {
  const current = tabOf(state.view);
  document.querySelectorAll('#tabbar [data-tab]').forEach((item) => {
    const active = item.dataset.tab === current;
    item.classList.toggle('active', active);
    if (active) item.setAttribute('aria-current', 'page');
    else item.removeAttribute('aria-current');
  });
  $('view-back').hidden = !mobileLayout.matches || !parentOf(state.view);
}

function goBack() {
  const parent = parentOf(state.view);
  if (parent) showView(parent);
}

// handleBack 处理 Android 的返回键：先关对话框，再收起播放页，再回上一级页面，
// 到了“本地音乐”再按就退到后台（音乐继续放），不直接关掉应用
async function handleBack() {
  const dialog = document.querySelector('dialog[open]');
  if (dialog) {
    // 和按 Esc 一样：对话框自己会把结果当作“取消”
    if (dialog.dispatchEvent(new Event('cancel', { cancelable: true }))) dialog.close();
    return;
  }
  if (isPlayerOpen()) {
    closePlayer();
    return;
  }
  const parent = parentOf(state.view);
  if (parent && mobileLayout.matches) {
    await showView(parent);
    return;
  }
  if (state.view !== 'library') {
    await showView('library');
    return;
  }
  await nativeCall('minimize');
}

// ---- “我的”页面：收藏、最近播放、播放队列和自己建的歌单 ----

async function loadMineView() {
  const own = state.playlists.filter((p) => !p.builtin);
  return {
    title: '我的',
    subtitle: `${own.length} 个歌单`,
    actions: [{ label: '新建歌单', icon: icons.plus, run: newPlaylist }],
    render: (body) => renderMine(body, own),
  };
}

async function newPlaylist() {
  const playlist = await createPlaylist();
  if (playlist) showView('playlist:' + playlist.id);
}

function mineItem(view, icon, label, count = '') {
  return `
    <button class="mine-item" type="button" data-view="${escapeHtml(view)}">
      ${icon}<span class="mine-label">${escapeHtml(label)}</span>
      <span class="mine-count">${escapeHtml(count)}</span>${icons.chevronRight}
    </button>`;
}

function renderMine(body, own) {
  const favorites = state.playlists.find((p) => p.builtin);
  const queueSize = state.queue.songs.length;
  body.innerHTML = `
    <div class="mine">
      <div class="mine-group">
        ${favorites ? mineItem(`playlist:${favorites.id}`, icons.heart, '我喜欢', favorites.count ? String(favorites.count) : '') : ''}
        ${mineItem('recent', icons.clock, '最近播放')}
        ${mineItem('queue', icons.queue, '播放队列', queueSize ? String(queueSize) : '')}
        ${mineItem('lyrics', icons.lyrics, '歌词')}
      </div>
      <h2 class="mine-heading">我的歌单</h2>
      <div class="mine-group">
        ${
          own.length
            ? own.map((p) => mineItem(`playlist:${p.id}`, icons.list, p.name, p.count ? String(p.count) : '')).join('')
            : '<p class="mine-empty">还没有歌单，点上面的“新建歌单”建一个</p>'
        }
      </div>
    </div>`;
  body.querySelectorAll('[data-view]').forEach((item) => {
    item.addEventListener('click', () => showView(item.dataset.view));
  });
}

// ---- 全屏播放页 ----
// 手机布局下底部是迷你播放条（封面、歌名、播放、下一首），点它打开全屏播放页。
// 全屏播放页和迷你播放条是同一个 <footer>，只是加了 player-open 这个 class，按钮都是同一批。

export function isPlayerOpen() {
  return app().classList.contains('player-open');
}

export function openPlayer() {
  if (!mobileLayout.matches) return;
  app().classList.add('player-open');
}

export function closePlayer() {
  app().classList.remove('player-open');
}

function initFullPlayer() {
  const track = document.querySelector('.pb-track');
  // 用捕获阶段，这样在迷你播放条上点封面时不会触发桌面版“封面打开歌词页”的逻辑
  track.addEventListener(
    'click',
    (event) => {
      if (!mobileLayout.matches) return;
      if (!isPlayerOpen()) {
        event.stopPropagation();
        openPlayer();
      } else if (event.target.closest('#cover-btn')) {
        event.stopPropagation();
        closePlayer();
        showView('lyrics');
      }
    },
    true,
  );
  $('pb-collapse').addEventListener('click', closePlayer);
  $('pb-lyrics').addEventListener('click', () => {
    closePlayer();
    showView('lyrics');
  });
  $('pb-queue').addEventListener('click', () => {
    closePlayer();
    showView('queue');
  });

  // 在播放页上往下滑收起
  const bar = $('player-bar');
  let startY = null;
  bar.addEventListener('touchstart', (event) => {
    const fromControl = event.target.closest('input, .pb-controls');
    startY = isPlayerOpen() && !fromControl && event.touches.length === 1 ? event.touches[0].clientY : null;
  }, { passive: true });
  bar.addEventListener('touchend', (event) => {
    if (startY === null) return;
    const distance = event.changedTouches[0].clientY - startY;
    startY = null;
    if (distance > 90) closePlayer();
  });
}
