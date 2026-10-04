package core

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path string, data []byte) string {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = byte(i * 7)
	}
	return img
}

func encodePNG(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, testImage()); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSetBackgroundImageFormats(t *testing.T) {
	c, dir := newCore(t)
	src := t.TempDir()

	var jpg, gifData bytes.Buffer
	if err := jpeg.Encode(&jpg, testImage(), nil); err != nil {
		t.Fatal(err)
	}
	pal := image.NewPaletted(image.Rect(0, 0, 4, 4), []color.Color{color.Black, color.White})
	if err := gif.Encode(&gifData, pal, nil); err != nil {
		t.Fatal(err)
	}
	// 最小的 WebP 文件头：RIFF <大小> WEBP VP8L ...，只判断文件头，内容不用完整
	webp := append([]byte("RIFF\x1a\x00\x00\x00WEBPVP8L"), make([]byte, 14)...)

	cases := []struct {
		file string
		data []byte
		ext  string
	}{
		{"a.png", encodePNG(t), ".png"},
		{"b.jpeg", jpg.Bytes(), ".jpg"},
		{"c.gif", gifData.Bytes(), ".gif"},
		{"d.webp", webp, ".webp"},
		// Android 的文件选择器给的文件名可能没有扩展名，按内容判断
		{"noext", encodePNG(t), ".png"},
	}
	for _, tc := range cases {
		name, err := c.SetBackgroundImage(writeFile(t, filepath.Join(src, tc.file), tc.data))
		if err != nil {
			t.Fatalf("%s：%v", tc.file, err)
		}
		if !validBackgroundName(name) || filepath.Ext(name) != tc.ext {
			t.Fatalf("%s 保存后的文件名不对：%s", tc.file, name)
		}
		saved, err := os.ReadFile(filepath.Join(dir, "backgrounds", name))
		if err != nil || !bytes.Equal(saved, tc.data) {
			t.Fatalf("%s 没有原样复制到数据文件夹：%v", tc.file, err)
		}
		// 以前的背景图片应该被删掉，只留下最新的一张
		entries, _ := os.ReadDir(filepath.Join(dir, "backgrounds"))
		if len(entries) != 1 {
			t.Fatalf("%s 之后 backgrounds 里应该只有一个文件，实际 %d 个", tc.file, len(entries))
		}
		if got := BackgroundContentType(name); !strings.HasPrefix(got, "image/") {
			t.Fatalf("%s 的 MIME 类型不对：%q", name, got)
		}
	}
}

