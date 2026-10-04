//go:build tools

// 这个文件不参与编译，只是让 go.mod 记住 golang.org/x/mobile 的版本：
// gomobile bind 生成的代码要用到 golang.org/x/mobile/bind，go mod tidy 时不能把它删掉。
package gocore

import _ "golang.org/x/mobile/bind"
