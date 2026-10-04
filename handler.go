package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

// MusicHandler 负责把本地 mp3 文件和封面图片提供给前端
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
	if !isMP3(path) {
		http.Error(w, "只允许播放 mp3", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, path)
}

func (h *MusicHandler) serveCover(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !isMP3(path) {
		http.Error(w, "只允许读取 mp3 封面", http.StatusForbidden)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

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

func isMP3(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".mp3"
}
