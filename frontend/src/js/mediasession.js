// 系统媒体控制：把正在播放的歌告诉系统（Windows 的音量浮窗、任务栏、键盘上的媒体键都能用），
// 并响应系统发来的 播放 / 暂停 / 上一首 / 下一首 / 拖动进度。
// 用的是浏览器标准的 navigator.mediaSession，WebView2 支持它。

import { audio, coverUrl, next, previous, togglePlay } from './player.js';
import { on, currentSong } from './state.js';
import { displayName, guard } from './util.js';

export function initMediaSession() {
  const session = navigator.mediaSession;
  if (!session || typeof MediaMetadata === 'undefined') return;
  const player = audio();

  on('song-changed', (song) => {
    session.metadata = song ? metadataOf(song) : null;
    if (!song) session.playbackState = 'none';
  });
  player.addEventListener('play', () => {
    session.playbackState = 'playing';
  });
  player.addEventListener('pause', () => {
    session.playbackState = 'paused';
  });
  player.addEventListener('durationchange', () => updatePosition(session, player));
  player.addEventListener('seeked', () => updatePosition(session, player));

  const handlers = {
    play: () => (currentSong() ? player.play().catch(() => {}) : togglePlay()),
    pause: () => player.pause(),
    stop: () => player.pause(),
    previoustrack: guard(previous),
    nexttrack: guard(next),
    seekto: (details) => seek(player, details.seekTime),
    seekbackward: (details) => seek(player, player.currentTime - (details.seekOffset || 10)),
    seekforward: (details) => seek(player, player.currentTime + (details.seekOffset || 10)),
  };
  for (const [action, handler] of Object.entries(handlers)) {
    try {
      session.setActionHandler(action, handler);
    } catch {
      // 老版本的 WebView 不认识某些操作（比如 stop），跳过就行
    }
  }
}

// metadataOf 生成系统显示用的歌曲信息。封面地址要写成完整的网址
function metadataOf(song) {
  const cover = coverUrl(song);
  return new MediaMetadata({
    title: displayName(song),
    artist: song.artist || '未知歌手',
    album: song.album || '',
    artwork: cover ? [{ src: new URL(cover, location.href).href, sizes: '512x512' }] : [],
  });
}

function seek(player, time) {
  if (!Number.isFinite(player.duration) || !Number.isFinite(time)) return;
  player.currentTime = Math.min(Math.max(0, time), player.duration);
}

// updatePosition 告诉系统进度条的总长度和当前位置（时长未知的时候不设置）
function updatePosition(session, player) {
  if (!session.setPositionState || !Number.isFinite(player.duration) || player.duration <= 0) return;
  try {
    session.setPositionState({
      duration: player.duration,
      playbackRate: player.playbackRate || 1,
      position: Math.min(player.currentTime, player.duration),
    });
  } catch {
    // 参数不合法（比如 position 比 duration 大一点点）时浏览器会报错，忽略
  }
}
