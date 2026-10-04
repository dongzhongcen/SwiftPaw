package jsrt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// maxResponseSize 是一次请求最多读取的响应大小，防止插件把内存吃光
const maxResponseSize = 32 << 20

// userAgent 是插件没有设置 User-Agent 时用的默认值
const userAgent = "SwiftPaw/1.0 (+https://github.com/dongzhongcen/SwiftPaw)"

// header 是一个请求头。用切片而不是 map，保持插件设置的顺序。
type header struct{ name, value string }

// httpRequest 是从 axios 配置里整理出来的请求，交给后台 goroutine 去发
type httpRequest struct {
	method       string
	url          string
	headers      []header
	body         []byte
	timeout      time.Duration
	responseType string
}

// newAxios 用 Go 实现 axios 的常用接口：axios(config)、axios(url, config)、
// axios.get/delete/head/options(url, config)、axios.post/put/patch(url, data, config)、
// axios.request(config)、axios.create(defaults)。
// defaults 是 create 传进来的默认配置，可以为 nil。
func (r *Runtime) newAxios(vm *goja.Runtime, defaults *goja.Object) *goja.Object {
	defaultsObj := vm.NewObject()
	if defaults != nil {
		for _, k := range defaults.Keys() {
			defaultsObj.Set(k, defaults.Get(k))
		}
	}
	if h, ok := defaultsObj.Get("headers").(*goja.Object); !ok || h == nil {
		headers := vm.NewObject()
		headers.Set("common", vm.NewObject())
		defaultsObj.Set("headers", headers)
	}

	request := func(config goja.Value) goja.Value {
		cfg := vm.NewObject()
		if obj, ok := config.(*goja.Object); ok {
			cfg = obj
		}
		return r.axiosRequest(vm, defaultsObj, cfg)
	}

	// axios 本身是一个函数：axios(config) 或 axios(url, config)
	instanceFn := func(call goja.FunctionCall) goja.Value {
		first := call.Argument(0)
		if s, ok := first.Export().(string); ok {
			cfg := toObject(vm, call.Argument(1))
			cfg.Set("url", s)
			return request(cfg)
		}
		return request(first)
	}
	instance := vm.ToValue(instanceFn).ToObject(vm)

	instance.Set("request", func(call goja.FunctionCall) goja.Value { return request(call.Argument(0)) })
	for _, method := range []string{"get", "delete", "head", "options"} {
		method := method
		instance.Set(method, func(call goja.FunctionCall) goja.Value {
			cfg := toObject(vm, call.Argument(1))
			cfg.Set("url", call.Argument(0))
			cfg.Set("method", method)
			return request(cfg)
		})
	}
	for _, method := range []string{"post", "put", "patch"} {
		method := method
		instance.Set(method, func(call goja.FunctionCall) goja.Value {
			cfg := toObject(vm, call.Argument(2))
			cfg.Set("url", call.Argument(0))
			cfg.Set("method", method)
			if data := call.Argument(1); !goja.IsUndefined(data) {
				cfg.Set("data", data)
			}
			return request(cfg)
		})
	}
	instance.Set("create", func(call goja.FunctionCall) goja.Value {
		return r.newAxios(vm, mergeConfig(vm, defaultsObj, toObject(vm, call.Argument(0))))
	})
	instance.Set("isAxiosError", func(call goja.FunctionCall) goja.Value {
		obj, ok := call.Argument(0).(*goja.Object)
		return vm.ToValue(ok && obj.Get("isAxiosError") != nil && obj.Get("isAxiosError").ToBoolean())
	})
	instance.Set("defaults", defaultsObj)
	// 拦截器只提供空实现，保证调用 axios.interceptors.request.use(...) 不会报错
	interceptors := vm.NewObject()
	for _, kind := range []string{"request", "response"} {
		m := vm.NewObject()
		m.Set("use", func(goja.FunctionCall) goja.Value { return vm.ToValue(0) })
		m.Set("eject", func(goja.FunctionCall) goja.Value { return goja.Undefined() })
		interceptors.Set(kind, m)
	}
	instance.Set("interceptors", interceptors)
	// TypeScript 编译出来的插件会写 axios_1.default.get(...)
	instance.Set("default", instance)
	return instance
}

