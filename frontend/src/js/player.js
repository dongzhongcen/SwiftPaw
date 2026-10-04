// 播放器：隐藏的 <audio> 加上底部播放栏的自定义控件。
// 播放顺序由 Go 里的队列决定，这里拿到队列快照后负责“真正去播放”和刷新播放栏。

import {
  PlayLibrary,
  PlaySongs,
  QueueNext,
  QueuePrevious,
  QueueSetMode,
  RecordPlay,
} from '../../wailsjs/go/main/App';
import { icons } from './icons.js';
import { saveConfig } from './config.js';
import { state, on, emit, currentSong } from './state.js';
import { $, displayName, songKey, formatTime, toast, guard } from './util.js';

// 播放模式，顺序就是点“模式”按钮时的切换顺序
export const playModes = [
  { value: 'sequence', label: '顺序播放', icon: icons.sequence },
  { value: 'repeat-one', label: '单曲循环', icon: icons.repeatOne },
  { value: 'shuffle', label: '随机播放', icon: icons.shuffle },
];

export const audio = () => $('audio');

let loadedKey = ''; // <audio> 里现在加载的是哪首歌
let recordedKey = ''; // 已经记录过“最近播放”的歌，同一次加载只记一次
let errorStreak = 0; // 连续播放失败的次数，防止所有歌都放不了时无限跳歌
let seeking = false; // 用户正在拖进度条

export function loadedSongKey() {
  return loadedKey;
}

export function initPlayer() {
  const player = audio();

  $('play').addEventListener('click', togglePlay);
  $('prev').addEventListener('click', guard(previous));
  $('next').addEventListener('click', guard(next));
  $('mode').addEventListener('click', guard(cycleMode));
  $('mute').addEventListener('click', () => {
    player.muted = !player.muted;
  });

  $('volume').addEventListener('input', (event) => {
    player.volume = Number(event.target.value) / 100;
    player.muted = false;
  });

  // 拖动进度条时先只更新时间文字，松开后再真正跳转
  const seek = $('seek');
  seek.addEventListener('input', () => {
    seeking = true;
    const time = (Number(seek.value) / 1000) * (player.duration || 0);
    $('time-current').textContent = formatTime(time);
    setRangeFill(seek, Number(seek.value) / 10);
  });
  seek.addEventListener('change', () => {
    if (Number.isFinite(player.duration)) {
      player.currentTime = (Number(seek.value) / 1000) * player.duration;
    }
    seeking = false;
  });

  player.addEventListener('timeupdate', updateProgress);
  player.addEventListener('durationchange', updateProgress);
  player.addEventListener('play', updatePlayButton);
  player.addEventListener('pause', updatePlayButton);

  player.addEventListener('ended', guard(async () => {
    await applyQueue(await QueueNext(true), true);
  }));

  player.addEventListener('playing', () => {
    errorStreak = 0;
    recordPlayOnce();
  });

  // 某首歌放不了（比如格式不支持或文件被删了），提示一下并跳到下一首
  player.addEventListener('error', guard(async () => {
    if (!loadedKey) return;
    errorStreak++;
    const name = $('pb-title').textContent;
    if (errorStreak >= state.queue.songs.length) {
      toast(`无法播放：${name}，队列里的歌都放不了，已停止`, true);
      return;
    }
    toast(`无法播放：${name}，已跳到下一首`, true);
    await applyQueue(await QueueNext(false), true, true);
  }));

  player.addEventListener('volumechange', () => {
    updateVolumeView();
    saveConfig({ volume: player.volume });
  });

  on('song-changed', updateTrackView);
  updateVolumeView();
}

// setVolume 在启动恢复时设置音量
export function setVolume(value) {
  const volume = Number(value);
  audio().volume = Number.isFinite(volume) ? Math.min(1, Math.max(0, volume)) : 1;
  updateVolumeView();
}

// applyQueue 拿到 Go 返回的最新队列后刷新界面；play 为 true 时开始播放
export async function applyQueue(newQueue, play, keepError = false) {
  const player = audio();
  state.queue = newQueue;
  if (!keepError) errorStreak = 0;
  const song = currentSong();

  if (!song) {
    player.pause();
    player.removeAttribute('src');
    player.load();
    loadedKey = '';
    emit('song-changed', null);
  } else if (songKey(song) !== loadedKey) {
    loadedKey = songKey(song);
    recordedKey = '';
    player.src = await mediaUrl(song);
    emit('song-changed', song);
  } else if (play) {
    player.currentTime = 0; // 同一首歌再放一次（比如单曲循环），从头开始
  }

  if (song && play) {
    player.play().catch(() => {});
  }

  updateModeButton();
  emit('queue-changed', state.queue);
  saveConfig({ lastSong: loadedKey, playMode: state.queue.mode });
}

