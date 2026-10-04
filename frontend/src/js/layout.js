// 整个窗口的 HTML 结构：左边侧边栏、中间内容区、底部播放栏

import { icons } from './icons.js';

export const layoutHtml = `
  <div class="app">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-logo">${icons.logo}</span>
        <span class="brand-name">极拍</span>
        <span class="brand-sub">SwiftPaw</span>
      </div>

      <nav class="nav" id="nav">
        <button class="nav-item" type="button" data-view="library">
          ${icons.music}<span class="nav-label">本地音乐</span><span class="nav-count" id="count-library"></span>
        </button>
        <button class="nav-item" type="button" data-view="playlist:1">
          ${icons.heart}<span class="nav-label">我喜欢</span><span class="nav-count" id="count-favorites"></span>
        </button>
        <button class="nav-item" type="button" data-view="recent">
          ${icons.clock}<span class="nav-label">最近播放</span>
        </button>
        <button class="nav-item" type="button" data-view="queue">
          ${icons.queue}<span class="nav-label">播放队列</span><span class="nav-count" id="count-queue">0</span>
        </button>
        <button class="nav-item" type="button" data-view="lyrics">
          ${icons.lyrics}<span class="nav-label">歌词</span>
        </button>
        <button class="nav-item" type="button" data-view="online">
          ${icons.globe}<span class="nav-label">在线搜索</span>
        </button>
        <button class="nav-item" type="button" data-view="plugins">
          ${icons.plugin}<span class="nav-label">插件</span>
        </button>
      </nav>

      <div class="nav-section">
        <span>我的歌单</span>
        <button class="icon-btn small" type="button" id="new-playlist" title="新建歌单" aria-label="新建歌单">${icons.plus}</button>
      </div>
      <nav class="nav playlist-nav" id="playlist-nav"></nav>

      <nav class="nav sidebar-footer">
        <button class="nav-item" type="button" data-view="settings">
          ${icons.settings}<span class="nav-label">设置</span>
        </button>
      </nav>
    </aside>

    <main class="content">
      <header class="view-header">
        <div class="view-heading">
          <h1 id="view-title" class="view-title"></h1>
          <p id="view-subtitle" class="view-subtitle"></p>
        </div>
        <div class="view-tools">
          <label class="search-box" id="search-box">
            ${icons.search}
            <input id="search" class="search-input" type="search" placeholder="搜索当前列表" title="按标题、歌手、专辑、文件名搜索，不区分大小写" autocomplete="off" />
          </label>
          <div class="view-actions" id="view-actions"></div>
        </div>
      </header>
      <div class="view-body" id="view-body"></div>
    </main>

    <footer class="player-bar">
      <div class="pb-track">
        <button class="pb-cover" type="button" id="cover-btn" title="歌词">
          <img id="cover" alt="" hidden />
          <span class="pb-cover-fallback">${icons.music}</span>
        </button>
        <div class="pb-meta">
          <div id="pb-title" class="pb-title">尚未选择歌曲</div>
          <div id="pb-artist" class="pb-artist">请选择一个音乐文件夹</div>
        </div>
        <button class="icon-btn" type="button" id="pb-fav" title="收藏" aria-label="收藏" disabled>${icons.heart}</button>
      </div>

      <div class="pb-center">
        <div class="pb-controls">
          <button class="icon-btn" type="button" id="mode" title="顺序播放" aria-label="播放模式">${icons.sequence}</button>
          <button class="icon-btn" type="button" id="prev" title="上一首" aria-label="上一首">${icons.prev}</button>
          <button class="icon-btn play-btn" type="button" id="play" title="播放" aria-label="播放">${icons.play}</button>
          <button class="icon-btn" type="button" id="next" title="下一首" aria-label="下一首">${icons.next}</button>
          <span class="pb-mode-label" id="mode-label">顺序播放</span>
        </div>
        <div class="pb-progress">
          <span id="time-current" class="pb-time">0:00</span>
          <input type="range" class="range" id="seek" min="0" max="1000" step="1" value="0" aria-label="播放进度" />
          <span id="time-total" class="pb-time">0:00</span>
        </div>
      </div>

      <div class="pb-right">
        <button class="icon-btn" type="button" id="mute" title="静音" aria-label="静音">${icons.volume}</button>
        <input type="range" class="range volume" id="volume" min="0" max="100" step="1" value="100" aria-label="音量" />
      </div>
    </footer>
  </div>
  <audio id="audio" preload="auto"></audio>
  <div id="toast" class="toast" role="status"></div>
`;
