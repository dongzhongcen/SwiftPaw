package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaultsForNewFields(t *testing.T) {
	c, dir := newCore(t)
	// 旧版本的配置文件里只有 theme，没有风格和背景
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"theme":"light","volume":0.5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := c.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Theme != "light" || config.Style != "" || config.ReduceBlur != "" {
		t.Fatalf("旧配置读出来不对：%+v", config)
	}
	if config.Background != (BackgroundConfig{Dim: 40}) {
		t.Fatalf("背景的默认设置不对：%+v", config.Background)
	}
}

func TestConfigStyleAndBackgroundRoundTrip(t *testing.T) {
	c, _ := newCore(t)
	config, _ := c.LoadConfig()
	config.Theme = "system"
	config.Style = "neon"
	config.ReduceBlur = "on"
	config.Background = BackgroundConfig{Image: "background-0123456789abcdef.webp", Blur: 12, Dim: 55}
	if err := c.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	got, err := c.LoadConfig()
	if err != nil || got != config {
		t.Fatalf("读回来的配置不一样：%+v %v", got, err)
	}
}

func TestConfigNormalize(t *testing.T) {
	c, dir := newCore(t)
	bad := `{"theme":"purple","style":"rainbow","reduceBlur":"maybe",
		"background":{"image":"../../etc/passwd","blur":99,"dim":-5}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := c.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	want := BackgroundConfig{Image: "", Blur: 30, Dim: 0}
	if config.Theme != "dark" || config.Style != "" || config.ReduceBlur != "" || config.Background != want {
		t.Fatalf("不认识的值应该换成默认值：%+v", config)
	}

	// 保存时也一样检查
	config.Style = "vinyl"
	config.Background = BackgroundConfig{Blur: -1, Dim: 100}
	if err := c.SaveConfig(config); err != nil {
		t.Fatal(err)
	}
	got, _ := c.LoadConfig()
	if got.Style != "vinyl" || got.Background.Blur != 0 || got.Background.Dim != 80 {
		t.Fatalf("保存时没有限制范围：%+v", got)
	}
}

func TestConfigBrokenJSON(t *testing.T) {
	c, dir := newCore(t)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"theme":`), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := c.LoadConfig()
	if err == nil {
		t.Fatal("坏掉的配置文件应该返回错误")
	}
	if config != defaultConfig() {
		t.Fatalf("坏掉的配置文件应该返回默认配置：%+v", config)
	}
}
