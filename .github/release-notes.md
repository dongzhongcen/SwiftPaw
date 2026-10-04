## 极拍 SwiftPaw 1.1.0

一个简洁的本地音乐播放器，现在有 Windows 版和 Android 版。

### 这个版本的改动

- **新增 Android 版**（Android 8.0 及以上）：和桌面版同一套界面和功能，手机和平板都有专门的布局
  - 扫描手机里的音乐，后台播放，通知栏和锁屏上可以控制播放、切歌、拖进度，耳机和蓝牙按键也能用
  - 插件、在线搜索、歌单、收藏、歌词都能用
- **新增示例插件 Jamendo**：`examples/plugins/jamendo.js` 通过 Jamendo 官方 API 搜索和播放以知识共享许可证发布的音乐，需要在插件的「设置」里填写自己申请的 Client ID

### 下载

- `SwiftPaw-1.1.0-win-x64-setup.exe`：Windows 安装程序，装好后开始菜单和桌面都有快捷方式
- `SwiftPaw-1.1.0-win-x64.zip`：Windows 免安装版，解压后直接运行 `SwiftPaw.exe`
- `SwiftPaw-1.1.0-android.apk`：Android 安装包

Windows：从旧版本升级时直接运行新的安装程序即可。需要 Windows 10 或 11（64 位）。程序依赖系统自带的 WebView2，Windows 11 和大多数 Windows 10 已经装好了；如果打开时提示缺少 WebView2，按提示安装即可。

Android：在手机上打开 APK 安装。第一次安装时系统会问是否允许“安装未知应用”，允许用来打开 APK 的应用（浏览器或文件管理器）就行。以后升级直接装新的 APK，数据会保留。

### 主要功能

- 扫描本地文件夹（Android 版是整个手机），播放 MP3、FLAC、WAV、OGG、M4A 等格式，显示内嵌封面
- 播放队列，顺序 / 单曲循环 / 随机三种模式
- 我喜欢、自建歌单、最近播放，保存在本地 SQLite 数据库
- LRC 歌词（同名 .lrc 文件或内嵌歌词，自动识别 GBK 编码），当前行高亮，点一行跳过去。Android 11 及以上只能显示内嵌歌词
- 深色 / 浅色 / 跟随系统主题
- 插件：装上插件就能在线搜索、播放、收藏歌曲，插件可以有自己的设置。仓库里有两个示例插件：`examples/plugins/archive-org.js` 搜索 Internet Archive 上的公有领域音频，`examples/plugins/jamendo.js` 搜索 Jamendo 上以知识共享许可证发布的音乐
- Windows：支持键盘媒体键和系统媒体控制，记住窗口大小；Android：后台播放和系统媒体通知

完整的改动记录见 [CHANGELOG.md](https://github.com/dongzhongcen/SwiftPaw/blob/main/CHANGELOG.md)。

### 免责声明

本软件不提供任何音乐源，也不内置任何第三方平台插件；插件由用户自行安装，其内容及合法性由用户和插件作者负责；请遵守当地法律及各平台服务条款，仅用于个人学习和合法用途。
