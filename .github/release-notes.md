## 极拍 SwiftPaw 1.0.0

第一个正式版本，一个简洁的 Windows 本地音乐播放器。

### 下载

- `SwiftPaw-1.0.0-win-x64-setup.exe`：安装程序，装好后开始菜单和桌面都有快捷方式
- `SwiftPaw-1.0.0-win-x64.zip`：免安装版，解压后直接运行 `SwiftPaw.exe`

需要 Windows 10 或 11（64 位）。程序依赖系统自带的 WebView2，Windows 11 和大多数 Windows 10 已经装好了；如果打开时提示缺少 WebView2，按提示安装即可。

### 主要功能

- 扫描本地文件夹，播放 MP3、FLAC、WAV、OGG、M4A 等格式，显示内嵌封面
- 播放队列，顺序 / 单曲循环 / 随机三种模式
- 我喜欢、自建歌单、最近播放，保存在本地 SQLite 数据库
- LRC 歌词（同名 .lrc 文件或内嵌歌词，自动识别 GBK 编码），当前行高亮，点一行跳过去
- 深色 / 浅色 / 跟随系统主题
- 插件：装上插件就能在线搜索、播放、收藏歌曲。仓库里的 `examples/plugins/archive-org.js` 是一个示例插件，可以搜索 Internet Archive 上的公有领域音频
- 支持键盘媒体键和 Windows 的系统媒体控制，记住窗口大小

完整的改动记录见 [CHANGELOG.md](https://github.com/dongzhongcen/SwiftPaw/blob/main/CHANGELOG.md)。
