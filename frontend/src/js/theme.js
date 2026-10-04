// 主题：配色（深色 / 浅色 / 跟随系统）加上可选的风格（唱片行 / 节拍器 / 霓虹）。
// 做法是在 <html> 上设置 data-theme：没选风格时是 "dark" 或 "light"，选了风格时是风格的名字，
// CSS 里不同主题用不同的颜色变量（css/theme.css 和 css/themes/*.css）。
// 另外 data-scheme 总是 "dark" 或 "light"，表示现在是深色还是浅色。

export const themes = [
  { value: 'dark', label: '深色' },
  { value: 'light', label: '浅色' },
  { value: 'system', label: '跟随系统' },
];

// 风格自带配色，选了风格时上面的配色不起作用。swatch 是设置页里预览小方块的颜色：[底色, 文字, 强调色]
export const styles = [
  { value: '', label: '极简', hint: '黑白，跟随配色' },
  { value: 'vinyl', label: '唱片行', hint: '暖色木纹和黑胶', scheme: 'dark', swatch: ['#241a14', '#f1e6d3', '#e9a23b'] },
  { value: 'metronome', label: '节拍器', hint: '明亮的节拍刻度', scheme: 'light', swatch: ['#fbfcfd', '#151a2d', '#2d46f5'] },
  { value: 'neon', label: '霓虹', hint: '夜色里的灯管', scheme: 'dark', swatch: ['#0b1426', '#eef3ff', '#ff4f9a'] },
];

const systemDark = window.matchMedia('(prefers-color-scheme: dark)');
let preference = 'dark';
let style = '';

// applyTheme 应用用户选的配色；选“跟随系统”时，系统切换深浅色会自动跟着变
export function applyTheme(value) {
  preference = themes.some((t) => t.value === value) ? value : 'dark';
  render();
}

// applyStyle 应用风格，空字符串表示不用风格
export function applyStyle(value) {
  style = styles.some((s) => s.value === value) ? value || '' : '';
  render();
}

export function currentTheme() {
  return preference;
}

export function currentStyle() {
  return style;
}

function render() {
  const scheme = currentScheme();
  document.documentElement.dataset.theme = style || scheme;
  document.documentElement.dataset.scheme = scheme;
  setWindowTheme();
}

// currentScheme 返回现在是深色还是浅色
function currentScheme() {
  if (style) return styles.find((s) => s.value === style).scheme;
  if (preference === 'system') return systemDark.matches ? 'dark' : 'light';
  return preference;
}

systemDark.addEventListener('change', () => {
  if (preference === 'system') render();
});

// Windows 上顺便让窗口标题栏也跟着变色（Android 上是状态栏的图标颜色；在浏览器里调试时没有 window.runtime，直接跳过）
function setWindowTheme() {
  try {
    const rt = window.runtime;
    if (!rt) return;
    if (!style && preference === 'system') rt.WindowSetSystemDefaultTheme();
    else if (currentScheme() === 'dark') rt.WindowSetDarkTheme();
    else rt.WindowSetLightTheme();
  } catch {
    // 其他平台没有这些方法，忽略
  }
}
