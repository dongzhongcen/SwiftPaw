// 判断前端运行在哪个平台上：Windows 桌面版（Wails）还是 Android 版（Capacitor）。
// 这个文件要在 main.js 里最先导入：Android 上要先把 window.go 和 window.runtime 准备好，
// 其他模块调用 wailsjs 里的函数时才能找到它们。

import { installAndroidBridge } from './native/android.js';

const capacitor = window.Capacitor;

// isAndroid 为 true 表示运行在 Android 版里
export const isAndroid = !!capacitor?.isNativePlatform?.() && capacitor.getPlatform?.() === 'android';

// <html data-platform="android">，CSS 里个别地方会用到
document.documentElement.dataset.platform = isAndroid ? 'android' : 'desktop';

if (isAndroid) installAndroidBridge(capacitor);