// axiosRequest 合并配置、发请求，返回一个 Promise
func (r *Runtime) axiosRequest(vm *goja.Runtime, defaults, config *goja.Object) goja.Value {
	promise, resolve, reject := vm.NewPromise()
	merged := mergeConfig(vm, defaults, config)

	req, err := r.buildRequest(vm, merged)
	if err != nil {
		_ = reject(makeAxiosError(vm, err.Error(), "ERR_BAD_REQUEST", merged, nil))
		return vm.ToValue(promise)
	}

	// 网络请求在后台 goroutine 里做，做完再回到事件循环里 resolve/reject
	go func() {
		status, statusText, respHeaders, body, finalURL, err := r.doHTTP(req)
		r.loop.RunOnLoop(func(vm *goja.Runtime) {
			if err != nil {
				code := "ERR_NETWORK"
				if errors.Is(err, context.DeadlineExceeded) {
					code = "ECONNABORTED"
					err = fmt.Errorf("timeout of %dms exceeded", req.timeout.Milliseconds())
				}
				_ = reject(makeAxiosError(vm, err.Error(), code, merged, nil))
				return
			}
			resp := r.buildResponse(vm, merged, req, status, statusText, respHeaders, body, finalURL)
			if validStatus(vm, merged, status) {
				_ = resolve(resp)
				return
			}
			code := "ERR_BAD_REQUEST"
			if status >= 500 {
				code = "ERR_BAD_RESPONSE"
			}
			_ = reject(makeAxiosError(vm, fmt.Sprintf("Request failed with status code %d", status), code, merged, resp))
		})
	}()
	return vm.ToValue(promise)
}

// buildRequest 把 axios 配置变成 httpRequest：拼 URL 和查询参数、整理请求头、序列化请求体
func (r *Runtime) buildRequest(vm *goja.Runtime, cfg *goja.Object) (*httpRequest, error) {
	req := &httpRequest{method: "GET", timeout: r.opts.Timeout, responseType: "json"}
	if m := str(cfg.Get("method")); m != "" {
		req.method = strings.ToUpper(m)
	}
	if t := cfg.Get("timeout"); t != nil && !goja.IsUndefined(t) && t.ToInteger() > 0 {
		req.timeout = time.Duration(t.ToInteger()) * time.Millisecond
	}
	if rt := str(cfg.Get("responseType")); rt != "" {
		req.responseType = strings.ToLower(rt)
	}

	// URL：baseURL + url，再加上 params
	rawURL := str(cfg.Get("url"))
	if base := str(cfg.Get("baseURL")); base != "" && !strings.Contains(rawURL, "://") {
		rawURL = strings.TrimRight(base, "/") + "/" + strings.TrimLeft(rawURL, "/")
	}
	if query := serializeParams(vm, cfg); query != "" {
		if strings.Contains(rawURL, "?") {
			rawURL += "&" + query
		} else {
			rawURL += "?" + query
		}
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("只支持 http/https 地址：%q", rawURL)
	}
	req.url = u.String()

	// 请求头：defaults.headers.common → defaults.headers[方法] → 配置里的 headers
	req.headers = collectHeaders(cfg.Get("headers"), strings.ToLower(req.method))

	body, contentType, err := serializeData(vm, r, cfg.Get("data"), req.headers)
	if err != nil {
		return nil, err
	}
	req.body = body
	if contentType != "" && findHeader(req.headers, "Content-Type") == "" {
		req.headers = append(req.headers, header{"Content-Type", contentType})
	}
	if findHeader(req.headers, "Accept") == "" {
		req.headers = append(req.headers, header{"Accept", "application/json, text/plain, */*"})
	}
	return req, nil
}

// doHTTP 真正发请求（在后台 goroutine 里执行，不能碰 JS 虚拟机）
func (r *Runtime) doHTTP(req *httpRequest) (int, string, http.Header, []byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), req.timeout)
	defer cancel()
	var body io.Reader
	if req.body != nil {
		body = bytes.NewReader(req.body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.method, req.url, body)
	if err != nil {
		return 0, "", nil, nil, "", err
	}
	for _, h := range req.headers {
		httpReq.Header.Add(h.name, h.value)
	}
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", userAgent)
	}
	resp, err := r.client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return 0, "", nil, nil, "", context.DeadlineExceeded
		}
		return 0, "", nil, nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		if ctx.Err() != nil {
			return 0, "", nil, nil, "", context.DeadlineExceeded
		}
		return 0, "", nil, nil, "", err
	}
	statusText := strings.TrimSpace(strings.TrimPrefix(resp.Status, fmt.Sprint(resp.StatusCode)))
	return resp.StatusCode, statusText, resp.Header, data, resp.Request.URL.String(), nil
}

