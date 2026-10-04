# 极拍 SwiftPaw

[![测试](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/test.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/test.yml)
[![构建 Windows 版](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml)
[![最新版本](https://img.shields.io/github/v/release/dongzhongcen/SwiftPaw?label=%E6%9C%80%E6%96%B0%E7%89%88%E6%9C%AC)](https://github.com/dongzhongcen/SwiftPaw/releases)
![Go](https://img.shields.io/github/go-mod/go-version/dongzhongcen/SwiftPaw)
![平台](https://img.shields.io/badge/%E5%B9%B3%E5%8F%B0-Windows%2010%20%2F%2011-0078D4)
![Wails](https://img.shields.io/badge/Wails-v2.16-red)

## 简介

极拍（SwiftPaw）是我用 Go 和 [Wails](https://wails.io) 写的一个 Windows 桌面音乐播放器。

我想做的播放器很简单：打开就能听本地的歌，界面干净，不需要登录。所以它的界面是黑白极简风格，所有数据（歌单、收藏、播放记录）都只存在自己电脑上。

后来我又给它加了插件系统：插件是一个小小的 JS 文件，告诉播放器去哪里搜歌、怎么拿到播放地址。这样播放器本身不绑定任何音乐平台，想听什么由插件决定。仓库里带了一个示例插件，可以搜索 Internet Archive 上的公有领域音乐。

代码里的注释我都尽量写得适合初学者看，每个文件也尽量保持短小。

## 功能特性

**本地音乐**
- 选择一个文件夹，自动扫描里面（包括子文件夹）的 MP3、FLAC、M4A、AAC、OGG、Opus、WAV
- 读取歌曲标签里的标题、歌手、专辑和内嵌封面
- 按标题、歌手、专辑、文件名搜索当前列表

**播放**
- 播放队列：下一首播放、从队列移除；顺序、单曲循环、随机三种模式（随机模式一轮内不重复）
- 某首歌放不了时自动跳到下一首，整个队列都放不了时停下来
- 支持键盘上的媒体键，Windows 的系统媒体浮窗里也能看到歌名和封面、控制播放
- 记住音量、播放模式、上次播放的歌和窗口大小

**歌单和歌词**
- 我喜欢、自建歌单、最近播放，保存在本地的 SQLite 数据库
- LRC 歌词：同名 `.lrc` 文件或音频文件内嵌的歌词，自动识别 UTF-8 / GBK 编码；当前行高亮，点一行就跳到那里

**插件和在线音乐**
- 从文件或网址安装插件，可以启用、停用、卸载
- 用插件在线搜索，搜到的歌可以直接播放、收藏、加入歌单
- 插件在内置的 JS 引擎（[goja](https://github.com/dop251/goja)）里运行，不需要装 Node.js；常用的 `axios`、`crypto-js`、`cheerio`、`dayjs` 等模块已经内置

**外观**
- 深色 / 浅色 / 跟随系统三种主题

## 项目结构

```
SwiftPaw/
├── main.go               # 程序入口：创建窗口、绑定 App
├── app.go                # App：扫描文件夹、播放队列
├── app_store.go          # 歌单、收藏、最近播放
├── app_config.go         # 配置（config.json）
├── app_lyrics.go         # 歌词
├── app_plugins.go        # 插件、在线搜索、在线播放
├── window.go             # 记住窗口大小
├── version.go            # 版本号
├── handler.go            # /music、/cover、/stream 三个本地地址
├── internal/
│   ├── music/            # 歌曲信息、扫描文件夹
│   ├── queue/            # 播放队列和播放模式
│   ├── store/            # SQLite：歌单、收藏、最近播放
│   ├── lyrics/           # LRC 解析、编码识别、找歌词
│   ├── jsrt/             # 插件的 JS 运行环境（goja + 内置模块）
│   ├── plugin/           # 插件的安装、加载和调用
│   ├── stream/           # 在线歌曲的播放代理
│   └── winstate/         # 窗口大小的保存和读取
├── frontend/             # 前端（原生 JS + Vite，没有用框架）
│   ├── src/js/           # 每个功能一个小模块：播放器、列表、歌词、插件页……
│   ├── src/css/          # 样式，颜色都在 theme.css 里
│   └── wailsjs/          # Go 方法的 JS 绑定（自动生成）
├── examples/plugins/     # 示例插件
├── scripts/
│   ├── genbindings/      # 不装 Wails 也能生成前端绑定
│   └── jslib/            # 把插件用的 JS 库打包成一个文件
├── build/                # 图标、Windows 安装程序的配置
└── .github/workflows/    # 测试和 Windows 打包
```

## 快速开始

### 直接下载

到 [Releases](https://github.com/dongzhongcen/SwiftPaw/releases) 下载：

- `SwiftPaw-x.y.z-win-x64-setup.exe`：安装版
- `SwiftPaw-x.y.z-win-x64.zip`：免安装版，解压后运行 `SwiftPaw.exe`

还没有发布的最新代码，可以在 [Actions → 构建 Windows 版](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml) 里下载 `SwiftPaw-win-x64` 构建产物。

需要 Windows 10 / 11（64 位）和 WebView2（Windows 11 自带，Windows 10 大多也装好了）。

### 第一次使用

1. 点“选择文件夹”，选你放音乐的文件夹
2. 点一首歌开始播放；点歌曲后面的红心收藏，或者加入歌单
3. 想在线搜歌：打开“插件”页 → “从文件安装”，选择 [`examples/plugins/archive-org.js`](examples/plugins/archive-org.js)，然后到“在线搜索”页搜索

### 从源码运行

需要先装好：

- [Go](https://go.dev/dl/) 1.25 或更新
- [Node.js](https://nodejs.org/) 20 或更新
- Wails 命令行工具：`go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0`

```bash
git clone https://github.com/dongzhongcen/SwiftPaw.git
cd SwiftPaw
wails dev                                   # 开发模式，改了代码会自动刷新
wails build -platform windows/amd64         # 编译，结果在 build/bin/SwiftPaw.exe
wails build -platform windows/amd64 -nsis   # 同时生成安装程序（需要装 NSIS）
```

运行测试：

```bash
go test ./internal/...
cd frontend && npm ci && npm run build
```

### 写一个插件

插件是一个 CommonJS 格式的 `.js` 文件：

```js
const axios = require("axios");

module.exports = {
  platform: "我的音乐源",
  version: "1.0.0",
  async search(query, page) {
    // 返回 { isEnd: 是否没有下一页了, data: [{ id, title, artist, album, artwork, duration }] }
  },
  async getMediaSource(musicItem, quality) {
    // 返回 { url: 播放地址, headers: 需要的请求头（可选） }
  },
  async getLyric(musicItem) {
    // 返回 { rawLrc: LRC 格式的歌词 }，没有歌词就返回 null
  },
};
```

完整的例子见 [`examples/plugins/archive-org.js`](examples/plugins/archive-org.js)，格式说明在 [`internal/plugin/types.go`](internal/plugin/types.go) 开头的注释里。

## 当前状态

现在是 **1.0.0**，上面列的功能都已经完成，有单元测试和界面走查。改动记录见 [CHANGELOG.md](CHANGELOG.md)。

目前知道的不足：

- 只在 Windows 上测试过。代码里没有专门针对 Windows 的部分，理论上 macOS / Linux 也能编译，但我没有试
- 插件的 JS 环境不是完整的 Node.js：内置了常用的模块，用到其他模块（比如 `webdav`）的插件会提示“暂不支持”
- 在线歌曲依赖插件。插件被卸载或者失效后，已经收藏的在线歌曲还在列表里，但放不了
- 还没有自动更新，新版本需要自己去 Releases 下载

有问题或者建议，欢迎提 [Issue](https://github.com/dongzhongcen/SwiftPaw/issues)。
