// 配置的读取和保存（上次的文件夹、歌曲、播放模式、音量……）。
// 内存里保留一份完整的配置，每次只改其中几个字段再整体保存，免得不同模块互相覆盖。

import { LoadConfig, SaveConfig } from '../../wailsjs/go/main/App';

let config = {};
let restoring = true; // 启动恢复期间不保存，避免把还没恢复完的状态写回去
let timer = 0;

export async function loadConfig() {
  config = (await LoadConfig()) || {};
  return config;
}

export function getConfig() {
  return config;
}

// saveConfig 合并要修改的字段，稍等一下再写文件（连续修改只写一次）
export function saveConfig(patch) {
  Object.assign(config, patch);
  if (restoring) return;
  clearTimeout(timer);
  timer = setTimeout(() => SaveConfig(config).catch(() => {}), 300);
}

// finishRestoring 在启动恢复完成后调用，之后的修改才会真正保存
export function finishRestoring() {
  restoring = false;
  saveConfig({});
}