func TestSetBackgroundImageCopiesFile(t *testing.T) {
	c, _ := newCore(t)
	src := writeFile(t, filepath.Join(t.TempDir(), "bg.png"), encodePNG(t))
	name, err := c.SetBackgroundImage(src)
	if err != nil {
		t.Fatal(err)
	}
	// 删掉原图后，保存的副本还在
	if err := os.Remove(src); err != nil {
		t.Fatal(err)
	}
	path, err := c.BackgroundImagePath(name)
	if err != nil {
		t.Fatalf("删掉原图后应该还能读到副本：%v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	// 同一张图再选一次，文件名不变
	src2 := writeFile(t, filepath.Join(t.TempDir(), "again.png"), encodePNG(t))
	name2, err := c.SetBackgroundImage(src2)
	if err != nil || name2 != name {
		t.Fatalf("同一张图应该得到同一个文件名：%s %s %v", name, name2, err)
	}
}

func TestSetBackgroundImageRejects(t *testing.T) {
	c, _ := newCore(t)
	src := t.TempDir()

	if _, err := c.SetBackgroundImage(writeFile(t, filepath.Join(src, "fake.png"), []byte("not an image"))); !errors.Is(err, ErrNotImage) {
		t.Fatalf("不是图片的文件应该被拒绝：%v", err)
	}
	if _, err := c.SetBackgroundImage(writeFile(t, filepath.Join(src, "x.svg"), []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"))); !errors.Is(err, ErrNotImage) {
		t.Fatalf("SVG 应该被拒绝：%v", err)
	}
	if _, err := c.SetBackgroundImage(src); !errors.Is(err, ErrNotImage) {
		t.Fatalf("文件夹应该被拒绝：%v", err)
	}
	if _, err := c.SetBackgroundImage(filepath.Join(src, "没有这个文件.png")); err == nil {
		t.Fatal("不存在的文件应该返回错误")
	}

	// 超过 20 MB：用稀疏文件，不真的占磁盘
	big := filepath.Join(src, "big.png")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(encodePNG(t))
	if err := f.Truncate(MaxBackgroundSize + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := c.SetBackgroundImage(big); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("超过 20 MB 的图片应该被拒绝：%v", err)
	}

	// 正好 20 MB 可以
	if err := os.Truncate(big, MaxBackgroundSize); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SetBackgroundImage(big); err != nil {
		t.Fatalf("正好 20 MB 的图片应该可以：%v", err)
	}
}

func TestRemoveBackgroundImage(t *testing.T) {
	c, dir := newCore(t)
	// 还没设置过背景时删除也不报错
	if err := c.RemoveBackgroundImage(); err != nil {
		t.Fatal(err)
	}
	name, err := c.SetBackgroundImage(writeFile(t, filepath.Join(t.TempDir(), "bg.png"), encodePNG(t)))
	if err != nil {
		t.Fatal(err)
	}
	// backgrounds 里别的文件（不是背景图片的）不动
	other := writeFile(t, filepath.Join(dir, "backgrounds", "notes.txt"), []byte("keep"))
	if err := c.RemoveBackgroundImage(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.BackgroundImagePath(name); err == nil {
		t.Fatal("删除后不应该还能找到背景图片")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("不应该删掉别的文件：%v", err)
	}
}

func TestBackgroundImagePathRejectsOtherFiles(t *testing.T) {
	c, dir := newCore(t)
	writeFile(t, filepath.Join(dir, "config.json"), []byte("{}"))
	for _, name := range []string{
		"", "config.json", "../config.json", "background-0123456789abcdef.png/../../config.json",
		"background-0123456789ABCDEF.png", "background-0123456789abcdef.svg", "/etc/passwd",
	} {
		if _, err := c.BackgroundImagePath(name); err == nil {
			t.Fatalf("%q 不应该被当成背景图片", name)
		}
	}
}

func TestBackgroundWithoutDataDir(t *testing.T) {
	c := New(Options{})
	defer c.Close()
	src := writeFile(t, filepath.Join(t.TempDir(), "bg.png"), encodePNG(t))
	if _, err := c.SetBackgroundImage(src); !errors.Is(err, errNoDataDir) {
		t.Fatalf("没有数据文件夹时应该返回错误：%v", err)
	}
	if err := c.RemoveBackgroundImage(); !errors.Is(err, errNoDataDir) {
		t.Fatalf("没有数据文件夹时应该返回错误：%v", err)
	}
}

func TestBackgroundHandler(t *testing.T) {
	c, dir := newCore(t)
	data := encodePNG(t)
	name, err := c.SetBackgroundImage(writeFile(t, filepath.Join(t.TempDir(), "bg.png"), data))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "config.json"), []byte(`{"secret":true}`))
	h := c.BackgroundHandler()

	get := func(query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/background?"+query, nil))
		return rec
	}
	rec := get("name=" + url.QueryEscape(name))
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), data) {
		t.Fatalf("读背景图片失败：%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type 不对：%s", ct)
	}
	for _, q := range []string{"", "name=config.json", "name=" + url.QueryEscape("../config.json"), "name=" + url.QueryEscape("/etc/passwd")} {
		if rec := get(q); rec.Code != 404 {
			t.Fatalf("%q 应该返回 404，实际 %d", q, rec.Code)
		}
	}
	if err := c.RemoveBackgroundImage(); err != nil {
		t.Fatal(err)
	}
	if rec := get("name=" + name); rec.Code != 404 {
		t.Fatalf("删除后应该返回 404，实际 %d", rec.Code)
	}
}
