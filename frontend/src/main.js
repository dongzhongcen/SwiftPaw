// 前端入口：搭好页面结构，初始化各个模块，然后恢复上次的播放状态。

// 要最先导入：Android 上它负责准备好调用 Go 内核的 window.go
import './js/platform.js';

import './css/theme.css';
import './css/base.css';
import './css/layout.css';
import './css/player.css';
import './css/songlist.css';
import './css/dialog.css';
import './css/lyrics.css';
import './css/settings.css';
import './css/online.css';
import './css/plugins.css';
import './css/mobile.css';

import { QueueSetMode } from '../wailsjs/go/main/App';
import { layoutHtml } from './js/layout.js';
import { initPlayer, applyQueue, setVolume, togglePlay } from './js/player.js';
import { initSidebar } from './js/sidebar.js';
import { initViews, showView } from './js/views.js';
import { initLyrics } from './js/lyrics.js';
import { initSettings } from './js/settings.js';
import { initOnline } from './js/online.js';
import { initPlugins } from './js/plugins.js';
import { initMediaSession } from './js/mediasession.js';
import { initAbout } from './js/about.js';
import { initShell } from './js/shell.js';
import { applyTheme } from './js/theme.js';
import { refreshFavorites, refreshPlaylists, toggleFavorite, isFavorite } from './js/actions.js';
import { loadFolder } from './js/library.js';
import { loadConfig, finishRestoring } from './js/config.js';
import { state, on, currentSong } from './js/state.js';
import { $, guard, showError, toast } from './js/util.js';
import { icons } from './js/icons.js';
import { isAndroid } from './js/platform.js';

document.querySelector('#app').innerHTML = layoutHtml;

initPlayer();
initSidebar();
initViews();
initLyrics();
initSettings();
initOnline();
initPlugins();
if (!isAndroid) initMediaSession(); // Android 上系统媒体控制由原生的播放服务负责
initAbout();
initShell();
initCoverButton();
initFavoriteButton();
initKeyboard();
startApp();

async function startApp() {
  try {
    const config = await loadConfig();
    applyTheme(config.theme);
    setVolume(config.volume);
    await applyQueue(await QueueSetMode(config.playMode || 'sequence'), false);
    await Promise.all([refreshPlaylists(), refreshFavorites()]).catch(showError);

    if (config.lastFolder) {
      await loadFolder(config.lastFolder, config.lastSong || '');
    } else {
      toast(isAndroid ? '请先扫描本机音乐' : '请选择音乐文件夹');
    }
  } catch (err) {
    showError(err);
  } finally {
    finishRestoring();
    showView(state.view);
  }
}

// 播放栏上的红心：收藏 / 取消收藏正在播放的歌
function initFavoriteButton() {
  const button = $('pb-fav');
  const update = () => {
    const song = currentSong();
    const liked = isFavorite(song);
    button.disabled = !song;
    button.innerHTML = liked ? icons.heartFilled : icons.heart;
    button.classList.toggle('on', liked);
    button.title = liked ? '取消收藏' : '收藏';
  };
  button.addEventListener('click', guard(async () => {
    const song = currentSong();
    if (song) await toggleFavorite(song);
  }));
  on('song-changed', update);
  on('favorites-changed', update);
}

// 点播放栏的封面打开歌词页，再点一次回到之前的页面
function initCoverButton() {
  $('cover-btn').addEventListener('click', () => {
    if (state.view === 'lyrics') showView(state.previousView || 'library');
    else showView('lyrics');
  });
}

// 空格键：播放 / 暂停（焦点在输入框里或者对话框打开时不处理）
function initKeyboard() {
  document.addEventListener('keydown', (event) => {
    if (event.code !== 'Space' || event.repeat) return;
    const target = event.target;
    const typing = target.closest?.('input, textarea, select, [contenteditable="true"]');
    if (typing || document.querySelector('dialog[open]')) return;
    event.preventDefault(); // 不让空格去“点击”当前聚焦的按钮，也不滚动页面
    togglePlay();
  });
}
