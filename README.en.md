# 极拍 SwiftPaw

[简体中文](README.md) | **English**

[![Tests](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/test.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/test.yml)
[![Build Windows](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml)
[![Latest release](https://img.shields.io/github/v/release/dongzhongcen/SwiftPaw?label=latest%20release)](https://github.com/dongzhongcen/SwiftPaw/releases)
![Go](https://img.shields.io/github/go-mod/go-version/dongzhongcen/SwiftPaw)
![Platform](https://img.shields.io/badge/platform-Windows%2010%20%2F%2011-0078D4)
![Wails](https://img.shields.io/badge/Wails-v2.16-red)

![SwiftPaw main window (dark theme)](docs/screenshots/dark.png)

## Introduction

SwiftPaw (极拍, "Jípāi") is a Windows desktop music player I wrote in Go with [Wails](https://wails.io).

What I wanted was a simple player: open it and listen to the music on my own computer, with a clean interface and no sign-in. So the interface is minimal black and white, and all data (playlists, favorites, play history) stays on your own computer.

Later I added a plugin system: a plugin is a small JS file that tells the player where to search for songs and how to get a playback URL. That way the player itself isn't tied to any music platform; the plugins decide what you can listen to. The repository comes with an example plugin that searches public-domain music on the Internet Archive.

I've tried to write the code comments so that beginners can follow them, and to keep each file short.

> The app's interface is currently in Chinese only. Below, button and page names are given in English with the Chinese label you'll see in the app in parentheses.

## Features

**Local music**
- Pick a folder and it scans it (including subfolders) for MP3, FLAC, M4A, AAC, OGG, Opus and WAV files
- Reads title, artist, album and embedded cover art from the song tags
- Search the current list by title, artist, album or file name

**Playback**
- Play queue: play next, remove from queue; three modes: in order, repeat one, and shuffle (no repeats within a round)
- If a song can't be played, it automatically skips to the next one, and stops if nothing in the queue can be played
- Supports the media keys on your keyboard; the song title and cover also show up in the Windows media flyout, where you can control playback
- Remembers volume, play mode, the last song played and the window size

**Playlists and lyrics**
- Favorites, your own playlists and recently played, stored in a local SQLite database
- LRC lyrics: from a `.lrc` file with the same name or embedded in the audio file, with automatic UTF-8 / GBK detection; the current line is highlighted, and clicking a line jumps there

**Plugins and online music**
- Install plugins from a file or a URL; enable, disable or uninstall them
- Search online through plugins; songs you find can be played right away, added to favorites or added to playlists
- Plugins run in a built-in JS engine ([goja](https://github.com/dop251/goja)), so you don't need Node.js; common modules such as `axios`, `crypto-js`, `cheerio` and `dayjs` are built in

**Appearance**
- Three themes: dark, light, and follow the system

## Screenshots

The local songs in the screenshots are test audio I generated myself, and I wrote the lyrics just for this; the online search uses the example plugin from this repository.

| Light theme | Lyrics | Online search |
| :---: | :---: | :---: |
| ![Light theme](docs/screenshots/light.png) | ![Lyrics page](docs/screenshots/lyrics.png) | ![Online search with the example plugin](docs/screenshots/online.png) |

## Project Structure

```
SwiftPaw/
├── main.go               # Entry point: creates the window, binds App
├── app.go                # App: folder scanning, play queue
├── app_store.go          # Playlists, favorites, recently played
├── app_config.go         # Settings (config.json)
├── app_lyrics.go         # Lyrics
├── app_plugins.go        # Plugins, online search, online playback
├── window.go             # Remembers the window size
├── version.go            # Version number
├── handler.go            # The three local endpoints: /music, /cover, /stream
├── internal/
│   ├── music/            # Song info, folder scanning
│   ├── queue/            # Play queue and play modes
│   ├── store/            # SQLite: playlists, favorites, recently played
│   ├── lyrics/           # LRC parsing, encoding detection, finding lyrics
│   ├── jsrt/             # JS runtime for plugins (goja + built-in modules)
│   ├── plugin/           # Installing, loading and calling plugins
│   ├── stream/           # Playback proxy for online songs
│   └── winstate/         # Saving and loading the window size
├── frontend/             # Frontend (plain JS + Vite, no framework)
│   ├── src/js/           # One small module per feature: player, lists, lyrics, plugins page…
│   ├── src/css/          # Styles; all colors are in theme.css
│   └── wailsjs/          # JS bindings for the Go methods (generated)
├── examples/plugins/     # Example plugin
├── scripts/
│   ├── genbindings/      # Generates the frontend bindings without installing Wails
│   └── jslib/            # Bundles the JS libraries used by plugins into one file
├── build/                # Icons, Windows installer config
└── .github/workflows/    # Tests and Windows packaging
```

## Getting Started

### Download

Download from [Releases](https://github.com/dongzhongcen/SwiftPaw/releases):

- `SwiftPaw-x.y.z-win-x64-setup.exe`: installer
- `SwiftPaw-x.y.z-win-x64.zip`: portable version; unzip it and run `SwiftPaw.exe`

For the latest code that hasn't been released yet, you can download the `SwiftPaw-win-x64` artifact from [Actions → 构建 Windows 版 (Build Windows)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml).

Requires Windows 10 / 11 (64-bit) and WebView2 (included with Windows 11, and already installed on most Windows 10 machines).

### First Use

1. Click "Choose folder" (选择文件夹) and pick the folder where you keep your music
2. Click a song to start playing; click the heart after a song to add it to favorites, or add it to a playlist
3. To search for songs online: open the "Plugins" (插件) page → "Install from file" (从文件安装), choose [`examples/plugins/archive-org.js`](examples/plugins/archive-org.js), then search on the "Online search" (在线搜索) page

### Running from Source

You'll need:

- [Go](https://go.dev/dl/) 1.25 or newer
- [Node.js](https://nodejs.org/) 20 or newer
- The Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0`

```bash
git clone https://github.com/dongzhongcen/SwiftPaw.git
cd SwiftPaw
wails dev                                   # Dev mode, reloads automatically when you change code
wails build -platform windows/amd64         # Build; output is build/bin/SwiftPaw.exe
wails build -platform windows/amd64 -nsis   # Also build the installer (requires NSIS)
```

Run the tests:

```bash
go test ./internal/...
cd frontend && npm ci && npm run build
```

### Writing a Plugin

A plugin is a `.js` file in CommonJS format:

```js
const axios = require("axios");

module.exports = {
  platform: "My music source",
  version: "1.0.0",
  async search(query, page) {
    // Return { isEnd: whether there are no more pages, data: [{ id, title, artist, album, artwork, duration }] }
  },
  async getMediaSource(musicItem, quality) {
    // Return { url: playback URL, headers: request headers it needs (optional) }
  },
  async getLyric(musicItem) {
    // Return { rawLrc: lyrics in LRC format }, or null if there are no lyrics
  },
};
```

For a complete example see [`examples/plugins/archive-org.js`](examples/plugins/archive-org.js); the format is described in the comments at the top of [`internal/plugin/types.go`](internal/plugin/types.go).

## Current Status

It's now at **1.0.0**. All the features listed above are done, with unit tests and UI walkthroughs. See [CHANGELOG.md](CHANGELOG.md) for the change history.

Known limitations:

- I've only tested it on Windows. There's nothing Windows-specific in the code, so in theory it should also build on macOS / Linux, but I haven't tried
- The plugin JS environment isn't full Node.js: common modules are built in, and plugins that use other modules (such as `webdav`) will show a "not supported yet" (暂不支持) message
- Online songs depend on plugins. If a plugin is uninstalled or stops working, online songs you've already added to favorites stay in your lists but can't be played
- There's no auto-update yet; you need to download new versions from Releases yourself

If you have questions or suggestions, feel free to open an [Issue](https://github.com/dongzhongcen/SwiftPaw/issues).
