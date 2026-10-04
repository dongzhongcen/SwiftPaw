package jsrt

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dop251/goja"
)

// nodeModules 是交给 goja_nodejs 处理的 Node.js 内置模块
var nodeModules = []string{"url", "buffer", "util", "process"}

// SupportedModules 返回插件可以 require 的所有模块名
func SupportedModules() []string {
	list := append([]string{"axios"}, bundledLibs...)
	return append(list, nodeModules...)
}

// makeRequire 生成插件里用的 require 函数。
// 不认识的模块会抛出中文错误，并记录下来，插件页面会显示“缺少哪些模块”。
func (r *Runtime) makeRequire(vm *goja.Runtime, nodeRequire goja.Callable) func(goja.FunctionCall) goja.Value {
	cache := map[string]goja.Value{}
	return func(call goja.FunctionCall) goja.Value {
		name := strings.TrimPrefix(call.Argument(0).String(), "node:")
		if v, ok := cache[name]; ok {
			return v
		}

		var module goja.Value
		switch {
		case name == "axios":
			module = r.newAxios(vm, nil)
		case slices.Contains(bundledLibs, name):
			factory, _ := goja.AssertFunction(vm.Get("__swiftpawLibs").ToObject(vm).Get(name))
			v, err := factory(goja.Undefined())
			if err != nil {
				panic(err)
			}
			module = v
		case slices.Contains(nodeModules, name):
			v, err := nodeRequire(goja.Undefined(), vm.ToValue(name))
			if err != nil {
				panic(err)
			}
			module = v
		default:
			r.mu.Lock()
			r.missing[name] = true
			r.mu.Unlock()
			panic(vm.NewGoError(fmt.Errorf("暂不支持模块 \"%s\"", name)))
		}
		cache[name] = module
		return module
	}
}
