import './style.css';
import './app.css';

import {
  LoadConfig,
  PlayLibrary,
  QueueAddNext,
  QueueNext,
  QueuePlayAt,
  QueuePrevious,
  QueueRemove,
  QueueSetMode,
  SaveConfig,
  ScanMusic,
  SelectFolder,
} from '../wailsjs/go/main/App';

document.querySelector('#app').innerHTML = `
  <main class="shell">
    <header class="hero">
      <p class="eyebrow">SwiftPaw Music Player</p>
      <h1>极拍 SwiftPaw</h1>
    </header>

    <section class="now-playing">
      <img id="cover" class="cover" alt="当前歌曲封面" />
      <div class="track-meta">
        <p id="current-title" class="track-title">尚未选择歌曲</p>
        <p id="current-artist" class="track-artist">请选择一个音乐文件夹</p>
      </div>
    </section>

    <section class="controls">
      <button id="pick" type="button">选择音乐文件夹</button>
      <audio id="player" controls></audio>
      <div class="button-row">
        <button id="previous" type="button">上一首</button>
        <button id="next" type="button">下一首</button>
        <button id="mode" type="button">顺序播放</button>
      </div>
    </section>

    <p id="info" class="info"></p>

    <nav class="tabs">
      <button id="tab-library" class="tab active" type="button">歌曲库 <span id="library-count">0</span></button>
      <button id="tab-queue" class="tab" type="button">播放队列 <span id="queue-count">0</span></button>
    </nav>
    <ul id="list" class="song-list"></ul>
    <ul id="queue-list" class="song-list" hidden></ul>
  </main>
`;

const info = document.getElementById('info');
const list = document.getElementById('list');
const queueList = document.getElementById('queue-list');
const player = document.getElementById('player');
const cover = document.getElementById('cover');
const currentTitle = document.getElementById('current-title');
const currentArtist = document.getElementById('current-artist');
const pickButton = document.getElementById('pick');
const previousButton = document.getElementById('previous');
const nextButton = document.getElementById('next');
const modeButton = document.getElementById('mode');
const libraryTab = document.getElementById('tab-library');
const queueTab = document.getElementById('tab-queue');
const libraryCount = document.getElementById('library-count');
const queueCount = document.getElementById('queue-count');

// 播放模式的显示名字，顺序就是点“模式”按钮时的切换顺序
const playModes = [
  { value: 'sequence', label: '顺序播放' },
  { value: 'repeat-one', label: '单曲循环' },
  { value: 'shuffle', label: '随机播放' },
];

let library = []; // 歌曲库：当前文件夹扫描到的所有歌
let queue = { songs: [], current: -1, mode: 'sequence', upcoming: [] }; // Go 那边返回的队列快照
let loadedPath = ''; // <audio> 里现在加载的是哪首歌
let currentFolder = '';
let isRestoring = false;
let errorStreak = 0; // 连续播放失败的次数，防止所有歌都放不了时无限跳歌
let volumeTimer = 0;

cover.hidden = true;

pickButton.addEventListener('click', async () => {
  try {
    const dir = await SelectFolder();
    if (!dir) return;

    await loadFolder(dir, '');
  } catch (err) {
    showError(err);
  }
});

previousButton.addEventListener('click', async () => {
  if (await startFromLibraryIfQueueEmpty()) return;
  await applyQueue(await QueuePrevious(), true);
});

nextButton.addEventListener('click', async () => {
  if (await startFromLibraryIfQueueEmpty()) return;
  await applyQueue(await QueueNext(false), true);
});

modeButton.addEventListener('click', async () => {
  const index = playModes.findIndex((mode) => mode.value === queue.mode);
  const nextMode = playModes[(index + 1) % playModes.length].value;
  await applyQueue(await QueueSetMode(nextMode), false);
});

libraryTab.addEventListener('click', () => showTab('library'));
queueTab.addEventListener('click', () => showTab('queue'));

