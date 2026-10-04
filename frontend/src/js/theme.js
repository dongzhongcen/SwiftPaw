// 主题：深色 / 浅色 / 跟随系统。
// 做法是在 <html> 上设置 data-theme="dark" 或 "light"，CSS 里不同主题用不同的颜色变量。

export const themes = [
  { value: 'dark', label: '深色' },
  { value: 'light', label: '浅色' },
  { value: 'system', label: '跟随系统' },
];

const systemDark = window.matchMedia('(prefers-color-scheme: dark)');
let preference = 'dark';

// applyTheme 应用用户选的主题；选“跟随系统”时，系统切换深浅色会自动跟着变
export function applyTheme(value) {
  preference = themes.some((t) => t.value === value) ? value : 'dark';
  const resolved = preference === 'system' ? (systemDark.matches ? 'dark' : 'light') : preference;
  document.documentElement.dataset.theme = resolved;
  setWindowTheme();
}

export function currentTheme() {
  return preference;
}

systemDark.addEventListener('change', () => {
  if (preference === 'system') applyTheme('system');
});

// Windows 上顺便让窗口标题栏也跟着变色（在浏览器里调试时没有 window.runtime，直接跳过）
function setWindowTheme() {
  try {
    const rt = window.runtime;
    if (!rt) return;
    if (preference === 'dark') rt.WindowSetDarkTheme();
    else if (preference === 'light') rt.WindowSetLightTheme();
    else rt.WindowSetSystemDefaultTheme();
  } catch {
    // 其他平台没有这些方法，忽略
  }
}
