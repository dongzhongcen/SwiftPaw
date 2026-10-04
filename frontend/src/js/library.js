// 本地音乐：选择文件夹、扫描、恢复上次播放的歌

import { PlayLibrary, ScanMusic, SelectFolder } from '../../wailsjs/go/main/App';
import { applyQueue } from './player.js';
import { saveConfig } from './config.js';
import { state, emit } from './state.js';
import { displayName, toast } from './util.js';

export async function pickFolder() {
  const dir = await SelectFolder();
  if (!dir) return;
  await loadFolder(dir, '');
}

export async function rescan() {
  if (state.folder) await loadFolder(state.folder, '');
}

// loadFolder 扫描文件夹；songPathToSelect 不为空时把那首歌放进队列（不自动播放）
export async function loadFolder(dir, songPathToSelect) {
  state.folder = dir;
  toast('正在扫描：' + dir);

  state.library = (await ScanMusic(dir)) || [];
  emit('library-changed');
  toast(`共找到 ${state.library.length} 首歌`);

  if (songPathToSelect) {
    const index = state.library.findIndex((song) => song.path === songPathToSelect);
    if (index !== -1) {
      await applyQueue(await PlayLibrary(index), false);
      toast(`已恢复上次播放：${displayName(state.library[index])}`);
    }
  }
  saveConfig({ lastFolder: dir });
}