player.addEventListener('ended', async () => {
  await applyQueue(await QueueNext(true), true);
});

player.addEventListener('playing', () => {
  errorStreak = 0;
});

// 某首歌放不了（比如格式不支持或文件被删了），提示一下并跳到下一首
player.addEventListener('error', async () => {
  if (!loadedPath) return;
  errorStreak++;
  const name = currentTitle.textContent;
  if (errorStreak >= queue.songs.length) {
    info.textContent = `无法播放：${name}，队列里的歌都放不了，已停止`;
    return;
  }
  info.textContent = `无法播放：${name}，已跳到下一首`;
  await applyQueue(await QueueNext(false), true, true);
});

// 音量变化后半秒再保存，拖动音量条时不会一直写文件
player.addEventListener('volumechange', () => {
  clearTimeout(volumeTimer);
  volumeTimer = setTimeout(saveCurrentState, 500);
});

startApp();

async function startApp() {
  try {
    const config = await LoadConfig();
    isRestoring = true;
    player.volume = clampVolume(config.volume);
    await applyQueue(await QueueSetMode(config.playMode || 'sequence'), false);

    if (!config.lastFolder) {
      info.textContent = '请选择音乐文件夹';
      return;
    }
    await loadFolder(config.lastFolder, config.lastSong || '');
  } catch (err) {
    showError(err);
  } finally {
    isRestoring = false;
  }
}

async function loadFolder(dir, songPathToSelect) {
  currentFolder = dir;
  info.textContent = '正在扫描：' + dir;

  library = (await ScanMusic(dir)) || [];
  renderLibrary();
  info.textContent = `共找到 ${library.length} 首歌`;

  if (songPathToSelect) {
    const savedIndex = library.findIndex((song) => song.path === songPathToSelect);
    if (savedIndex !== -1) {
      await applyQueue(await PlayLibrary(savedIndex), false);
      info.textContent = `已恢复上次播放：${displayName(library[savedIndex])}`;
    }
  }

  await saveCurrentState();
}

// 队列还是空的时候按上一首/下一首，就从歌曲库第一首开始放
async function startFromLibraryIfQueueEmpty() {
  if (queue.songs.length > 0 || library.length === 0) return false;
  await applyQueue(await PlayLibrary(0), true);
  return true;
}

// applyQueue 拿到 Go 返回的最新队列后刷新界面；play 为 true 时开始播放
async function applyQueue(newQueue, play, keepError = false) {
  queue = newQueue;
  if (!keepError) errorStreak = 0;
  const song = queue.current >= 0 ? queue.songs[queue.current] : null;

  if (!song) {
    player.pause();
    player.removeAttribute('src');
    player.load();
    loadedPath = '';
    showNothingPlaying();
  } else if (song.path !== loadedPath) {
    loadedPath = song.path;
    player.src = '/music?path=' + encodeURIComponent(song.path);
    updateCurrentSongView(song);
  } else if (play) {
    player.currentTime = 0; // 同一首歌再放一次（比如单曲循环），从头开始
  }

  if (song && play) {
    player.play().catch(() => {});
    info.textContent = '正在播放：' + displayName(song);
  }

  modeButton.textContent = playModes.find((mode) => mode.value === queue.mode)?.label || '顺序播放';
  queueCount.textContent = String(queue.songs.length);
  updateActiveSong();
  renderQueue();
  await saveCurrentState();
}

function renderLibrary() {
  list.innerHTML = '';
  libraryCount.textContent = String(library.length);

  library.forEach((song, index) => {
    const li = createSongRow(song, '下一首播放', async () => {
      await applyQueue(await QueueAddNext(song), false);
      info.textContent = `已设为下一首播放：${displayName(song)}`;
    });
    li.addEventListener('click', async () => {
      await applyQueue(await PlayLibrary(index), true);
    });
    list.appendChild(li);
  });

  updateActiveSong();
}

