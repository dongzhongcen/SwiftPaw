// Package jsrt 是插件的 JavaScript 运行环境，基于 goja（纯 Go 写的 JS 引擎）。
//
// 每个插件有自己独立的 Runtime：一个 goja 虚拟机 + 一个事件循环（跑在单独的 goroutine 上）。
// goja 的虚拟机不是线程安全的，所以所有 JS 相关的操作都要通过 loop.RunOnLoop 放到事件循环里执行，
// 然后用 channel 把结果送回调用方。这和 Java 里“所有 UI 操作都要切到 UI 线程”是一个思路。
package jsrt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
)

// DefaultTimeout 是每次调用插件函数的默认超时时间
const DefaultTimeout = 15 * time.Second

// ErrClosed 表示运行环境已经关闭（比如插件被卸载了）
var ErrClosed = errors.New("插件已经停止运行")

// Env 是插件通过全局变量 env 能读到的信息
type Env struct {
	AppVersion    string
	OS            string            // 默认 win32
	Lang          string            // 默认 zh-CN
	UserVariables map[string]string // 用户给插件填的变量，目前为空
}

// Options 是创建运行环境的参数
type Options struct {
	Name       string        // 插件名，用在错误信息和日志里
	Timeout    time.Duration // 每次调用的超时，0 表示用 DefaultTimeout
	HTTPClient *http.Client  // axios 用的 HTTP 客户端，nil 时自动创建一个带 Cookie 的
	Env        Env
	Logger     func(level, message string) // console.log 的输出，nil 时丢弃
}

// Runtime 是一个插件的 JS 运行环境
type Runtime struct {
	opts    Options
	loop    *eventloop.EventLoop
	vm      *goja.Runtime // 只在事件循环里使用；例外是 Interrupt，它是线程安全的
	client  *http.Client
	exports *goja.Object // 插件的 module.exports

	jsonParse     goja.Callable
	jsonStringify goja.Callable

	mu      sync.Mutex
	missing map[string]bool // 插件 require 了但我们不支持的模块
	closed  bool
}

// New 创建并启动一个运行环境
func New(opts Options) (*Runtime, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.Env.OS == "" {
		opts.Env.OS = "win32"
	}
	if opts.Env.Lang == "" {
		opts.Env.Lang = "zh-CN"
	}
	client := opts.HTTPClient
	if client == nil {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Jar: jar, CheckRedirect: onlyHTTPRedirects}
	}

	// 不允许从磁盘加载任何 JS 文件：插件只能 require 我们内置的模块
	registry := require.NewRegistry(require.WithLoader(func(string) ([]byte, error) {
		return nil, require.ModuleFileDoesNotExistError
	}))
	loop := eventloop.NewEventLoop(eventloop.WithRegistry(registry), eventloop.EnableConsole(false))
	r := &Runtime{opts: opts, loop: loop, client: client, missing: map[string]bool{}}

	_, err := r.run(func(vm *goja.Runtime) (goja.Value, error) {
		r.vm = vm
		return nil, r.setupGlobals(vm)
	}, false)
	if err != nil {
		return nil, err
	}
	loop.Start()
	return r, nil
}

// setupGlobals 准备插件能用的全局变量：require、console、env、URL、Buffer 等
func (r *Runtime) setupGlobals(vm *goja.Runtime) error {
	// 让 Go 结构体的字段名在 JS 里按 json 标签显示（比如 Status -> status）
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	jsonObj := vm.Get("JSON").ToObject(vm)
	r.jsonParse, _ = goja.AssertFunction(jsonObj.Get("parse"))
	r.jsonStringify, _ = goja.AssertFunction(jsonObj.Get("stringify"))

	url.Enable(vm)    // URL、URLSearchParams
	buffer.Enable(vm) // Buffer
	nodeRequire, _ := goja.AssertFunction(vm.Get("require"))

	if err := setupBrowserGlobals(vm, r); err != nil {
		return err
	}
	if err := loadLibs(vm); err != nil {
		return err
	}
	return vm.Set("require", r.makeRequire(vm, nodeRequire))
}

