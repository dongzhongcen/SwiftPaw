package jsrt

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"strings"

	"github.com/dop251/goja"
)

// setupBrowserGlobals 提供插件常用的全局对象：console、env、crypto.getRandomValues、atob/btoa 等。
// setTimeout / clearTimeout 由 goja_nodejs 的事件循环提供。
func setupBrowserGlobals(vm *goja.Runtime, r *Runtime) error {
	global := vm.GlobalObject()
	global.Set("globalThis", global)

	// console：输出交给 Options.Logger，界面上不会弹出来
	console := vm.NewObject()
	for _, level := range []string{"log", "info", "warn", "error", "debug"} {
		level := level
		console.Set(level, func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = r.formatLogArg(vm, arg)
			}
			r.log(level, strings.Join(parts, " "))
			return goja.Undefined()
		})
	}
	global.Set("console", console)

	// env：插件可以读到的环境信息
	env := vm.NewObject()
	env.Set("appVersion", r.opts.Env.AppVersion)
	env.Set("os", r.opts.Env.OS)
	env.Set("lang", r.opts.Env.Lang)
	env.Set("getUserVariables", func(goja.FunctionCall) goja.Value {
		vars := vm.NewObject()
		for k, v := range r.opts.Env.UserVariables {
			vars.Set(k, v)
		}
		return vars
	})
	global.Set("env", env)

	// process：有些插件会读 process.env 或 process.platform
	process := vm.NewObject()
	process.Set("env", vm.NewObject())
	process.Set("platform", r.opts.Env.OS)
	global.Set("process", process)

	// crypto.getRandomValues：crypto-js 生成随机盐（AES 加密）时需要
	global.Set("__swiftpawRandomUint32", func(goja.FunctionCall) goja.Value {
		var b [4]byte
		_, _ = rand.Read(b[:])
		return vm.ToValue(binary.LittleEndian.Uint32(b[:]))
	})

	// atob / btoa：Base64 解码 / 编码（和浏览器一样按 Latin-1 处理）
	global.Set("btoa", func(call goja.FunctionCall) goja.Value {
		s := call.Argument(0).String()
		b := make([]byte, 0, len(s))
		for _, c := range s {
			if c > 0xFF {
				panic(vm.NewTypeError("btoa: 字符串里有超出 Latin-1 范围的字符"))
			}
			b = append(b, byte(c))
		}
		return vm.ToValue(base64.StdEncoding.EncodeToString(b))
	})
	global.Set("atob", func(call goja.FunctionCall) goja.Value {
		s := strings.TrimRight(strings.Map(func(c rune) rune {
			if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
				return -1
			}
			return c
		}, call.Argument(0).String()), "=")
		b, err := base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			panic(vm.NewTypeError("atob: 不是合法的 Base64"))
		}
		runes := make([]rune, len(b))
		for i, c := range b {
			runes[i] = rune(c)
		}
		return vm.ToValue(string(runes))
	})

	_, err := vm.RunString(`
		// goja_nodejs 的 Buffer 缺几个静态方法，cheerio 等库会用到
		if (typeof Buffer !== "undefined") {
			if (!Buffer.isBuffer) Buffer.isBuffer = (b) => b instanceof Buffer;
			if (!Buffer.byteLength) {
				Buffer.byteLength = (s, encoding) =>
					typeof s === "string" ? Buffer.from(s, encoding).length : s.byteLength ?? s.length;
			}
		}
		globalThis.crypto = {
			getRandomValues(array) {
				const bytes = array.BYTES_PER_ELEMENT || 1;
				for (let i = 0; i < array.length; i++) {
					const n = __swiftpawRandomUint32();
					array[i] = bytes >= 4 ? n : n & ((1 << (bytes * 8)) - 1);
				}
				return array;
			},
		};
	`)
	return err
}

// formatLogArg 把 console.log 的参数变成字符串：对象用 JSON 显示
func (r *Runtime) formatLogArg(vm *goja.Runtime, v goja.Value) string {
	if obj, ok := v.(*goja.Object); ok {
		if _, isFunc := goja.AssertFunction(obj); !isFunc && obj.ClassName() != "Error" {
			if s, err := r.jsonStringify(goja.Undefined(), v); err == nil && !goja.IsUndefined(s) {
				return s.String()
			}
		}
	}
	return v.String()
}
