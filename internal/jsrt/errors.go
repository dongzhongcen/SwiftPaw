package jsrt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// PluginError 是插件抛出的错误。Message 是给用户看的中文说明。
type PluginError struct {
	Message string
	Stack   string // JS 的调用栈，方便排查问题
}

func (e *PluginError) Error() string { return e.Message }

// jsError 把 JS 里的各种错误（throw 的值、Promise reject 的原因、goja 的异常）变成 Go 的 error。
// 必须在事件循环里调用。
func (r *Runtime) jsError(vm *goja.Runtime, reason any) error {
	name := r.opts.Name
	switch e := reason.(type) {
	case nil:
		return &PluginError{Message: fmt.Sprintf("插件「%s」出错了", name)}
	case *goja.InterruptedError:
		return fmt.Errorf("插件「%s」响应超时，已被中断", name)
	case *goja.Exception:
		err := r.jsError(vm, e.Value())
		var pe *PluginError
		if errors.As(err, &pe) && pe.Stack == "" {
			pe.Stack = e.String()
		}
		return err
	case *goja.CompilerSyntaxError:
		return &PluginError{Message: fmt.Sprintf("插件「%s」代码有语法错误：%v", name, e)}
	case goja.Value:
		return &PluginError{Message: fmt.Sprintf("插件「%s」出错：%s", name, describeJSError(vm, e))}
	case error:
		return &PluginError{Message: fmt.Sprintf("插件「%s」出错：%v", name, e)}
	}
	return &PluginError{Message: fmt.Sprintf("插件「%s」出错：%v", name, reason)}
}

// describeJSError 尽量从 JS 错误对象里找出可读的信息；axios 的错误会带上 HTTP 状态码
func describeJSError(vm *goja.Runtime, v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "未知错误"
	}
	obj, ok := v.(*goja.Object)
	if !ok {
		return v.String()
	}
	message := ""
	if m := obj.Get("message"); m != nil && !goja.IsUndefined(m) {
		message = m.String()
	}
	if isAxios := obj.Get("isAxiosError"); isAxios != nil && isAxios.ToBoolean() {
		if resp, ok := obj.Get("response").(*goja.Object); ok {
			return fmt.Sprintf("网络请求失败，服务器返回 %v（%s）", resp.Get("status"), shortURL(obj))
		}
		return fmt.Sprintf("网络请求失败：%s（%s）", translateNetError(message), shortURL(obj))
	}
	if message == "" {
		return v.String()
	}
	// GoError 是 Go 代码里抛出的错误（比如 require 不支持的模块），名字对用户没意义
	if n := obj.Get("name"); n != nil && !goja.IsUndefined(n) && n.String() != "Error" && n.String() != "GoError" {
		return n.String() + ": " + message
	}
	return message
}

func shortURL(axiosErr *goja.Object) string {
	cfg, ok := axiosErr.Get("config").(*goja.Object)
	if !ok {
		return ""
	}
	u := cfg.Get("url")
	if u == nil || goja.IsUndefined(u) {
		return ""
	}
	s := u.String()
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i] // 不把查询参数（可能有签名之类的）显示出来
	}
	return s
}

// translateNetError 把常见的英文网络错误换成中文
func translateNetError(msg string) string {
	switch {
	case strings.Contains(msg, "timeout"):
		return "请求超时"
	case strings.Contains(msg, "no such host"):
		return "找不到服务器（DNS 解析失败）"
	case strings.Contains(msg, "connection refused"):
		return "服务器拒绝连接"
	}
	return msg
}