// run 把 fn 放到事件循环里执行并等待结果。fn 返回 Promise 时等 Promise 完成。
// started 表示事件循环是否已经启动（New 里第一次调用时还没启动）。
func (r *Runtime) run(fn func(vm *goja.Runtime) (goja.Value, error), started bool) (goja.Value, error) {
	type result struct {
		value goja.Value
		err   error
	}
	done := make(chan result, 1) // 带缓冲：超时后没人接收也不会卡住事件循环
	finish := func(v goja.Value, err error) {
		select {
		case done <- result{v, err}:
		default:
		}
	}
	job := func(vm *goja.Runtime) {
		vm.ClearInterrupt()
		v, err := fn(vm)
		if err != nil {
			finish(nil, err)
			return
		}
		r.settle(vm, v, finish)
	}

	if !started {
		// 事件循环还没启动：直接在当前 goroutine 里执行（New 的初始化阶段）
		r.loop.Run(job)
	} else if !r.loop.RunOnLoop(job) {
		return nil, ErrClosed
	}

	timer := time.NewTimer(r.opts.Timeout)
	defer timer.Stop()
	select {
	case res := <-done:
		return res.value, res.err
	case <-timer.C:
		// 打断正在执行的 JS（比如死循环），让事件循环能继续处理别的调用
		r.vm.Interrupt("timeout")
		return nil, fmt.Errorf("插件「%s」响应超时（超过 %d 秒）", r.opts.Name, int(r.opts.Timeout.Seconds()))
	}
}

// settle 处理 JS 函数的返回值：普通值直接返回，Promise 等它 resolve/reject
func (r *Runtime) settle(vm *goja.Runtime, v goja.Value, finish func(goja.Value, error)) {
	promise, ok := exportPromise(v)
	if !ok {
		finish(v, nil)
		return
	}
	switch promise.State() {
	case goja.PromiseStateFulfilled:
		finish(promise.Result(), nil)
		return
	case goja.PromiseStateRejected:
		finish(nil, r.jsError(vm, promise.Result()))
		return
	}
	// 还没完成：调用 promise.then(成功回调, 失败回调)，回调同样在事件循环里执行
	then, _ := goja.AssertFunction(v.ToObject(vm).Get("then"))
	onFulfilled := vm.ToValue(func(call goja.FunctionCall) goja.Value {
		finish(call.Argument(0), nil)
		return goja.Undefined()
	})
	onRejected := vm.ToValue(func(call goja.FunctionCall) goja.Value {
		finish(nil, r.jsError(vm, call.Argument(0)))
		return goja.Undefined()
	})
	if _, err := then(v, onFulfilled, onRejected); err != nil {
		finish(nil, r.jsError(vm, err))
	}
}

func exportPromise(v goja.Value) (*goja.Promise, bool) {
	if v == nil {
		return nil, false
	}
	p, ok := v.Export().(*goja.Promise)
	return p, ok
}

// Load 执行插件代码（CommonJS 格式），记住它的 module.exports
func (r *Runtime) Load(source string) error {
	// 把插件代码包在函数里，提供 module、exports、require 三个变量，和 Node.js 一样
	wrapped := "(function(module, exports, require) {" + source + "\n})"
	program, err := goja.Compile(r.opts.Name+".js", wrapped, false)
	if err != nil {
		return fmt.Errorf("插件代码有语法错误：%v", err)
	}
	_, err = r.run(func(vm *goja.Runtime) (goja.Value, error) {
		fnValue, err := vm.RunProgram(program)
		if err != nil {
			return nil, r.jsError(vm, err)
		}
		fn, _ := goja.AssertFunction(fnValue)
		module := vm.NewObject()
		exports := vm.NewObject()
		module.Set("exports", exports)
		if _, err := fn(goja.Undefined(), module, exports, vm.Get("require")); err != nil {
			return nil, r.jsError(vm, err)
		}
		r.exports = pickExports(vm, module.Get("exports"))
		if r.exports == nil {
			return nil, errors.New("插件没有导出任何内容（需要设置 module.exports）")
		}
		return nil, nil
	}, true)
	return err
}