// buildResponse 生成 axios 的响应对象 {data, status, statusText, headers, config, request}
func (r *Runtime) buildResponse(vm *goja.Runtime, cfg *goja.Object, req *httpRequest, status int, statusText string,
	headers http.Header, body []byte, finalURL string) *goja.Object {
	h := vm.NewObject()
	for name, values := range headers {
		key := strings.ToLower(name)
		if key == "set-cookie" {
			h.Set(key, values)
		} else {
			h.Set(key, strings.Join(values, ", "))
		}
	}

	var data goja.Value
	switch req.responseType {
	case "arraybuffer":
		data = vm.ToValue(vm.NewArrayBuffer(body))
	case "text":
		data = vm.ToValue(string(body))
	default:
		// 和 axios 一样：能解析成 JSON 就解析，解析不了就保留字符串
		data = vm.ToValue(string(body))
		if parsed, err := r.jsonParse(goja.Undefined(), data); err == nil {
			data = parsed
		}
	}

	resp := vm.NewObject()
	resp.Set("data", data)
	resp.Set("status", status)
	resp.Set("statusText", statusText)
	resp.Set("headers", h)
	resp.Set("config", cfg)
	request := vm.NewObject()
	request.Set("responseURL", finalURL)
	resp.Set("request", request)
	return resp
}

func validStatus(vm *goja.Runtime, cfg *goja.Object, status int) bool {
	if fn, ok := goja.AssertFunction(cfg.Get("validateStatus")); ok {
		v, err := fn(goja.Undefined(), vm.ToValue(status))
		return err == nil && v.ToBoolean()
	}
	return status >= 200 && status < 300
}

// makeAxiosError 生成和 axios 一样的错误对象：message、code、config、response、isAxiosError
func makeAxiosError(vm *goja.Runtime, message, code string, cfg *goja.Object, resp *goja.Object) *goja.Object {
	ctor, _ := vm.Get("Error").(*goja.Object)
	errObj, err := vm.New(ctor, vm.ToValue(message))
	if err != nil {
		errObj = vm.NewObject()
		errObj.Set("message", message)
	}
	errObj.Set("name", "AxiosError")
	errObj.Set("code", code)
	errObj.Set("config", cfg)
	errObj.Set("isAxiosError", true)
	if resp != nil {
		errObj.Set("response", resp)
		errObj.Set("status", resp.Get("status"))
	}
	return errObj
}

// mergeConfig 合并默认配置和本次配置，headers 字段单独合并
func mergeConfig(vm *goja.Runtime, defaults, config *goja.Object) *goja.Object {
	out := vm.NewObject()
	for _, src := range []*goja.Object{defaults, config} {
		if src == nil {
			continue
		}
		for _, k := range src.Keys() {
			if k != "headers" {
				out.Set(k, src.Get(k))
			}
		}
	}
	headers := vm.NewObject()
	for _, src := range []*goja.Object{defaults, config} {
		if src == nil {
			continue
		}
		if h, ok := src.Get("headers").(*goja.Object); ok {
			for _, k := range h.Keys() {
				headers.Set(k, h.Get(k))
			}
		}
	}
	out.Set("headers", headers)
	return out
}

// collectHeaders 整理请求头。axios 的 headers 里可能有 common / get / post 这种按方法分组的默认值。
func collectHeaders(v goja.Value, method string) []header {
	obj, ok := v.(*goja.Object)
	if !ok {
		return nil
	}
	groups := map[string]bool{"common": true, "get": true, "post": true, "put": true, "patch": true,
		"delete": true, "head": true, "options": true}
	var list []header
	add := func(o *goja.Object) {
		for _, k := range o.Keys() {
			val := o.Get(k)
			if groups[strings.ToLower(k)] || val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
				continue
			}
			list = setHeader(list, k, val.String())
		}
	}
	for _, group := range []string{"common", method} {
		if g, ok := obj.Get(group).(*goja.Object); ok {
			add(g)
		}
	}
	add(obj)
	return list
}

// setHeader 设置请求头（名字不区分大小写，后设置的覆盖前面的）
func setHeader(list []header, name, value string) []header {
	for i, h := range list {
		if strings.EqualFold(h.name, name) {
			list[i].value = value
			return list
		}
	}
	return append(list, header{name, value})
}

func findHeader(list []header, name string) string {
	for _, h := range list {
		if strings.EqualFold(h.name, name) {
			return h.value
		}
	}
	return ""
}