function renderQueue() {
  queueList.innerHTML = '';

  if (queue.songs.length === 0) {
    queueList.innerHTML = '<li class="queue-empty">播放队列是空的，在歌曲库里点一首歌开始播放</li>';
    return;
  }

  const rows = [];
  if (queue.current >= 0) rows.push({ index: queue.current, label: '正在播放' });
  queue.upcoming.forEach((index, i) => rows.push({ index, label: i === 0 ? '接下来' : '' }));

  rows.forEach(({ index, label }) => {
    if (label) {
      const header = document.createElement('li');
      header.className = 'queue-label';
      header.textContent = label;
      queueList.appendChild(header);
    }

    const song = queue.songs[index];
    const li = createSongRow(song, '移除', async () => {
      await applyQueue(await QueueRemove(index), !player.paused);
    });
    li.classList.toggle('active', index === queue.current);
    li.addEventListener('click', async () => {
      await applyQueue(await QueuePlayAt(index), true);
    });
    queueList.appendChild(li);
  });

  const hidden = queue.songs.length - rows.length;
  if (hidden > 0) {
    const more = document.createElement('li');
    more.className = 'queue-empty';
    more.textContent = `还有 ${hidden} 首在后面（已经放过的或排得比较远的）`;
    queueList.appendChild(more);
  }
}

// 一行歌曲：歌名、格式、歌手，再加一个操作按钮
function createSongRow(song, actionLabel, onAction) {
  const li = document.createElement('li');
  li.className = 'song-item';
  li.title = song.path;
  li.dataset.path = song.path;
  li.innerHTML = `
    <span class="song-title">${escapeHtml(displayName(song))}</span>
    <span class="song-format">${escapeHtml(song.format || '')}</span>
    <span class="song-artist">${escapeHtml(song.artist || '未知歌手')}</span>
    <button class="row-action" type="button">${escapeHtml(actionLabel)}</button>
  `;
  li.querySelector('.row-action').addEventListener('click', async (event) => {
    event.stopPropagation(); // 不要触发整行的“播放”
    try {
      await onAction();
    } catch (err) {
      showError(err);
    }
  });
  return li;
}

function showTab(name) {
  const isQueue = name === 'queue';
  list.hidden = isQueue;
  queueList.hidden = !isQueue;
  libraryTab.classList.toggle('active', !isQueue);
  queueTab.classList.toggle('active', isQueue);
}

function updateCurrentSongView(song) {
  currentTitle.textContent = displayName(song);
  currentArtist.textContent = [song.artist || '未知歌手', song.album].filter(Boolean).join(' · ');
  cover.hidden = false;
  cover.src = '/cover?path=' + encodeURIComponent(song.path) + '&t=' + Date.now();
  cover.onerror = () => {
    cover.hidden = true;
  };
}

function showNothingPlaying() {
  currentTitle.textContent = '尚未选择歌曲';
  currentArtist.textContent = library.length ? '在歌曲库里点一首歌开始播放' : '请选择一个音乐文件夹';
  cover.hidden = true;
}

// 歌曲库里高亮正在播放的歌（按路径比较，因为歌曲库和队列的顺序可能不一样）
function updateActiveSong() {
  let activeItem = null;
  list.querySelectorAll('.song-item').forEach((item) => {
    const active = item.dataset.path === loadedPath;
    item.classList.toggle('active', active);
    if (active) activeItem = item;
  });
  if (activeItem) activeItem.scrollIntoView({ block: 'nearest' });
}

async function saveCurrentState() {
  if (isRestoring) return;

  await SaveConfig({
    lastFolder: currentFolder,
    lastSong: loadedPath,
    playMode: queue.mode,
    volume: player.volume,
  });
}

function clampVolume(value) {
  const volume = Number(value);
  if (!Number.isFinite(volume)) return 1;
  return Math.min(1, Math.max(0, volume));
}

function displayName(song) {
  return song.title || song.name || '未知歌曲';
}

function showError(err) {
  info.textContent = '出错了：' + err;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}
