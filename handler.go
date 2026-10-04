package main

import (
	"net/http"

	"musicplayer/internal/music"
)

// MusicHandler 负责把本地音频文件、封面图片、背景图片和在线歌曲（通过代理）提供给前端
type MusicHandler struct {
	stream     http.Handler // 在线歌曲的播放代理
	background http.Handler // 自定义背景图片（数据文件夹里的副本）
}

func NewMusicHandler(stream, background http.Handler) *MusicHandler {
	return &MusicHandler{stream: stream, background: background}
}

func (h *MusicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/music":
		h.serveMusic(w, r)
	case "/cover":
		h.serveCover(w, r)
	case "/stream":
		h.stream.ServeHTTP(w, r)
	case "/background":
		h.background.ServeHTTP(w, r)
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
	picture, err := music.ReadCover(r.URL.Query().Get("path"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", picture.MIME)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(picture.Data)
}
