// 窗口大小对应的几种布局。和 css/mobile.css 里的 @media 条件保持一致。
//
//   桌面布局：宽度 ≥ 860（Windows 版的最小窗口宽度），左边侧边栏、底部完整的播放栏
//   手机布局：宽度 < 860，或者横屏而且高度 ≤ 500（手机横着拿）
//     - 竖屏、宽度 < 720：标签栏在底部
//     - 其他情况（平板竖屏、手机横屏）：标签栏竖着放在左边
//     播放栏变成迷你播放条，点开是全屏的播放页

// mobileLayout 匹配时用手机布局
export const mobileLayout = window.matchMedia('(max-width: 859.98px), (max-height: 500px) and (orientation: landscape)');

// narrowScreen 匹配时歌曲列表放不下所有列，行尾按钮收进“更多”里
export const narrowScreen = window.matchMedia('(max-width: 719.98px)');
