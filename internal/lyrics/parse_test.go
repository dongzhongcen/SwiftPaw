package lyrics

import (
	"reflect"
	"testing"
)

func TestParseTimestampsAndSort(t *testing.T) {
	text := "[ti:海边小路]\n[ar:测试歌手]\n[by:someone]\n" +
		"[00:12.30]第一句测试歌词\r\n" +
		"[01:40.00][00:20.5]第二句测试歌词\n" +
		"[00:05]前奏\n" +
		"[02:03.456]三位毫秒\n" +
		"[00:30.00]\n"
	got := Parse(text)
	want := []Line{
		{5, "前奏"},
		{12.3, "第一句测试歌词"},
		{20.5, "第二句测试歌词"},
		{30, ""},
		{100, "第二句测试歌词"},
		{123.456, "三位毫秒"},
	}
	if got.Plain {
		t.Fatal("有时间轴的歌词不应该是 Plain")
	}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Fatalf("解析结果不对：\n得到 %v\n期望 %v", got.Lines, want)
	}
}

func TestParseOffset(t *testing.T) {
	// offset 为正数表示歌词整体提前
	got := Parse("[offset:+500]\n[00:01.00]一\n[00:00.20]零")
	want := []Line{{0, "零"}, {0.5, "一"}}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Fatalf("offset 处理不对：%v", got.Lines)
	}

	got = Parse("[offset:-1000]\n[00:01.00]一")
	if got.Lines[0].Time != 2 {
		t.Fatalf("负 offset 应该推后：%v", got.Lines)
	}
}

func TestParsePlainText(t *testing.T) {
	got := Parse("[ti:无时间轴]\n第一行\n\n第二行\n")
	if !got.Plain {
		t.Fatal("没有时间标签时应该是 Plain")
	}
	want := []Line{{0, "第一行"}, {0, "第二行"}}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Fatalf("纯文本歌词不对：%v", got.Lines)
	}
}

func TestParseEmpty(t *testing.T) {
	got := Parse("")
	if got.Plain || len(got.Lines) != 0 {
		t.Fatalf("空歌词：%+v", got)
	}
}

func TestParseWordTagsAndBrackets(t *testing.T) {
	got := Parse("[00:01.00]<00:01.00>逐<00:01.50>字\n[00:02.00][合唱]大家一起")
	want := []Line{{1, "逐字"}, {2, "[合唱]大家一起"}}
	if !reflect.DeepEqual(got.Lines, want) {
		t.Fatalf("得到 %v", got.Lines)
	}
}

func TestParseTime(t *testing.T) {
	cases := map[string]float64{
		"00:00":     0,
		"01:02":     62,
		"01:02.5":   62.5,
		"01:02.50":  62.5,
		"01:02.500": 62.5,
		"01:02:05":  62.05,
		"100:00.00": 6000,
	}
	for in, want := range cases {
		got, ok := parseTime(in)
		if !ok || got != want {
			t.Errorf("parseTime(%q) = %v, %v；期望 %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"ti:abc", "1:2:3:4", "aa:bb", ""} {
		if _, ok := parseTime(bad); ok {
			t.Errorf("parseTime(%q) 不应该成功", bad)
		}
	}
}
