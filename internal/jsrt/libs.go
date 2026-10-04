package jsrt

import (
	_ "embed"
	"sync"

	"github.com/dop251/goja"
)

// libs.js 是用 esbuild 打包好的 crypto-js、qs、dayjs、he、big-integer、cheerio，
// 由 scripts/jslib/build.mjs 生成。许可证见 THIRD_PARTY_NOTICES.md。
//
//go:embed libs/libs.js
var libsSource string

var (
	libsOnce    sync.Once
	libsProgram *goja.Program
	libsErr     error
)

// bundledLibs 是 libs.js 里提供的模块名
var bundledLibs = []string{"crypto-js", "qs", "dayjs", "he", "big-integer", "cheerio"}

// loadLibs 在虚拟机里执行 libs.js。编译结果（Program）可以在多个虚拟机之间共享，只编译一次。
func loadLibs(vm *goja.Runtime) error {
	libsOnce.Do(func() {
		libsProgram, libsErr = goja.Compile("libs.js", libsSource, false)
	})
	if libsErr != nil {
		return libsErr
	}
	_, err := vm.RunProgram(libsProgram)
	return err
}