// serializeParams 把 params 对象变成查询字符串，保持插件写的顺序（有些接口的签名依赖参数顺序）
func serializeParams(vm *goja.Runtime, cfg *goja.Object) string {
	params := cfg.Get("params")
	if params == nil || goja.IsUndefined(params) || goja.IsNull(params) {
		return ""
	}
	if fn, ok := goja.AssertFunction(cfg.Get("paramsSerializer")); ok {
		if v, err := fn(goja.Undefined(), params); err == nil {
			return v.String()
		}
	}
	if s, ok := params.Export().(string); ok {
		return s
	}
	obj, ok := params.(*goja.Object)
	if !ok {
		return ""
	}
	if isURLSearchParams(vm, obj) {
		return obj.String()
	}
	return encodeForm(vm, obj, true)
}

// encodeForm 编码成 a=1&b=2。brackets 为 true 时数组写成 key[]=v（axios 的默认写法）
func encodeForm(vm *goja.Runtime, obj *goja.Object, brackets bool) string {
	var parts []string
	for _, k := range obj.Keys() {
		v := obj.Get(k)
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			continue
		}
		if arr, ok := v.(*goja.Object); ok && arr.ClassName() == "Array" {
			key := k
			if brackets {
				key += "[]"
			}
			for _, item := range arr.Keys() {
				parts = append(parts, axiosEncode(key)+"="+axiosEncode(paramString(vm, arr.Get(item))))
			}
			continue
		}
		parts = append(parts, axiosEncode(k)+"="+axiosEncode(paramString(vm, v)))
	}
	return strings.Join(parts, "&")
}

// paramString：对象用 JSON，其他用 String()
func paramString(vm *goja.Runtime, v goja.Value) string {
	if obj, ok := v.(*goja.Object); ok && obj.ClassName() != "Date" {
		stringify, _ := goja.AssertFunction(vm.Get("JSON").ToObject(vm).Get("stringify"))
		if s, err := stringify(goja.Undefined(), v); err == nil {
			return s.String()
		}
	}
	return v.String()
}

// serializeData 序列化请求体，返回内容和默认的 Content-Type
func serializeData(vm *goja.Runtime, r *Runtime, data goja.Value, headers []header) ([]byte, string, error) {
	if data == nil || goja.IsUndefined(data) || goja.IsNull(data) {
		return nil, "", nil
	}
	switch v := data.Export().(type) {
	case string:
		// axios 对字符串请求体默认用表单类型
		return []byte(v), "application/x-www-form-urlencoded", nil
	case goja.ArrayBuffer:
		return v.Bytes(), "application/octet-stream", nil
	case []byte:
		return v, "application/octet-stream", nil
	}
	obj, ok := data.(*goja.Object)
	if !ok {
		return []byte(data.String()), "", nil
	}
	if isURLSearchParams(vm, obj) {
		return []byte(obj.String()), "application/x-www-form-urlencoded;charset=utf-8", nil
	}
	if strings.Contains(strings.ToLower(findHeader(headers, "Content-Type")), "x-www-form-urlencoded") {
		return []byte(encodeForm(vm, obj, false)), "", nil
	}
	s, err := r.jsonStringify(goja.Undefined(), data)
	if err != nil {
		return nil, "", err
	}
	return []byte(s.String()), "application/json", nil
}

func isURLSearchParams(vm *goja.Runtime, obj *goja.Object) bool {
	ctor, ok := vm.Get("URLSearchParams").(*goja.Object)
	return ok && vm.InstanceOf(obj, ctor)
}

// axiosEncode 和 axios 的 encode 一样：encodeURIComponent 之后把 : $ , [ ] 换回来，空格变成 +
func axiosEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			strings.IndexByte("-_.!~*'():$,[]@", c) >= 0:
			b.WriteByte(c)
		case c == ' ':
			b.WriteByte('+')
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func toObject(vm *goja.Runtime, v goja.Value) *goja.Object {
	obj := vm.NewObject()
	if src, ok := v.(*goja.Object); ok {
		for _, k := range src.Keys() {
			obj.Set(k, src.Get(k))
		}
	}
	return obj
}

func str(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	return v.String()
}

// onlyHTTPRedirects 只允许跳转到 http/https 地址，最多 10 次
func onlyHTTPRedirects(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("重定向次数太多")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("不允许重定向到 %s 地址", req.URL.Scheme)
	}
	return nil
}