// pickExports 兼容 TypeScript 编译出来的写法：exports.default = {...}
func pickExports(vm *goja.Runtime, v goja.Value) *goja.Object {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	obj := v.ToObject(vm)
	if def := obj.Get("default"); def != nil && !goja.IsUndefined(def) && !goja.IsNull(def) {
		if d, ok := def.(*goja.Object); ok && d.Get("platform") != nil && obj.Get("platform") == nil {
			return d
		}
	}
	return obj
}

// Meta 返回插件导出对象里所有“不是函数”的字段（转成 JSON），以及函数名列表
func (r *Runtime) Meta() (json.RawMessage, []string, error) {
	var funcs []string
	v, err := r.run(func(vm *goja.Runtime) (goja.Value, error) {
		if r.exports == nil {
			return nil, errors.New("插件还没有加载")
		}
		for _, key := range r.exports.Keys() {
			if _, ok := goja.AssertFunction(r.exports.Get(key)); ok {
				funcs = append(funcs, key)
			}
		}
		return r.exports, nil
	}, true)
	if err != nil {
		return nil, nil, err
	}
	data, err := r.toJSON(v)
	return data, funcs, err
}

// Has 判断插件有没有实现某个函数
func (r *Runtime) Has(name string) bool {
	_, funcs, err := r.Meta()
	if err != nil {
		return false
	}
	for _, f := range funcs {
		if f == name {
			return true
		}
	}
	return false
}

// Call 调用插件导出的函数 name，参数会先转成 JSON 再变成 JS 对象，返回值转成 JSON。
// 函数可以是 async 的（返回 Promise），会等它完成；整个过程超过超时时间就返回错误。
func (r *Runtime) Call(name string, args ...any) (json.RawMessage, error) {
	argJSON := make([]string, len(args))
	for i, a := range args {
		data, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		argJSON[i] = string(data)
	}
	v, err := r.run(func(vm *goja.Runtime) (goja.Value, error) {
		if r.exports == nil {
			return nil, errors.New("插件还没有加载")
		}
		fn, ok := goja.AssertFunction(r.exports.Get(name))
		if !ok {
			return nil, fmt.Errorf("插件「%s」没有实现 %s", r.opts.Name, name)
		}
		jsArgs := make([]goja.Value, len(argJSON))
		for i, s := range argJSON {
			parsed, err := r.jsonParse(goja.Undefined(), vm.ToValue(s))
			if err != nil {
				return nil, err
			}
			jsArgs[i] = parsed
		}
		ret, err := fn(r.exports, jsArgs...)
		if err != nil {
			return nil, r.jsError(vm, err)
		}
		return ret, nil
	}, true)
	if err != nil {
		return nil, err
	}
	return r.toJSON(v)
}

// toJSON 在事件循环里调用 JSON.stringify，把 JS 值变成 JSON（undefined 变成 null）
func (r *Runtime) toJSON(v goja.Value) (json.RawMessage, error) {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return json.RawMessage("null"), nil
	}
	var out string
	_, err := r.run(func(vm *goja.Runtime) (goja.Value, error) {
		s, err := r.jsonStringify(goja.Undefined(), v)
		if err != nil {
			return nil, r.jsError(vm, err)
		}
		if goja.IsUndefined(s) {
			out = "null"
		} else {
			out = s.String()
		}
		return nil, nil
	}, true)
	return json.RawMessage(out), err
}

// MissingModules 返回插件 require 过但不支持的模块名
func (r *Runtime) MissingModules() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := make([]string, 0, len(r.missing))
	for name := range r.missing {
		list = append(list, name)
	}
	return list
}

// Close 停止事件循环。正在执行的 JS 会被打断，还没完成的定时器和请求都会被丢弃。
func (r *Runtime) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()
	r.vm.Interrupt("closed")
	r.loop.Terminate()
}

func (r *Runtime) log(level, msg string) {
	if r.opts.Logger != nil {
		r.opts.Logger(level, msg)
	}
}
