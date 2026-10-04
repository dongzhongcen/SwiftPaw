// Android 上的“播放器”。桌面版用隐藏的 <audio> 播放；Android 版要在后台、锁屏时也能放，
// 并且要有系统的媒体通知，所以真正的播放器是原生的（PlaybackService 里的 ExoPlayer）。
// 这个类模仿 <audio> 里 player.js、lyrics.js 用到的那部分属性和事件，其他模块不用关心是哪一种。
//
// 和 <audio> 不一样的地方：
// - src 写成 'native:<歌曲 key>'，原生那边从 Go 的播放队列里找到这首歌再加载（在线歌曲由原生向插件要地址）
// - 一首放完自动接下一首、放不了时跳过、记“最近播放”都由原生做（界面在后台时也要能做），
//   做完后原生发 queueChanged 事件，前端照着刷新界面；所以这里不会发出 ended 和 error 事件

import { nativeCall, onNative } from './android.js';

export class NativeAudio extends EventTarget {
  constructor() {
    super();
    this.paused = true;
    this.duration = NaN;
    this.playbackRate = 1;
    this._src = '';
    this._key = '';
    this._position = 0; // 原生上次报告的位置（秒）
    this._stamp = 0; // 报告的时间，播放中按时间往后推算当前位置
    this._buffering = false;
    this._volume = 1;
    this._muted = false;
    this._timer = 0;

    onNative('playerState', (state) => this._onState(state));
  }

  get src() {
    return this._src;
  }

  set src(value) {
    this._src = String(value || '');
    const key = this._src.startsWith('native:') ? this._src.slice('native:'.length) : '';
    this._load(key);
  }

  removeAttribute(name) {
    if (name === 'src') this.src = '';
  }

  load() {
    // 设置 src 时已经加载了
  }

  get currentTime() {
    if (this.paused || this._buffering) return this._position;
    const elapsed = (performance.now() - this._stamp) / 1000;
    const time = this._position + elapsed * this.playbackRate;
    return Number.isFinite(this.duration) ? Math.min(time, this.duration) : time;
  }

  set currentTime(value) {
    const time = Math.max(0, Number(value) || 0);
    this._position = time;
    this._stamp = performance.now();
    nativeCall('playerSeek', { position: time }).catch(() => {});
    this._fire('timeupdate');
    this._fire('seeked');
  }

  get volume() {
    return this._volume;
  }

  set volume(value) {
    this._volume = Math.min(1, Math.max(0, Number(value)));
    this._sendVolume();
  }

  get muted() {
    return this._muted;
  }

  set muted(value) {
    this._muted = !!value;
    this._sendVolume();
  }

  play() {
    if (!this._key) return Promise.resolve();
    this._setPaused(false);
    return nativeCall('playerPlay').then(() => undefined);
  }

  pause() {
    this._setPaused(true);
    nativeCall('playerPause').catch(() => {});
  }

  // ---- 内部 ----

  _load(key) {
    if (key === this._key) return;
    this._key = key;
    this._position = 0;
    this._stamp = performance.now();
    this.duration = NaN;
    this._fire('durationchange');
    this._fire('timeupdate');
    if (!key) this._setPaused(true);
    nativeCall('playerLoad', { key, play: false }).catch(() => {});
  }

  _sendVolume() {
    nativeCall('playerSetVolume', { volume: this._muted ? 0 : this._volume }).catch(() => {});
    this._fire('volumechange');
  }

  _onState(state) {
    // 加载新歌的过程中，原生可能还在报告上一首的状态，忽略
    if ((state.key || '') !== this._key) return;
    this._position = Number(state.position) || 0;
    this._stamp = performance.now();
    this._buffering = !!state.buffering;
    const duration = Number(state.duration) > 0 ? Number(state.duration) : NaN;
    if (duration !== this.duration && !(Number.isNaN(duration) && Number.isNaN(this.duration))) {
      this.duration = duration;
      this._fire('durationchange');
    }
    this._setPaused(!state.playing);
    if (state.playing && !state.buffering) this._fire('playing');
    this._fire('timeupdate');
  }

  _setPaused(paused) {
    if (paused === this.paused) return;
    this.paused = paused;
    this._fire(paused ? 'pause' : 'play');
    clearInterval(this._timer);
    // 播放中每 250 毫秒发一次 timeupdate（和 <audio> 差不多），进度条和歌词靠它走
    if (!paused) this._timer = setInterval(() => this._fire('timeupdate'), 250);
  }

  _fire(name) {
    this.dispatchEvent(new Event(name));
  }
}
