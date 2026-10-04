// 歌词页：左边大封面和歌曲信息，右边滚动的歌词。
// 当前唱到的那一行高亮并自动滚到中间；点某一行可以跳到那个时间。

import { GetLyrics } from '../../wailsjs/go/main/App';
import { audio, coverUrl } from './player.js';
import { registerView } from './views.js';
import { state, on, currentSong } from './state.js';
import { escapeHtml, displayName, songKey, isOnline } from './util.js';

const sourceNames = { lrc: '同名 LRC 文件', embedded: '音频文件内嵌', plugin: '插件' };

const cache = new Map(); // 歌曲 key -> 歌词，切回同一首歌时不用重新读
let lyrics = null; // 当前歌曲的歌词，null 表示还在加载
let lyricsKey = ''; // lyrics 属于哪首歌
let activeIndex = -1;
let userScrollUntil = 0; // 用户手动滚动后的几秒内不自动滚动

export function initLyrics() {
  registerView('lyrics', loadView);
  on('song-changed', (song) => loadLyrics(song));
  audio().addEventListener('timeupdate', () => highlight(false));
  audio().addEventListener('seeked', () => highlight(true));
}

async function loadLyrics(song) {
  const key = song ? songKey(song) : '';
  lyricsKey = key;
  activeIndex = -1;
  if (!song) {
    lyrics = { lines: [], plain: false, source: '' };
  } else if (cache.has(key)) {
    lyrics = cache.get(key);
  } else {
    lyrics = null;
    rerenderIfVisible();
    let result;
    try {
      result = await GetLyrics(song);
    } catch (err) {
      result = { lines: [], plain: false, source: '', error: String(err) };
    }
    result.lines ||= [];
    cache.set(key, result);
    if (lyricsKey !== key) return; // 加载期间又切歌了
    lyrics = result;
  }
  rerenderIfVisible();
}

function rerenderIfVisible() {
  if (state.view !== 'lyrics') return;
  const body = document.getElementById('view-body');
  renderLyrics(body);
  const song = currentSong();
  document.getElementById('view-subtitle').textContent = song ? subtitleOf(song) : '';
}

function subtitleOf(song) {
  return [displayName(song), song.artist].filter(Boolean).join(' - ');
}

function loadView() {
  const song = currentSong();
  return {
    title: '歌词',
    subtitle: song ? subtitleOf(song) : '还没有播放歌曲',
    actions: [],
    render: renderLyrics,
  };
}

function renderLyrics(body) {
  const song = currentSong();
  activeIndex = -1;
  let linesHtml;
  if (!song) {
    linesHtml = '<p class="lyrics-tip">还没有播放歌曲</p>';
  } else if (!lyrics) {
    linesHtml = '<p class="lyrics-tip">正在加载歌词…</p>';
  } else if (lyrics.lines.length === 0) {
    const tip = isOnline(song)
      ? `「${song.source}」插件没有提供这首歌的歌词`
      : '把同名的 .lrc 文件放在歌曲旁边就能显示，比如 晴天.mp3 和 晴天.lrc';
    linesHtml = `<p class="lyrics-tip">暂无歌词</p>
      <p class="lyrics-tip small">${escapeHtml(tip)}</p>`;
  } else {
    linesHtml = lyrics.lines
      .map((line, i) => `<p class="lyric-line" data-index="${i}">${escapeHtml(line.text) || '&nbsp;'}</p>`)
      .join('');
  }

  const source = lyrics?.source ? `歌词来源：${sourceNames[lyrics.source] || lyrics.source}` : '';
  body.innerHTML = `
    <div class="lyrics-view ${lyrics?.plain ? 'plain' : ''}">
      <div class="lyrics-side">
        <div class="lyrics-cover">
          ${song && coverUrl(song) ? `<img src="${escapeHtml(coverUrl(song))}" alt="" onerror="this.hidden=true" />` : ''}
        </div>
        <h2 class="lyrics-title">${escapeHtml(song ? displayName(song) : '')}</h2>
        <p class="lyrics-meta">${escapeHtml(song ? [song.artist, song.album].filter(Boolean).join(' · ') : '')}</p>
        <p class="lyrics-source">${escapeHtml(source)}${lyrics?.plain ? ' · 没有时间轴' : ''}</p>
      </div>
      <div class="lyrics-scroll" id="lyrics-scroll">
        <div class="lyrics-pad"></div>
        ${linesHtml}
        <div class="lyrics-pad"></div>
      </div>
    </div>`;

  const scroller = body.querySelector('#lyrics-scroll');
  scroller.addEventListener('wheel', () => {
    userScrollUntil = Date.now() + 3000;
  });
  // 点击一行跳到这一行的时间（纯文本歌词没有时间，不处理）
  scroller.addEventListener('click', (event) => {
    const line = event.target.closest('.lyric-line');
    if (!line || !lyrics || lyrics.plain) return;
    const player = audio();
    player.currentTime = lyrics.lines[Number(line.dataset.index)].time;
    if (player.paused) player.play().catch(() => {});
    userScrollUntil = 0;
  });
  highlight(true);
}

// currentLineIndex 用二分查找找到“时间不超过当前播放时间”的最后一行
export function currentLineIndex(lines, time) {
  let lo = 0;
  let hi = lines.length - 1;
  let found = -1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if (lines[mid].time <= time) {
      found = mid;
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }
  return found;
}

function highlight(force) {
  if (state.view !== 'lyrics' || !lyrics || lyrics.plain || lyrics.lines.length === 0) return;
  const scroller = document.getElementById('lyrics-scroll');
  if (!scroller) return;
  // 提前 0.2 秒切换，看起来和歌声更同步
  const index = currentLineIndex(lyrics.lines, audio().currentTime + 0.2);
  if (index === activeIndex && !force) return;

  scroller.querySelector('.lyric-line.active')?.classList.remove('active');
  activeIndex = index;
  const line = scroller.querySelector(`.lyric-line[data-index="${index}"]`);
  if (!line) return;
  line.classList.add('active');
  if (Date.now() < userScrollUntil) return;
  const top = line.offsetTop - scroller.clientHeight / 2 + line.offsetHeight / 2;
  scroller.scrollTo({ top, behavior: force ? 'auto' : 'smooth' });
}
