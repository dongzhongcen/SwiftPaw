// 一些各个模块都会用到的小工具函数

// escapeHtml 把文字里的特殊符号转义，防止歌名里有 < > 时把页面结构弄乱
export function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}

// displayName 是界面上显示的歌名
export function displayName(song) {
  return song?.title || song?.name || '未知歌曲';
}

// songKey 是一首歌的唯一标识，和 Go 里 store.Key 的规则一致：本地歌曲就是文件路径
export function songKey(song) {
  return song?.path || '';
}

// formatTime 把秒数变成 3:07 这样的格式
export function formatTime(seconds) {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00';
  const s = Math.floor(seconds);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

// $ 是 document.getElementById 的简写
export const $ = (id) => document.getElementById(id);

// errorText 把 Go 返回的错误（通常是字符串）变成能显示的文字
export function errorText(err) {
  if (!err) return '未知错误';
  if (typeof err === 'string') return err;
  return err.message || String(err);
}

let toastTimer = 0;

// toast 在右下角显示一条提示，几秒后自动消失
export function toast(message, isError = false) {
  const box = $('toast');
  box.textContent = message;
  box.classList.toggle('error', isError);
  box.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => box.classList.remove('show'), isError ? 4500 : 2500);
}

export function showError(err) {
  toast('出错了：' + errorText(err), true);
}

// guard 包一层 try/catch：按钮点击之类的异步操作出错时弹出提示，而不是悄悄失败
export function guard(fn) {
  return async (...args) => {
    try {
      return await fn(...args);
    } catch (err) {
      showError(err);
    }
  };
}
