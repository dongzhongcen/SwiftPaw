# 极拍 SwiftPaw · Android 版

这个文件夹是 Android 版的代码。界面用的是仓库根目录 `frontend/` 里和桌面版同一套前端，
播放器的功能（播放队列、歌单、歌词、插件……）用的是同一份 Go 代码（`internal/core`）。

```
mobile/
├── gocore/          Go 内核给 Java 用的接口，用 gomobile bind 编译成 .aar
├── android/         Capacitor 生成的 Android 工程（Java 代码在 app/src/main/java）
├── scripts/
│   ├── build-aar.sh   编译 Go 内核 → android/app/libs/swiftpaw-core.aar
│   ├── build-apk.sh   完整编译安装包（前端 + Go 内核 + Gradle）
│   └── make-icons.py  根据 build/appicon.png 生成启动图标
├── capacitor.config.json
├── package.json     Capacitor 的版本
└── NOTICES.md       Android 依赖的许可证（会合并进根目录的 THIRD_PARTY_NOTICES.md）
```

## 它是怎么工作的

```
前端（WebView）──调用──> Capacitor 插件 SwiftPawPlugin（Java）──> Go 内核（gocore → internal/core）
```

- Go 内核通过 `gomobile bind` 编译成 `swiftpaw-core.aar`，Java 里是 `com.dongzhongcen.swiftpaw.core.gocore.Core`。
  它只有一个通用的 `call(方法名, JSON 参数)`，按名字调用 `internal/core` 里的方法，所以前端调用的方法名和参数和桌面版完全一样。
- 数据（配置、SQLite 数据库、插件）保存在应用的私有文件夹里，卸载应用时会一起删除。
- 数据库用的是纯 Go 的 `modernc.org/sqlite`，不需要 cgo 版的 SQLite，桌面版和 Android 版是同一份代码。

## 本地编译

需要：

- Go（版本见根目录的 `go.mod`）
- JDK 21
- Node.js 22 或更新（Capacitor 8 的要求）
- Android SDK（compileSdk 36）和 NDK（CI 用的是 27.3.13750724），设置好 `ANDROID_HOME`，`ANDROID_NDK_HOME` 不设置时用 SDK 里最新的 NDK

```bash
# debug 版（自动用调试签名，可以直接装到手机上）
mobile/scripts/build-apk.sh debug
# 安装包在 mobile/android/app/build/outputs/apk/debug/app-debug.apk
```

正式版需要签名，先设置这几个环境变量再运行 `mobile/scripts/build-apk.sh release`：

| 环境变量 | 内容 |
|----------|------|
| `ANDROID_KEYSTORE_FILE` | 签名文件（.jks）的路径 |
| `ANDROID_KEYSTORE_PASSWORD` | 签名文件的密码 |
| `ANDROID_KEY_ALIAS` | 密钥的别名 |
| `ANDROID_KEY_PASSWORD` | 密钥的密码 |

只改了前端时不用重新编 Go 内核：`cd frontend && npm run build`，然后 `cd mobile && npx cap sync android`，再用 Gradle 打包。

## CI

`.github/workflows/build-android.yml`：推送到 main 和发 PR 时编译安装包，上传为 `SwiftPaw-android` artifact。

- 仓库里配置了签名用的 Secrets（`ANDROID_KEYSTORE_BASE64`、`ANDROID_KEYSTORE_PASSWORD`、`ANDROID_KEY_ALIAS`、`ANDROID_KEY_PASSWORD`）时编签名的正式版 `SwiftPaw-x.y.z-android.apk`
- 拿不到 Secrets 时（比如 fork 的仓库发来的 PR）编 debug 版 `SwiftPaw-x.y.z-android-debug.apk`
- 推送 `v` 开头的标签时，由 `build-windows.yml` 调用这个流程，正式版安装包和 Windows 版一起放进同一个 Release

## 版本号

在 `android/app/build.gradle` 里：`appVersionName`（比如 1.1.0）和 `appVersionCode`（比如 10100，每次发布都要变大）。