// mediaUrl 返回 <audio> 要加载的地址。本地文件由 Go 的 /music 路由提供。
async function mediaUrl(song) {
  return '/music?path=' + encodeURIComponent(song.path);
}

// playSongs 用一组歌替换播放队列并从第 index 首开始播放（“播放全部”和点击某一行都用它）
export async function playSongs(songs, index = 0) {
  if (!songs.length) {
    toast('列表里还没有歌');
    return;
  }
  await applyQueue(await PlaySongs(songs, index), true);
}

// togglePlay 播放/暂停；还没有歌时从本地音乐第一首开始
export const togglePlay = guard(async () => {
  const player = audio();
  if (!currentSong()) {
    await startFromLibraryIfQueueEmpty();
    return;
  }
  if (player.paused) player.play().catch(() => {});
  else player.pause();
});

async function previous() {
  if (await startFromLibraryIfQueueEmpty()) return;
  await applyQueue(await QueuePrevious(), true);
}

async function next() {
  if (await startFromLibraryIfQueueEmpty()) return;
  await applyQueue(await QueueNext(false), true);
}

async function cycleMode() {
  const index = playModes.findIndex((mode) => mode.value === state.queue.mode);
  const nextMode = playModes[(index + 1) % playModes.length];
  await applyQueue(await QueueSetMode(nextMode.value), false);
  toast(nextMode.label);
}

// 队列还是空的时候按播放/上一首/下一首，就从本地音乐第一首开始放
async function startFromLibraryIfQueueEmpty() {
  if (state.queue.songs.length > 0) return false;
  if (state.library.length === 0) {
    toast('请先选择音乐文件夹');
    return true;
  }
  await applyQueue(await PlayLibrary(0), true);
  return true;
}

// 第一次真正开始播放（playing 事件）时记一次“最近播放”
function recordPlayOnce() {
  const song = currentSong();
  if (!song || recordedKey === loadedKey) return;
  recordedKey = loadedKey;
  RecordPlay(song)
    .then(() => emit('recent-changed'))
    .catch(() => {});
}

// ---- 播放栏的显示 ----

function updateModeButton() {
  const mode = playModes.find((m) => m.value === state.queue.mode) || playModes[0];
  const button = $('mode');
  button.innerHTML = mode.icon;
  button.title = mode.label;
  button.dataset.mode = mode.value;
  $('mode-label').textContent = mode.label;
}

function updatePlayButton() {
  const playing = !audio().paused;
  const button = $('play');
  button.innerHTML = playing ? icons.pause : icons.play;
  button.title = playing ? '暂停' : '播放';
  button.setAttribute('aria-label', button.title);
}

function updateProgress() {
  const player = audio();
  const duration = player.duration;
  $('time-total').textContent = formatTime(duration);
  if (seeking) return;
  $('time-current').textContent = formatTime(player.currentTime);
  const ratio = Number.isFinite(duration) && duration > 0 ? player.currentTime / duration : 0;
  const seek = $('seek');
  seek.value = String(Math.round(ratio * 1000));
  setRangeFill(seek, ratio * 100);
}

function updateVolumeView() {
  const player = audio();
  const volume = player.muted ? 0 : player.volume;
  const range = $('volume');
  range.value = String(Math.round(volume * 100));
  setRangeFill(range, volume * 100);
  const button = $('mute');
  button.innerHTML = player.muted || player.volume === 0 ? icons.mute : icons.volume;
  button.title = player.muted ? '取消静音' : '静音';
}

// 进度条左边已播放的部分用 CSS 变量 --fill 画出来
function setRangeFill(range, percent) {
  range.style.setProperty('--fill', `${Math.min(100, Math.max(0, percent))}%`);
}

// updateTrackView 刷新播放栏左边的封面、歌名和歌手
export function updateTrackView(song) {
  const cover = $('cover');
  if (!song) {
    $('pb-title').textContent = '尚未选择歌曲';
    $('pb-artist').textContent = state.library.length ? '在列表里点一首歌开始播放' : '请选择一个音乐文件夹';
    cover.hidden = true;
    $('time-current').textContent = '0:00';
    $('time-total').textContent = '0:00';
    setRangeFill($('seek'), 0);
    return;
  }
  $('pb-title').textContent = displayName(song);
  $('pb-title').title = displayName(song);
  $('pb-artist').textContent = [song.artist || '未知歌手', song.album].filter(Boolean).join(' · ');
  cover.hidden = false;
  cover.onerror = () => {
    cover.hidden = true;
  };
  cover.src = coverUrl(song);
}

// coverUrl 返回封面地址：本地歌曲从文件标签里读
export function coverUrl(song) {
  return '/cover?path=' + encodeURIComponent(song.path);
}
