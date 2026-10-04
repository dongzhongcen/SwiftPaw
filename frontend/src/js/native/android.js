// Android 版的桥。
// 桌面版里前端通过 Wails 生成的 window.go.main.App.方法名(...) 调用 Go；
// Android 版没有 Wails，Go 内核被编译成了 Android 库，由原生插件 SwiftPaw 转发调用。
// 这里造一个同样的 window.go.main.App，所以 wailsjs 里的函数和其他模块的代码都不用改。

let cap;

// nativeCall 调用原生插件 SwiftPaw 的一个方法
export function nativeCall(method, options = {}) {
  return cap.nativePromise('SwiftPaw', method, options);
}

// callCore 调用 Go 内核的一个方法，参数和返回值都用 JSON 传
export async function callCore(method, args) {
  const result = await nativeCall('call', { method, args: JSON.stringify(args) });
  return result.json ? JSON.parse(result.json) : null;
}

// onNative 订阅原生插件发来的事件（比如返回键）
export function onNative(eventName, callback) {
  cap.addListener('SwiftPaw', eventName, callback);
}

// Android 上没有“音乐文件夹”，歌曲库来自系统的媒体库（手机里所有的音乐）。
// 界面上需要一个文件夹名的地方显示这个名字，配置里的 lastFolder 也存它
export const DEVICE_LIBRARY = '本机音乐';

// 桌面版和 Android 版做法不一样的方法放在这里，其他方法直接交给 Go 内核
const overrides = {
  // “选择文件夹”换成扫描整个手机（第一次会请求读取音乐的权限）
  SelectFolder: async () => DEVICE_LIBRARY,
  ScanMusic: async () => {
    const result = await nativeCall('scanLibrary');
    return JSON.parse(result.json);
  },
  // “从文件安装插件”用系统的文件选择器；取消时和桌面版一样返回 id 为空的结果
  InstallPluginFromFile: async () => {
    if (!(await callCore('PluginsAvailable', []))) return callCore('InstallPluginFile', ['']);
    const result = await nativeCall('installPluginFile');
    return result.json ? JSON.parse(result.json) : { id: '' };
  },
  // 选背景图片用系统的文件选择器，原生代码把图片复制出来交给 Go 内核；取消时和桌面版一样返回空字符串
  SelectBackgroundImage: async () => {
    const result = await nativeCall('pickBackgroundImage');
    return result.json ? JSON.parse(result.json) : '';
  },
};

// overrideMethod 让某个方法在 Android 上换一种做法（比如选择文件夹要用系统的选择器）
export function overrideMethod(name, fn) {
  overrides[name] = fn;
}

export function installAndroidBridge(capacitor) {
  cap = capacitor;
  const App = new Proxy(
    {},
    {
      get(_, name) {
        if (typeof name !== 'string') return undefined;
        return overrides[name] || ((...args) => callCore(name, args));
      },
    },
  );
  window.go = { main: { App } };

  // Wails 的 window.runtime 里前端用到的几个方法
  window.runtime = {
    BrowserOpenURL: (url) => nativeCall('openUrl', { url }).catch(() => {}),
    // 状态栏和导航栏的图标颜色跟着主题走：DARK 是给深色背景用的浅色图标
    WindowSetDarkTheme: () => setBarsStyle('DARK'),
    WindowSetLightTheme: () => setBarsStyle('LIGHT'),
    WindowSetSystemDefaultTheme: () => setBarsStyle('DEFAULT'),
  };
}

function setBarsStyle(style) {
  cap.nativePromise('SystemBars', 'setStyle', { style }).catch(() => {});
}
