// 视觉效果：“减少模糊效果”开关和自定义背景图片。
// 背景图片由 Go 复制到数据文件夹（见 internal/core/background.go），前端用 /background?name=... 显示：
// 桌面版由 Go 的资源服务提供，Android 版由 WebView 拦截后交给 Go 内核找到文件。

import { isAndroid } from './platform.js';

const root = document.documentElement;

// applyReduceBlur：value 是配置里的 reduceBlur，空字符串表示自动（Android 上默认减少，手机上模糊比较费电、可能卡）
export function applyReduceBlur(value) {
  root.dataset.reduceBlur = String(reduceBlurEnabled(value));
}

export function reduceBlurEnabled(value) {
  return value === 'on' || (!value && isAndroid);
}

// backgroundUrl 返回背景图片的地址
export function backgroundUrl(image) {
  return image ? `/background?name=${encodeURIComponent(image)}` : '';
}

let loading = 0; // 连着换图片时，只认最后一次

// applyBackground 显示背景图片和它的模糊、遮罩设置；图片读不出来（比如文件被删了）时不显示背景
export function applyBackground(background = {}) {
  const { image = '', blur = 0, dim = 40 } = background;
  root.style.setProperty('--bg-blur', `${blur}px`);
  root.style.setProperty('--bg-dim', String(dim / 100));
  const token = ++loading;
  if (!image) {
    delete root.dataset.bg;
    root.style.removeProperty('--bg-image');
    return;
  }
  const url = backgroundUrl(image);
  const img = new Image();
  img.onload = () => {
    if (token !== loading) return;
    root.style.setProperty('--bg-image', `url("${url}")`);
    root.dataset.bg = 'on';
  };
  img.onerror = () => {
    if (token !== loading) return;
    delete root.dataset.bg;
    root.style.removeProperty('--bg-image');
  };
  img.src = url;
}
