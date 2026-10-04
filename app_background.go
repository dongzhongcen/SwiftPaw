package main

import "github.com/wailsapp/wails/v2/pkg/runtime"

// 自定义背景图片（图片的检查、复制和删除在 internal/core/background.go）

// SelectBackgroundImage 弹出选择文件的窗口，把选中的图片复制到数据文件夹，返回保存后的文件名。
// 用户取消选择时返回空字符串
func (a *App) SelectBackgroundImage() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择背景图片",
		Filters: []runtime.FileFilter{
			{DisplayName: "图片 (*.jpg;*.jpeg;*.png;*.webp;*.gif)", Pattern: "*.jpg;*.jpeg;*.png;*.webp;*.gif"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}
	return a.core.SetBackgroundImage(path)
}

// RemoveBackgroundImage 删掉保存的背景图片
func (a *App) RemoveBackgroundImage() error {
	return a.core.RemoveBackgroundImage()
}
