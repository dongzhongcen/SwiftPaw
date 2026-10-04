// Package gocore 是给 Android 用的 Go 接口，用 gomobile bind 编译成 .aar（见 mobile/scripts/build-aar.sh）。
//
// gomobile 只支持很简单的参数类型（字符串、数字、[]byte 等），所以这里不一个个导出方法，
// 而是提供一个通用的 Call(方法名, JSON 参数)：按名字调用 internal/core 里 *core.Core 的公开方法，
// 参数和返回值都用 JSON。这样桌面版前端调用的方法名、参数和返回值在 Android 上完全一样，
// 以后给 core 加方法也不用改这里。
package gocore

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"musicplayer/internal/core"
)

// hidden 是不能从前端调用的方法：生命周期由 Java 代码管理，Stream 和 BackgroundHandler 只给桌面版用
var hidden = map[string]bool{"Close": true, "Stream": true, "BackgroundHandler": true}

// Core 是给 Java 用的内核对象，整个程序只创建一个
type Core struct {
	core    *core.Core
	methods map[string]reflect.Value
}

// Picture 是一张封面图片
type Picture struct {
	MIME string
	Data []byte
}

// NewCore 创建内核。dataDir 是应用私有的文件夹（Context.getFilesDir()），appVersion 是版本号
func NewCore(dataDir, appVersion string) *Core {
	c := core.New(core.Options{DataDir: dataDir, AppVersion: appVersion, Logf: log.Printf})
	return &Core{core: c, methods: methodsOf(c)}
}

// methodsOf 列出可以调用的方法
func methodsOf(c *core.Core) map[string]reflect.Value {
	v := reflect.ValueOf(c)
	t := v.Type()
	methods := map[string]reflect.Value{}
	for i := 0; i < t.NumMethod(); i++ {
		name := t.Method(i).Name
		if !hidden[name] {
			methods[name] = v.Method(i)
		}
	}
	return methods
}

// Methods 返回所有可以调用的方法名（JSON 数组），调试用
func (m *Core) Methods() string {
	names := make([]string, 0, len(m.methods))
	for name := range m.methods {
		names = append(names, name)
	}
	data, _ := json.Marshal(names)
	return string(data)
}

// Call 调用内核的一个方法。argsJSON 是参数数组（比如 [3, {"path": "..."}]），可以为空字符串；
// 返回值是方法结果的 JSON（没有返回值时是 null）。方法返回的 error 会变成 Java 的异常
func (m *Core) Call(method, argsJSON string) (string, error) {
	fn, ok := m.methods[method]
	if !ok {
		return "", fmt.Errorf("没有这个方法：%s", method)
	}
	var args []json.RawMessage
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("%s 的参数不是 JSON 数组：%v", method, err)
		}
	}
	t := fn.Type()
	if len(args) > t.NumIn() {
		return "", fmt.Errorf("%s 最多 %d 个参数，收到 %d 个", method, t.NumIn(), len(args))
	}
	in := make([]reflect.Value, t.NumIn())
	for i := range in {
		p := reflect.New(t.In(i))
		// 少传的参数用零值，和 Wails 的行为一样；JSON 里的 null 也当作零值
		if i < len(args) && string(args[i]) != "null" {
			if err := json.Unmarshal(args[i], p.Interface()); err != nil {
				return "", fmt.Errorf("%s 的第 %d 个参数不对：%v", method, i+1, err)
			}
		}
		in[i] = p.Elem()
	}

	var result any
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	for _, out := range fn.Call(in) {
		if out.Type() == errorType {
			if !out.IsNil() {
				return "", out.Interface().(error)
			}
			continue
		}
		result = out.Interface()
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("%s 的返回值转 JSON 失败：%v", method, err)
	}
	return string(data), nil
}

// Cover 读取本地歌曲内嵌的封面；没有封面时返回错误（Java 里是异常）
func (m *Core) Cover(path string) (*Picture, error) {
	picture, err := m.core.Cover(path)
	if err != nil {
		return nil, err
	}
	if picture.MIME == "" {
		picture.MIME = "image/jpeg"
	}
	return &Picture{MIME: picture.MIME, Data: picture.Data}, nil
}

// BackgroundImage 是一张保存在数据文件夹里的背景图片
type BackgroundImage struct {
	Path string // 完整路径，Java 直接打开这个文件
	MIME string
}

// Background 找到前端 /background?name=... 要的背景图片。name 必须是 SetBackgroundImage 返回的文件名，
// 不能拿它读别的文件；找不到时返回错误（Java 里是异常）
func (m *Core) Background(name string) (*BackgroundImage, error) {
	path, err := m.core.BackgroundImagePath(name)
	if err != nil {
		return nil, err
	}
	return &BackgroundImage{Path: path, MIME: core.BackgroundContentType(name)}, nil
}

// Close 关闭数据库和插件运行环境
func (m *Core) Close() {
	m.core.Close()
}
