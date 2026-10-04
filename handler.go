package main

import (
	"net/http"
	"os"

	"github.com/dhowden/tag"

	"musicplayer/internal/music"
)

// MusicHandler 负责把本地音频文件和封面图片提供给前端
type MusicHandler struct{}

func NewMusicHandler() *MusicHandler {
	return &MusicHandler{}
}

func (h *MusicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/music":
		h.serveMusic(w, r)
	case "/cover":
		h.serveCover(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *MusicHandler) serveMusic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !music.IsAudio(path) {
		http.Error(w, "只允许播放音频文件", http.StatusForbidden)
		return
	}

	// 明确告诉 WebView 文件类型，避免 Windows 上 .flac、.opus 这类扩展名识别不出来
	w.Header().Set("Content-Type", music.ContentType(path))
	http.ServeFile(w, r, path)
}

func (h *MusicHandler) serveCover(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !music.IsAudio(path) {
		http.Error(w, "只允许读取音频文件的封面", http.StatusForbidden)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// tag 库能读 mp3、flac、m4a、ogg 里内嵌的封面
	metadata, err := tag.ReadFrom(file)
	if err != nil || metadata.Picture() == nil {
		http.NotFound(w, r)
		return
	}

	picture := metadata.Picture()
	w.Header().Set("Content-Type", picture.MIMEType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(picture.Data)
}
