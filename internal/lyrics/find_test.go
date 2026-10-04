package lyrics

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecode(t *testing.T) {
	gbk, _ := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("[00:01.00]你好"))
	if got := Decode(gbk); got != "[00:01.00]你好" {
		t.Errorf("GBK 解码失败：%q", got)
	}
	if got := Decode(append([]byte{0xEF, 0xBB, 0xBF}, "你好"...)); got != "你好" {
		t.Errorf("UTF-8 BOM：%q", got)
	}
	if got := Decode([]byte("普通 UTF-8")); got != "普通 UTF-8" {
		t.Errorf("UTF-8：%q", got)
	}
	utf16 := []byte{0xFF, 0xFE, 0x60, 0x4F, 0x7D, 0x59} // “你好”的 UTF-16LE
	if got := Decode(utf16); got != "你好" {
		t.Errorf("UTF-16：%q", got)
	}
}

func TestFindLrcFile(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "海边小路.mp3")
	write(t, audio, []byte("not really mp3"))
	gbk, _ := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("[ti:海边小路]\n[00:01.00]第一句测试歌词"))
	write(t, filepath.Join(dir, "海边小路.LRC"), gbk) // 大写扩展名 + GBK 编码

	got, err := Find(audio)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "lrc" || len(got.Lines) != 1 || got.Lines[0].Text != "第一句测试歌词" {
		t.Fatalf("没有正确读取同名 LRC：%+v", got)
	}
}

func TestFindEmbedded(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "song.mp3")
	write(t, audio, id3WithLyrics("[00:02.00]内嵌歌词\n[00:01.00]第一句"))

	got, err := Find(audio)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "embedded" || len(got.Lines) != 2 || got.Lines[0].Text != "第一句" {
		t.Fatalf("没有正确读取内嵌歌词：%+v", got)
	}
}

func TestFindNothing(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "none.mp3")
	write(t, audio, []byte("x"))
	got, err := Find(audio)
	if err != nil || got.Source != "" || len(got.Lines) != 0 {
		t.Fatalf("没有歌词时应该返回空：%+v, %v", got, err)
	}
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// id3WithLyrics 手工拼一个只有 USLT（歌词）帧的 ID3v2.4 标签，后面跟一点假的音频数据
func id3WithLyrics(text string) []byte {
	var frame bytes.Buffer
	frame.WriteByte(3)       // 文字编码：3 = UTF-8
	frame.WriteString("chi") // 语言
	frame.WriteByte(0)       // 描述（空字符串）结尾的 0
	frame.WriteString(text)  // 歌词正文
	body := frame.Bytes()

	var tagBody bytes.Buffer
	tagBody.WriteString("USLT")
	tagBody.Write(syncsafe(len(body)))
	tagBody.Write([]byte{0, 0}) // 帧标志
	tagBody.Write(body)

	var out bytes.Buffer
	out.WriteString("ID3")
	out.Write([]byte{4, 0, 0}) // 版本 2.4.0，标志 0
	out.Write(syncsafe(tagBody.Len()))
	out.Write(tagBody.Bytes())
	out.Write(make([]byte, 128)) // 假的音频数据
	return out.Bytes()
}

// syncsafe 是 ID3v2 的长度写法：每个字节只用低 7 位
func syncsafe(n int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32((n&0x7f)|(n&0x3f80)<<1|(n&0x1fc000)<<2|(n&0xfe00000)<<3))
	return b
}
