package jsrt

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// echoServer 把收到的请求原样写回去（JSON），测试 axios 发出去的请求对不对
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status/404":
			http.Error(w, `{"msg":"没找到"}`, http.StatusNotFound)
			return
		case "/slow":
			time.Sleep(500 * time.Millisecond)
		case "/text":
			w.Write([]byte("不是 JSON"))
			return
		case "/redirect":
			http.Redirect(w, r, "/echo?from=redirect", http.StatusFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		json.NewEncoder(w).Encode(map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"ua":          r.Header.Get("User-Agent"),
			"referer":     r.Header.Get("Referer"),
			"token":       r.Header.Get("X-Token"),
			"contentType": r.Header.Get("Content-Type"),
			"body":        string(body),
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAxios(t *testing.T) {
	srv := echoServer(t)
	rt := newTestRuntime(t, `
		const axios = require("axios");
		const base = "`+srv.URL+`";
		module.exports = {
			platform: "axios",
			async get() {
				const res = await axios.get(base + "/echo?x=0", {
					params: { q: "晴天 周杰伦", page: 2, list: [1, 2], skip: undefined },
					headers: { Referer: "https://example.com/", "User-Agent": "TestUA" },
				});
				return [res.status, res.data.method, res.data.query, res.data.ua, res.data.referer,
					res.headers["content-type"], res.headers["set-cookie"].join(";")].join("|");
			},
			async postJSON() {
				const res = await axios.post(base + "/echo", { name: "你好", n: 1 });
				return res.data.contentType + "|" + res.data.body;
			},
			async postString() {
				const res = await axios.post(base + "/echo", "a=1&b=2");
				return res.data.contentType + "|" + res.data.body;
			},
			async postForm() {
				const res = await axios({
					method: "post", url: base + "/echo", data: { a: "x y", b: 2 },
					headers: { "content-type": "application/x-www-form-urlencoded" },
				});
				return res.data.contentType + "|" + res.data.body;
			},
			async postSearchParams() {
				const res = await axios.post(base + "/echo", new URLSearchParams({ k: "v" }));
				return res.data.body;
			},
			async callForms() {
				const a = await axios(base + "/echo", { method: "put" });
				const b = await axios.request({ url: base + "/echo", method: "delete" });
				const c = await axios.default.get(base + "/echo");
				return [a.data.method, b.data.method, c.data.method].join(",");
			},
			async created() {
				const api = axios.create({ baseURL: base + "/api/", headers: { "X-Token": "t1" }, timeout: 5000 });
				const res = await api.get("/songs", { params: { id: 7 } });
				return res.data.path + "?" + res.data.query + "|" + res.data.token;
			},
			async notFound() {
				try {
					await axios.get(base + "/status/404");
					return "没有抛出错误";
				} catch (e) {
					return [e.message, e.response.status, e.response.data.msg, axios.isAxiosError(e)].join("|");
				}
			},
			async notFoundUncaught() {
				await axios.get(base + "/status/404?sign=secret");
			},
			async timeout() {
				try {
					await axios.get(base + "/slow", { timeout: 100 });
					return "没有超时";
				} catch (e) {
					return e.code + "|" + e.message;
				}
			},
			async text() {
				const res = await axios.get(base + "/text");
				const raw = await axios.get(base + "/echo", { responseType: "text" });
				return typeof res.data + ":" + res.data + "|" + typeof raw.data;
			},
			async arraybuffer() {
				const res = await axios.get(base + "/text", { responseType: "arraybuffer" });
				return String(res.data.byteLength);
			},
			async redirect() {
				const res = await axios.get(base + "/redirect");
				return res.data.query;
			},
			async badScheme() {
				try { await axios.get("file:///etc/passwd"); } catch (e) { return e.message; }
			},
		};`, 0)

	cases := []struct{ fn, want string }{
		{"get", "200|GET|x=0&q=%E6%99%B4%E5%A4%A9+%E5%91%A8%E6%9D%B0%E4%BC%A6&page=2&list[]=1&list[]=2|TestUA|https://example.com/|application/json|a=1;b=2"},
		{"postJSON", `application/json|{"name":"你好","n":1}`},
		{"postString", "application/x-www-form-urlencoded|a=1&b=2"},
		{"postForm", "application/x-www-form-urlencoded|a=x+y&b=2"},
		{"postSearchParams", "k=v"},
		{"callForms", "PUT,DELETE,GET"},
		{"created", "/api/songs?id=7|t1"},
		{"notFound", "Request failed with status code 404|404|没找到|true"},
		{"timeout", "ECONNABORTED|timeout of 100ms exceeded"},
		{"text", "string:不是 JSON|string"},
		{"arraybuffer", "11"},
		{"redirect", "from=redirect"},
		{"badScheme", `只支持 http/https 地址："file:///etc/passwd"`},
	}
	for _, c := range cases {
		if got := callString(t, rt, c.fn); got != c.want {
			t.Errorf("%s：\n得到 %s\n期望 %s", c.fn, got, c.want)
		}
	}

	// 没有被插件 catch 的 HTTP 错误，变成可读的中文错误，而且不显示查询参数
	_, err := rt.Call("notFoundUncaught")
	if err == nil || !strings.Contains(err.Error(), "服务器返回 404") || strings.Contains(err.Error(), "secret") {
		t.Errorf("未捕获的 HTTP 错误信息不对：%v", err)
	}
}

func TestAxiosDefaultUserAgent(t *testing.T) {
	srv := echoServer(t)
	rt := newTestRuntime(t, `
		const axios = require("axios");
		module.exports = { platform: "ua", async ua() { return (await axios.get("`+srv.URL+`/echo")).data.ua; } };`, 0)
	if got := callString(t, rt, "ua"); !strings.HasPrefix(got, "SwiftPaw/") {
		t.Fatalf("默认 User-Agent 不对：%q", got)
	}
}
