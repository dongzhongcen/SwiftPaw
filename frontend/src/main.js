import './style.css';
import './app.css';

import { LoadConfig, SaveConfig, ScanMusic, SelectFolder } from '../wailsjs/go/main/App';

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
    <ul id="list" class="song-list"></ul>
  </main>
`;

const info = document.getElementById('info');
const list = document.getElementById('list');
const player = document.getElementById('player');
const cover = document.getElementById('cover');
const currentTitle = document.getElementById('current-title');
const currentArtist = document.getElementById('current-artist');
const pickButton = document.getElementById('pick');
const previousButton = document.getElementById('previous');
const nextButton = document.getElementById('next');
const modeButton = document.getElementById('mode');

const playModes = ['顺序播放', '单曲循环', '随机播放'];

let songs = [];
let currentIndex = -1;
let currentFolder = '';
let currentModeIndex = 0;
let isRestoring = false;

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

previousButton.addEventListener('click', () => {
  playByIndex(getPreviousIndex());
});

nextButton.addEventListener('click', () => {
  playByIndex(getNextIndex());
});

modeButton.addEventListener('click', () => {
  currentModeIndex = (currentModeIndex + 1) % playModes.length;
  modeButton.textContent = playModes[currentModeIndex];
});

player.addEventListener('ended', () => {
  if (songs.length === 0) return;

  if (playModes[currentModeIndex] === '单曲循环') {
    player.currentTime = 0;
    player.play();
    return;
  }

  playByIndex(getNextIndex());
});

startApp();

async function startApp() {
  try {
    const config = await LoadConfig();
    if (!config || !config.lastFolder) {
      info.textContent = '请选择音乐文件夹';
      return;
    }

    isRestoring = true;
    await loadFolder(config.lastFolder, config.lastSong || '');
    isRestoring = false;
  } catch (err) {
    isRestoring = false;
    showError(err);
  }
}

async function loadFolder(dir, songPathToSelect) {
  currentFolder = dir;
  info.textContent = '正在扫描：' + dir;

  songs = await ScanMusic(dir);
  currentIndex = -1;
  renderSongList();

  info.textContent = `共找到 ${songs.length} 首歌`;

  if (songPathToSelect) {
    const savedIndex = songs.findIndex((song) => song.path === songPathToSelect);
    if (savedIndex !== -1) {
      selectSong(savedIndex, false);
      info.textContent = `已恢复上次播放：${displayName(songs[savedIndex])}`;
    }
  }

  await saveCurrentState();
}

function renderSongList() {
  list.innerHTML = '';

  songs.forEach((song, index) => {
    const li = document.createElement('li');
    li.className = 'song-item';
    li.title = song.path;
    li.innerHTML = `
      <span class="song-title">${escapeHtml(displayName(song))}</span>
      <span class="song-artist">${escapeHtml(song.artist || '未知歌手')}</span>
    `;
    li.addEventListener('click', () => {
      playByIndex(index);
    });
    list.appendChild(li);
  });
}

function playByIndex(index) {
  selectSong(index, true);
}

async function selectSong(index, shouldPlay) {
  if (index < 0 || index >= songs.length) return;

  currentIndex = index;
  const song = songs[currentIndex];

  player.src = '/music?path=' + encodeURIComponent(song.path);
  updateCurrentSongView(song);
  updateActiveSong();
  await saveCurrentState();

  if (shouldPlay) {
    player.play();
    info.textContent = '正在播放：' + displayName(song);
  }
}

function updateCurrentSongView(song) {
  currentTitle.textContent = displayName(song);
  currentArtist.textContent = song.artist || '未知歌手';
  cover.hidden = false;
  cover.src = '/cover?path=' + encodeURIComponent(song.path) + '&t=' + Date.now();
  cover.onerror = () => {
    cover.hidden = true;
  };
}

function updateActiveSong() {
  const items = list.querySelectorAll('.song-item');

  items.forEach((item, index) => {
    item.classList.toggle('active', index === currentIndex);
  });

  const activeItem = items[currentIndex];
  if (activeItem) {
    activeItem.scrollIntoView({ block: 'nearest' });
  }
}

function getNextIndex() {
  if (songs.length === 0) return -1;

  if (playModes[currentModeIndex] === '随机播放') {
    return getRandomIndex();
  }

  return (currentIndex + 1) % songs.length;
}

function getPreviousIndex() {
  if (songs.length === 0) return -1;

  if (playModes[currentModeIndex] === '随机播放') {
    return getRandomIndex();
  }

  return (currentIndex - 1 + songs.length) % songs.length;
}

function getRandomIndex() {
  if (songs.length <= 1) return 0;

  let randomIndex = currentIndex;
  while (randomIndex === currentIndex) {
    randomIndex = Math.floor(Math.random() * songs.length);
  }
  return randomIndex;
}

async function saveCurrentState() {
  if (isRestoring) return;

  await SaveConfig({
    lastFolder: currentFolder,
    lastSong: currentIndex >= 0 ? songs[currentIndex].path : '',
  });
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
