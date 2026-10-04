package jsrt

import "testing"

func TestCryptoJS(t *testing.T) {
	rt := newTestRuntime(t, `
		const CryptoJS = require("crypto-js");
		module.exports = {
			platform: "crypto",
			md5(s) { return CryptoJS.MD5(s).toString(); },
			sha256(s) { return CryptoJS.SHA256(s).toString(CryptoJS.enc.Hex); },
			aesRoundTrip(text) {
				// 用口令加密：需要随机盐（crypto.getRandomValues）
				const cipher = CryptoJS.AES.encrypt(text, "secret").toString();
				return CryptoJS.AES.decrypt(cipher, "secret").toString(CryptoJS.enc.Utf8);
			},
			aesFixed() {
				// 固定 key 和 iv（很多接口这样加密参数），结果可以和其他语言对照
				const key = CryptoJS.enc.Utf8.parse("0123456789abcdef");
				const iv = CryptoJS.enc.Utf8.parse("fedcba9876543210");
				return CryptoJS.AES.encrypt("hello", key, { iv, mode: CryptoJS.mode.CBC, padding: CryptoJS.pad.Pkcs7 }).toString();
			},
			base64(s) { return CryptoJS.enc.Base64.stringify(CryptoJS.enc.Utf8.parse(s)); },
		};`, 0)
	if got := callString(t, rt, "md5", "hello"); got != "5d41402abc4b2a76b9719d911017c592" {
		t.Errorf("MD5 不对：%s", got)
	}
	if got := callString(t, rt, "sha256", "abc"); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("SHA256 不对：%s", got)
	}
	if got := callString(t, rt, "aesRoundTrip", "极拍 SwiftPaw"); got != "极拍 SwiftPaw" {
		t.Errorf("AES 加解密不一致：%s", got)
	}
	// 期望值用 openssl 算出来：echo -n hello | openssl enc -aes-128-cbc -K 3031..66 -iv 6665..30 -base64
	if got := callString(t, rt, "aesFixed"); got != "e+3OsGVfOJQQaXifNkYgkQ==" {
		t.Errorf("AES-CBC 结果不对：%s", got)
	}
	if got := callString(t, rt, "base64", "你好"); got != "5L2g5aW9" {
		t.Errorf("Base64 不对：%s", got)
	}
}

func TestBundledLibs(t *testing.T) {
	rt := newTestRuntime(t, `
		const qs = require("qs");
		const dayjs = require("dayjs");
		const he = require("he");
		const bigInt = require("big-integer");
		const cheerio = require("cheerio");
		module.exports = {
			platform: "libs",
			run() {
				const $ = cheerio.load('<ul id="list"><li class="song" data-id="1">晴天</li><li class="song" data-id="2">稻香</li></ul>');
				return [
					qs.stringify({ a: 1, b: [1, 2], c: { d: "你" } }),
					JSON.stringify(qs.parse("x=1&y[]=2&y[]=3")),
					dayjs("2026-10-04T08:00:00Z").add(1, "day").toISOString(),
					he.decode("&lt;b&gt;&amp;&#20320;"),
					bigInt("123456789012345678901234567890").add(1).toString(),
					$(".song").map((i, el) => $(el).attr("data-id") + ":" + $(el).text()).get().join(","),
				].join(" | ");
			},
		};`, 0)
	want := "a=1&b%5B0%5D=1&b%5B1%5D=2&c%5Bd%5D=%E4%BD%A0 | {\"x\":\"1\",\"y\":[\"2\",\"3\"]} | 2026-10-05T08:00:00.000Z | <b>&你 | 123456789012345678901234567891 | 1:晴天,2:稻香"
	if got := callString(t, rt, "run"); got != want {
		t.Fatalf("内置库结果不对：\n得到 %s\n期望 %s", got, want)
	}
}
