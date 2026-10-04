# 更新日志

这里记录极拍 SwiftPaw 每个版本的改动。版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.1.0] - 未发布

### 新增

- **Android 版（开发中）**：代码在 `mobile/`，要求 Android 8.0 及以上
  - 用 Capacitor 打包，界面和桌面版用同一套前端
  - Go 内核（播放队列、歌单、歌词、插件等）用 gomobile 编译成 `.aar`，由 Capacitor 插件调用
  - GitHub Actions 新增“构建 Android 版”：推送到 main 和发 PR 时编译安装包并上传；推送 `v` 开头的标签时，签名的 `SwiftPaw-x.y.z-android.apk` 和 Windows 版的安装包放进同一个 Release
  - Android 版本号 1.1.0（versionCode 10100）
- **手机和平板布局**：窗口宽度小于 860 时（或者手机横屏）自动换成手机布局，桌面版的界面不变
  - 竖屏手机：底部标签栏（音乐、我的、在线、插件、设置），标签栏上面是迷你播放条
  - 平板和手机横屏：标签栏放在左边
  - 新增“我的”页面，放我喜欢、最近播放、播放队列、歌词和自己建的歌单；子页面左上角有返回按钮
  - 点迷你播放条打开全屏播放页（大封面、进度条、播放模式、上一首/下一首），往下滑或点左上角收起
  - 屏幕窄时歌曲列表改成两行（标题 / 歌手），“下一首播放”“加入歌单”等收进每行的“更多”菜单
  - 触摸屏上按钮更大，行尾按钮一直显示，去掉了点过之后一直停在悬停状态的效果；避开刘海、状态栏和手势条
  - Android 的返回键依次关闭对话框、收起播放页、回到上一级页面，在“本地音乐”再按退到后台（不关掉应用）
- **示例插件 Jamendo**：`examples/plugins/jamendo.js` 通过 Jamendo 官方 API v3.0 搜索歌曲、播放（无损音质只在音乐人允许下载时提供）和获取歌词；需要在插件的「设置」里填写自己在 devportal.jamendo.com 申请的 Client ID，没填或填错时给出中文提示。Jamendo 的音乐以知识共享许可证发布，使用时请遵守 Jamendo 的 API 使用条款

### 改动

- 桌面版和 Android 版共用的功能从 `main` 包挪到 `internal/core`，桌面版的 `App` 只负责转发，以及弹出选择文件的窗口、记住窗口大小这些和窗口有关的事情；功能和数据保存的位置都不变
- `THIRD_PARTY_NOTICES.md` 增加 Android 版用到的 Go 模块、Capacitor、AndroidX 等依赖
- `.gitignore` 忽略 `*.jks`、`*.keystore` 签名密钥文件，防止误提交
- 依赖更新：golang.org/x/text 0.41.0、golang.org/x/net 0.58.0、golang.org/x/crypto 0.55.0（随 golang.org/x/mobile 一起升级）

## [1.0.1] - 2026-10-04

### 新增

- **插件设置（用户变量）**：插件可以在导出对象里用 `userVariables: [{ key, name, hint }]` 声明需要用户填写的设置（比如 API Key），`name` 和 `hint` 可以不写
  - 插件页里声明了设置的插件会多一个“设置”按钮，打开对话框填写，保存后插件马上重新加载，不用重启
  - 设置保存在本地的 `plugins.json`，插件通过 `env.getUserVariables()` 读取；更新、重新加载插件时保留，卸载插件时一起删除
- **安装程序检查旧版本**：安装前先看有没有装过极拍
  - 有旧版本时询问是否替换；相同版本询问是否重新安装；已安装的版本更新时提醒会降级
  - 同意后先静默卸载旧版本，再装到原来的文件夹；歌单、收藏、插件和插件设置（在 `%AppData%\SwiftPaw`）都会保留
  - 极拍正在运行时，提示先关闭再继续
- **许可证**：项目使用 MIT 许可证（`LICENSE`）；`THIRD_PARTY_NOTICES.md` 改为由 `scripts/notices` 生成，列出编译进程序的 Go 模块、插件运行时内置的 JS 库和字体的许可证；压缩包和安装程序里都带上 `LICENSE`、`THIRD_PARTY_NOTICES.md` 和 `OFL.txt`
- **免责声明**：README、设置页“关于”和“从网址安装插件”的确认框里都加了免责声明

### 改动

- 测试数据和注释里的歌名、歌手、歌词换成自己编的中性文字
- 删除没用到的图片 `frontend/src/assets/images/logo-universal.png`

## [1.0.0] - 2026-10-04

第一个正式版本。

### 新增

- **发布**
  - GitHub Actions 自动打包 Windows 版（exe 压缩包和 NSIS 安装程序）
  - 推送 `v` 开头的标签时自动创建 Release
- **系统媒体控制**：支持键盘媒体键；Windows 的系统媒体浮窗显示歌名、歌手和封面，也可以控制播放、暂停、切歌、拖动进度
- **记住窗口大小**：保存在 `window.json`，最大化时不保存
- **关于**：设置页显示版本号和项目主页
- **插件系统**（#5）
  - 插件是 JS 文件，在内置的 goja 引擎里运行，内置 `axios`、`crypto-js`、`qs`、`dayjs`、`he`、`big-integer`、`cheerio` 等模块
  - 可以从文件或网址安装插件，也可以启用、停用、卸载
  - 在线搜索页：用插件搜歌，搜到的歌可以播放、收藏、加入歌单
  - 在线歌曲通过本地代理播放，可以带上插件要求的请求头，支持拖动进度
  - 示例插件 `examples/plugins/archive-org.js`，搜索 Internet Archive 上的公有领域音频
- **歌词、搜索和主题**（#4）
  - LRC 歌词：同名 `.lrc` 文件或内嵌歌词，自动识别 UTF-8 / UTF-16 / GBK 编码；当前行高亮，点一行跳过去
  - 搜索当前列表（标题、歌手、专辑、文件名）
  - 深色 / 浅色 / 跟随系统主题
- **歌单**（#2）
  - 我喜欢、自建歌单、最近播放，保存在本地 SQLite 数据库
  - 界面改为侧边栏 + 底部播放栏
- **格式和播放队列**（#1）
  - 支持 FLAC、M4A、AAC、OGG、Opus、WAV
  - 播放队列：下一首播放、移除；顺序、单曲循环、随机（一轮内不重复）

[1.0.1]: https://github.com/dongzhongcen/SwiftPaw/releases/tag/v1.0.1
[1.0.0]: https://github.com/dongzhongcen/SwiftPaw/releases/tag/v1.0.0
