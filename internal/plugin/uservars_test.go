package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// keyPlugin 是测试用插件：声明了用户变量，没填 key 时搜索会报错，填了以后把 key 放进搜索结果里
const keyPlugin = `
const vars = env.getUserVariables(); // 有的插件在加载时就读设置，重新加载插件后才能拿到新值
module.exports = {
	platform: "需要密钥",
	version: "VERSION",
	userVariables: [
		{ key: "key", name: "API Key", hint: "填你自己申请的 key" },
		{ key: "region" },                 // 没有 name 和 hint
		{ name: "没有 key，应该被跳过" },
		{ key: "key", name: "重复的 key，应该被跳过" },
		"格式不对",
	],
	async search(query) {
		const key = env.getUserVariables().key;
		if (!key) throw new Error("请在插件设置-用户变量中填写 API Key (key)");
		return { isEnd: true, data: [{ id: "1", title: query + "-" + key + "-" + (vars.region || "无") }] };
	},
};`

func keyPluginSource(version string) string {
	return strings.Replace(keyPlugin, "VERSION", version, 1)
}

func TestParseUserVariables(t *testing.T) {
	dir := t.TempDir()
	m := newManager(t, dir)
	writePlugin(t, filepath.Join(dir, "plugins", "k.js"), keyPluginSource("1"))
	writePlugin(t, filepath.Join(dir, "plugins", "plain.js"), `module.exports = { platform: "普通" };`)
	m.LoadAll()

	list := m.List()
	got, _ := json.Marshal(list[0].UserVariables)
	want := `[{"key":"key","name":"API Key","hint":"填你自己申请的 key"},{"key":"region","name":"region","hint":""}]`
	if string(got) != want {
		t.Fatalf("用户变量解析不对：\n得到 %s\n期望 %s", got, want)
	}
	// 没有声明用户变量的插件是空列表（前端据此不显示“设置”按钮），不是 null
	if plain, _ := json.Marshal(list[1].UserVariables); string(plain) != "[]" {
		t.Fatalf("没有用户变量时应该是 []：%s", plain)
	}
	if _, err := m.SetUserVariables("plain", map[string]string{"key": "x"}); err == nil {
		t.Fatal("没有声明用户变量的插件不能保存设置")
	}
}

func TestUserVariablesSaveReloadUninstall(t *testing.T) {
	dir := t.TempDir()
	m := newManager(t, dir)
	src := filepath.Join(dir, "下载的.js")
	writePlugin(t, src, keyPluginSource("1"))
	info, err := m.InstallFile(src)
	if err != nil {
		t.Fatal(err)
	}

	// 没填 key：插件自己报错
	if _, err := m.Search("需要密钥", "歌", 1, ""); err == nil || !strings.Contains(err.Error(), "用户变量") {
		t.Fatalf("没填 key 时应该报插件自己的错误：%v", err)
	}

	// 保存后马上生效（插件被重新加载），没声明的变量不会被保存，前后空格会去掉
	saved, err := m.SetUserVariables(info.ID, map[string]string{"key": "  test-key-123 ", "region": "cn", "other": "x"})
	if err != nil || saved.Error != "" {
		t.Fatalf("保存设置失败：%+v %v", saved, err)
	}
	res, err := m.Search("需要密钥", "歌", 1, "")
	if err != nil || len(res.Data) != 1 || res.Data[0].Title != "歌-test-key-123-cn" {
		t.Fatalf("填了 key 以后应该能搜索：%+v %v", res, err)
	}
	values, _ := m.UserVariables(info.ID)
	if len(values) != 2 || values["key"] != "test-key-123" || values["other"] != "" {
		t.Fatalf("读出来的设置不对：%v", values)
	}

	// 设置保存在 plugins.json，重新打开（相当于重启应用）还在
	m2 := newManager(t, dir)
	if res, err := m2.Search("需要密钥", "歌", 1, ""); err != nil || res.Data[0].Title != "歌-test-key-123-cn" {
		t.Fatalf("重启后设置应该还在：%+v %v", res, err)
	}
	// 重新加载、更新插件（安装新版本）后设置也保留
	if err := m2.LoadAll(); err != nil {
		t.Fatal(err)
	}
	writePlugin(t, src, keyPluginSource("2"))
	if info2, err := m2.InstallFile(src); err != nil || info2.Version != "2" {
		t.Fatalf("更新插件失败：%+v %v", info2, err)
	}
	if res, err := m2.Search("需要密钥", "歌", 1, ""); err != nil || res.Data[0].Title != "歌-test-key-123-cn" {
		t.Fatalf("更新插件后设置应该还在：%+v %v", res, err)
	}

	// 清空 region 只保留 key
	if _, err := m2.SetUserVariables(info.ID, map[string]string{"key": "test-key-123", "region": ""}); err != nil {
		t.Fatal(err)
	}
	if res, _ := m2.Search("需要密钥", "歌", 1, ""); res.Data[0].Title != "歌-test-key-123-无" {
		t.Fatalf("清空的设置不应该再传给插件：%+v", res)
	}

	// 卸载后设置被删除，plugins.json 里也没有了
	if err := m2.Uninstall(info.ID); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "plugins.json"))
	if strings.Contains(string(data), "test-key-123") {
		t.Fatalf("卸载后 plugins.json 里不应该还有设置：%s", data)
	}
	// 重新安装是全新的插件，需要重新填
	if _, err := m2.InstallFile(src); err != nil {
		t.Fatal(err)
	}
	if values, _ := m2.UserVariables(info.ID); len(values) != 0 {
		t.Fatalf("重新安装后不应该还有旧设置：%v", values)
	}
	if _, err := m2.Search("需要密钥", "歌", 1, ""); err == nil {
		t.Fatal("重新安装后没填 key，搜索应该报错")
	}
}

func TestUserVariablesMovedOnUpdate(t *testing.T) {
	dir := t.TempDir()
	m := newManager(t, dir)
	// 手动复制进来的插件，文件名和平台名不一样
	writePlugin(t, filepath.Join(dir, "plugins", "手动.js"), keyPluginSource("1"))
	m.LoadAll()
	if _, err := m.SetUserVariables("手动", map[string]string{"key": "k1"}); err != nil {
		t.Fatal(err)
	}
	// 从文件安装新版本：文件名变成平台名，设置跟着搬过去
	src := filepath.Join(dir, "新版.js")
	writePlugin(t, src, keyPluginSource("2"))
	info, err := m.InstallFile(src)
	if err != nil || info.ID != "需要密钥" {
		t.Fatalf("更新失败：%+v %v", info, err)
	}
	if values, _ := m.UserVariables(info.ID); values["key"] != "k1" {
		t.Fatalf("设置应该搬到新 id 下：%v", values)
	}
	if res, err := m.Search("需要密钥", "歌", 1, ""); err != nil || res.Data[0].Title != "歌-k1-无" {
		t.Fatalf("更新后应该还能用原来的设置搜索：%+v %v", res, err)
	}
}
