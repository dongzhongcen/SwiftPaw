# 极拍 SwiftPaw

[简体中文](README.md) | **English**

[![Tests](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/test.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/test.yml)
[![Build Windows](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml)
[![Build Android](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-android.yml/badge.svg)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-android.yml)
[![Latest release](https://img.shields.io/github/v/release/dongzhongcen/SwiftPaw?label=latest%20release)](https://github.com/dongzhongcen/SwiftPaw/releases)
![Go](https://img.shields.io/github/go-mod/go-version/dongzhongcen/SwiftPaw)
![Platform](https://img.shields.io/badge/platform-Windows%2010%20%2F%2011%20%7C%20Android%208.0%2B-0078D4)
![Wails](https://img.shields.io/badge/Wails-v2.16-red)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

![SwiftPaw main window (dark theme)](docs/screenshots/dark.png)

## Introduction

SwiftPaw (极拍, "Jípāi") is a Windows desktop music player I wrote in Go with [Wails](https://wails.io), and it now has an Android version too.

What I wanted was a simple player: open it and listen to the music on my own computer, with a clean interface and no sign-in. So the interface is minimal black and white, and all data (playlists, favorites, play history) stays on your own computer.

Later I added a plugin system: a plugin is a small JS file that tells the player where to search for songs and how to get a playback URL. That way the player itself isn't tied to any music platform; the plugins decide what you can listen to. The repository comes with an example plugin that searches public-domain music on the Internet Archive.

The Android version is packaged with [Capacitor](https://capacitorjs.com). It uses the same frontend as the desktop version (which switches to a phone layout when the window is narrow), and the same Go code for the play queue, playlists, lyrics and plugins, compiled for Android with gomobile.

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
- Plugins can have their own settings (such as an API key); fill them in with the "Settings" (设置) button on the Plugins page, and they take effect as soon as you save
- Plugins run in a built-in JS engine ([goja](https://github.com/dop251/goja)), so you don't need Node.js; common modules such as `axios`, `crypto-js`, `cheerio` and `dayjs` are built in

**Appearance**
- Three themes: dark, light, and follow the system

**Android version**
- Requires Android 8.0 or newer and works on phones and tablets: a bottom tab bar in portrait, and a tab bar on the left on tablets and in landscape
- "Scan device music" (扫描本机音乐) finds all the music on your phone, with no folder to pick
- Background playback: keeps playing when you switch apps or lock the screen; pause, skip and seek from the notification and the lock screen, and headset and Bluetooth buttons work too
- Pauses automatically for phone calls or when another app plays sound, and when you unplug your headphones
- Tap the mini player to open a full-screen player and swipe down to close it; the back button goes back one level at a time instead of quitting
- Plugins, online search, playlists and favorites work the same as on the desktop (each device keeps its own data)

## Screenshots

The local songs in the screenshots are test audio I generated myself, and I wrote the lyrics just for this; the online search uses the example plugin from this repository.

| Light theme | Lyrics | Online search |
| :---: | :---: | :---: |
| ![Light theme](docs/screenshots/light.png) | ![Lyrics page](docs/screenshots/lyrics.png) | ![Online search with the example plugin](docs/screenshots/online.png) |

Android version:

| Local music | Player | Mine | Light theme |
| :---: | :---: | :---: | :---: |
| ![Local music on Android](docs/screenshots/android-library.png) | ![Full-screen player on Android](docs/screenshots/android-player.png) | ![The "Mine" page on Android](docs/screenshots/android-mine.png) | ![Light theme on Android](docs/screenshots/android-light.png) |

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
│   ├── core/             # Features shared by desktop and Android (App and the Android plugin both forward here)
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
│   ├── notices/          # Generates THIRD_PARTY_NOTICES.md
│   └── jslib/            # Bundles the JS libraries used by plugins into one file
├── mobile/               # Android version: Go core interface, Capacitor project, build scripts (see mobile/README.md)
├── build/                # Icons, Windows installer config
└── .github/workflows/    # Tests, Windows and Android packaging
```

## Getting Started

### Download

Download from [Releases](https://github.com/dongzhongcen/SwiftPaw/releases):

- `SwiftPaw-x.y.z-win-x64-setup.exe`: installer
- `SwiftPaw-x.y.z-win-x64.zip`: portable version; unzip it and run `SwiftPaw.exe`

For the latest code that hasn't been released yet, you can download the `SwiftPaw-win-x64` artifact from [Actions → 构建 Windows 版 (Build Windows)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-windows.yml).

Requires Windows 10 / 11 (64-bit) and WebView2 (included with Windows 11, and already installed on most Windows 10 machines).

For Android, download `SwiftPaw-x.y.z-android.apk` (requires Android 8.0 or newer). It isn't in an app store, so open the APK on your phone to install it.
The first time, Android asks whether to allow "Install unknown apps"; allow it for the app you used to open the APK (such as your browser or file manager).
To upgrade, just install the new APK; playlists, favorites and plugins are kept. For the latest unreleased code, download the `SwiftPaw-android` artifact from [Actions → 构建 Android 版 (Build Android)](https://github.com/dongzhongcen/SwiftPaw/actions/workflows/build-android.yml).

The installer and `SwiftPaw.exe` are not code-signed yet, so on first run Windows may show "Windows protected your PC". Click "More info", then "Run anyway". This is expected and does not mean the file is harmful.

To upgrade, just run the new installer: it checks for an existing install, asks whether to replace it, then uninstalls the old version and installs into the same folder. If SwiftPaw is running, it asks you to close it first. Playlists, favorites, plugins and plugin settings are stored in `%AppData%\SwiftPaw`, so upgrading keeps them.

Both the zip and the install folder include `LICENSE`, `THIRD_PARTY_NOTICES.md` (licenses of the third-party open-source software) and `OFL.txt` (the font license).

### First Use

1. Click "Choose folder" (选择文件夹) and pick the folder where you keep your music
2. Click a song to start playing; click the heart after a song to add it to favorites, or add it to a playlist
3. To search for songs online: open the "Plugins" (插件) page → "Install from file" (从文件安装), choose [`examples/plugins/archive-org.js`](examples/plugins/archive-org.js), then search on the "Online search" (在线搜索) page

On Android, tap "Scan device music" (扫描本机音乐) the first time and allow SwiftPaw to access "Music and audio". After you add new songs to your phone, tap "Rescan" (重新扫描) to see them.
To install a plugin on your phone, use "Install from URL" (从网址安装) and paste the plugin's address (for example the GitHub raw URL of the example plugin), or save the `.js` file to your phone and use "Install from file".

### Using the Jamendo Plugin

[`examples/plugins/jamendo.js`](examples/plugins/jamendo.js) searches and plays music from Jamendo through the official Jamendo API. It supports song search, playback in several qualities, and lyrics (when a song has them). You need your own Client ID:

1. Sign up and log in at [devportal.jamendo.com](https://devportal.jamendo.com), then create an app
2. Copy the app's Client ID
3. Open the "Plugins" (插件) page → "Install from file" (从文件安装) and choose `examples/plugins/jamendo.js`, or use "Install from URL" (从网址安装) with `https://raw.githubusercontent.com/dongzhongcen/SwiftPaw/main/examples/plugins/jamendo.js`
4. Click "Settings" (「设置」) on the Jamendo plugin, paste the Client ID into the "Client ID" field and save, then search on the "Online search" (在线搜索) page

If the Client ID is missing or wrong, the plugin tells you to check it in 「设置」. Music on Jamendo is published by its artists under Creative Commons licenses, and each track may use a different license; you must follow [Jamendo's API terms of use](https://devportal.jamendo.com/api_terms_of_use) and the track's license. Lossless quality is only offered for tracks whose artists allow downloads.

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

Building the Android version needs JDK 21, Node.js 22, the Android SDK and NDK; see [mobile/README.md](mobile/README.md) (in Chinese) for the steps:

```bash
mobile/scripts/build-apk.sh debug   # Output goes to mobile/android/app/build/outputs/apk/debug/
```

Run the tests:

```bash
go test ./internal/... ./mobile/...
cd frontend && npm ci && npm run build
```

### Writing a Plugin

A plugin is a `.js` file in CommonJS format, compatible with common plugin formats:

```js
const axios = require("axios");

module.exports = {
  platform: "My music source",
  version: "1.0.0",
  // Optional: settings the user fills in; the Plugins page shows a "Settings" button. name and hint are optional
  userVariables: [{ key: "key", name: "API Key", hint: "Get one from the service's website" }],
  async search(query, page) {
    const { key } = env.getUserVariables(); // Read the user's settings
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

It's now at **1.1.0**. All the features listed above are done, with unit tests and UI walkthroughs. See [CHANGELOG.md](CHANGELOG.md) for the change history.

Known limitations:

- I've only tested the desktop version on Windows. There's nothing Windows-specific in the code, so in theory it should also build on macOS / Linux, but I haven't tried
- Android: `.lrc` files next to your songs can't be read on Android 11 and newer (the system only lets regular apps read media files); lyrics embedded in the audio files are shown. Playlists and favorites aren't synced between desktop and Android
- The plugin JS environment isn't full Node.js: common modules are built in, and plugins that use other modules (such as `webdav`) will show a "not supported yet" (暂不支持) message
- Online songs depend on plugins. If a plugin is uninstalled or stops working, online songs you've already added to favorites stay in your lists but can't be played
- There's no auto-update yet; you need to download new versions from Releases yourself

If you have questions or suggestions, feel free to open an [Issue](https://github.com/dongzhongcen/SwiftPaw/issues).

## License

[MIT](LICENSE). The third-party open-source software used and their licenses are listed in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

## Disclaimer

This software does not provide any music sources and does not include any third-party platform plugins. Plugins are installed by users themselves, and users and plugin authors are responsible for their content and legality. Please comply with local laws and the terms of service of each platform, and use this software only for personal learning and lawful purposes.

The example plugins in this repository, `examples/plugins/archive-org.js` and `examples/plugins/jamendo.js`, are written by this project: the first only searches public-domain audio on the Internet Archive; the second accesses Creative Commons–licensed music through the official Jamendo API, and requires users to get their own Client ID and follow Jamendo's API terms of use.
