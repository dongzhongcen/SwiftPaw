package jsrt

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// newTestRuntime 创建运行环境并加载一段插件代码
func newTestRuntime(t *testing.T, source string, timeout time.Duration) *Runtime {
	t.Helper()
	rt, err := New(Options{Name: "测试插件", Timeout: timeout, Env: Env{AppVersion: "1.0.0"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rt.Close)
	if err := rt.Load(source); err != nil {
		t.Fatalf("加载插件失败：%v", err)
	}
	return rt
}

// callString 调用插件函数，把返回的 JSON 解析成字符串
func callString(t *testing.T, rt *Runtime, name string, args ...any) string {
	t.Helper()
	raw, err := rt.Call(name, args...)
	if err != nil {
		t.Fatalf("调用 %s 失败：%v", name, err)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("%s 的返回值不是字符串：%s", name, raw)
	}
	return s
}

func TestLoadMetaAndCall(t *testing.T) {
	rt := newTestRuntime(t, `
		module.exports = {
			platform: "测试", version: "0.1.0", author: "me", supportedSearchType: ["music"],
			search(query, page) { return { isEnd: true, data: [{ id: 1, title: query + page }] }; },
		};`, 0)
	meta, funcs, err := rt.Meta()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), `"platform":"测试"`) || len(funcs) != 1 || funcs[0] != "search" {
		t.Fatalf("Meta 不对：%s %v", meta, funcs)
	}
	raw, err := rt.Call("search", "晴天", 2)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"isEnd":true,"data":[{"id":1,"title":"晴天2"}]}` {
		t.Fatalf("返回值不对：%s", raw)
	}
	if !rt.Has("search") || rt.Has("getLyric") {
		t.Fatal("Has 判断不对")
	}
}

func TestExportsDefault(t *testing.T) {
	// TypeScript 编译出来的插件常见写法
	rt := newTestRuntime(t, `
		Object.defineProperty(exports, "__esModule", { value: true });
		exports.default = { platform: "ts", hello() { return "hi"; } };`, 0)
	if got := callString(t, rt, "hello"); got != "hi" {
		t.Fatalf("exports.default 没有生效：%q", got)
	}
}

func TestAsyncAwait(t *testing.T) {
	rt := newTestRuntime(t, `
		const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
		module.exports = {
			platform: "async",
			async hello(name) {
				await sleep(20);
				const parts = await Promise.all([sleep(5).then(() => "你好"), Promise.resolve(name)]);
				return parts.join("，");
			},
		};`, 0)
	if got := callString(t, rt, "hello", "世界"); got != "你好，世界" {
		t.Fatalf("async/await 结果不对：%q", got)
	}
}

func TestPluginThrows(t *testing.T) {
	rt := newTestRuntime(t, `
		module.exports = {
			platform: "bad",
			syncThrow() { throw new Error("同步出错"); },
			async asyncThrow() { await null; throw new TypeError("异步出错"); },
			rejectString() { return Promise.reject("就是不行"); },
			useMissing() { return undefinedVariable + 1; },
		};`, 0)
	cases := map[string]string{
		"syncThrow":    "同步出错",
		"asyncThrow":   "TypeError: 异步出错",
		"rejectString": "就是不行",
		"useMissing":   "undefinedVariable",
		"notExist":     "没有实现 notExist",
	}
	for fn, want := range cases {
		_, err := rt.Call(fn)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s：错误信息应该包含 %q，实际是 %v", fn, want, err)
		}
		if err != nil && !strings.Contains(err.Error(), "测试插件") {
			t.Errorf("%s：错误信息里应该有插件名：%v", fn, err)
		}
	}
}

func TestTimeout(t *testing.T) {
	rt := newTestRuntime(t, `
		module.exports = {
			platform: "slow",
			forever() { while (true) {} },
			never() { return new Promise(() => {}); },
			ok() { return "还活着"; },
		};`, 300*time.Millisecond)

	for _, fn := range []string{"forever", "never"} {
		start := time.Now()
		_, err := rt.Call(fn)
		if err == nil || !strings.Contains(err.Error(), "超时") {
			t.Fatalf("%s 应该超时，实际：%v", fn, err)
		}
		if time.Since(start) > 2*time.Second {
			t.Fatalf("%s 超时用了太久：%v", fn, time.Since(start))
		}
	}
	// 死循环被打断以后，插件还能继续用
	if got := callString(t, rt, "ok"); got != "还活着" {
		t.Fatalf("超时后再调用失败：%q", got)
	}
}

func TestSyntaxError(t *testing.T) {
	rt, err := New(Options{Name: "语法错误"})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	if err := rt.Load("module.exports = {"); err == nil || !strings.Contains(err.Error(), "语法错误") {
		t.Fatalf("应该报语法错误：%v", err)
	}
}

func TestRequireMissing(t *testing.T) {
	rt, err := New(Options{Name: "缺模块"})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	err = rt.Load(`const webdav = require("webdav"); module.exports = { platform: "x" };`)
	if err == nil || !strings.Contains(err.Error(), `暂不支持模块 "webdav"`) {
		t.Fatalf("应该提示不支持的模块：%v", err)
	}
	if m := rt.MissingModules(); len(m) != 1 || m[0] != "webdav" {
		t.Fatalf("MissingModules 不对：%v", m)
	}
}

func TestEnvAndGlobals(t *testing.T) {
	rt := newTestRuntime(t, `
		module.exports = {
			platform: "env",
			info() {
				const u = new URL("https://example.com/a?x=1&y=%E4%BD%A0");
				const sp = new URLSearchParams({ q: "晴天 周杰伦", page: 1 });
				return [env.appVersion, env.os, env.lang, JSON.stringify(env.getUserVariables()),
					u.searchParams.get("y"), sp.toString(), btoa("abc"), atob("YWJj"),
					Buffer.from("你好").toString("base64"), typeof setTimeout, typeof clearTimeout].join("|");
			},
		};`, 0)
	want := "1.0.0|win32|zh-CN|{}|你|q=%E6%99%B4%E5%A4%A9+%E5%91%A8%E6%9D%B0%E4%BC%A6&page=1|YWJj|abc|5L2g5aW9|function|function"
	if got := callString(t, rt, "info"); got != want {
		t.Fatalf("全局对象不对：\n得到 %s\n期望 %s", got, want)
	}
}

func TestClearTimeout(t *testing.T) {
	rt := newTestRuntime(t, `
		module.exports = {
			platform: "timer",
			run() {
				return new Promise((resolve) => {
					const id = setTimeout(() => resolve("不应该执行"), 30);
					clearTimeout(id);
					setTimeout(() => resolve("清除成功"), 60);
				});
			},
		};`, 0)
	if got := callString(t, rt, "run"); got != "清除成功" {
		t.Fatalf("clearTimeout 没生效：%q", got)
	}
}
