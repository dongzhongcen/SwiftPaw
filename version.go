package main

// Version 是 SwiftPaw 的版本号。设置页的“关于”里显示，也会通过 env.appVersion 告诉插件。
// 发新版本时改这里，同时改 wails.json 里的 info.productVersion。
const Version = "1.1.0"

// AppVersion 返回版本号给前端
func (a *App) AppVersion() string {
	return Version
}
