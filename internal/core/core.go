// Package core 是桌面版和 Android 版共用的播放器内核：歌曲库、播放队列、歌单（SQLite）、
// 歌词、插件和配置都在这里。
//
// 桌面版（根目录的 main 包）把这里的方法包一层绑定给 Wails；Android 版通过 mobile/gocore
// 用 gomobile 编译成 .aar，由 Capacitor 插件调用。和平台有关的事情（弹出选择文件的窗口、
// 记住窗口大小、扫描手机里的音乐）不放在这里，由各自的平台代码负责。
package core

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"musicplayer/internal/music"
	"musicplayer/internal/plugin"
	"musicplayer/internal/queue"
	"musicplayer/internal/store"
	"musicplayer/internal/stream"
)

// errNoDataDir 在没有保存数据的文件夹时返回（比如桌面版找不到 %AppData%）
var errNoDataDir = errors.New("找不到保存数据的文件夹")

// errNoPlugins 在插件管理器没能创建时返回
var errNoPlugins = errors.New("插件功能不可用：找不到保存插件的文件夹")

// Options 是创建 Core 的参数
type Options struct {
	// DataDir 保存配置、数据库和插件的文件夹，不存在时会自动创建。
	// 桌面版是 %AppData%\SwiftPaw，Android 版是应用自己的私有文件夹。为空表示找不到文件夹，
	// 这时播放队列还能用，歌单和插件会返回错误
	DataDir string
	// AppVersion 是程序的版本号，设置页显示，也会通过 env.appVersion 告诉插件
	AppVersion string
	// Logf 用来输出插件日志等信息，为 nil 时用 log.Printf
	Logf func(format string, args ...any)
}

// Core 是播放器内核。所有公开方法都可以在多个 goroutine 里同时调用
// （Android 上界面和后台播放服务会同时用它）。
type Core struct {
	dataDir string
	version string

	mu      sync.Mutex
	library []music.Song // 最近一次扫描到的歌曲库

	queue *queue.Queue // 播放队列
	store *store.Store // 歌单、收藏、最近播放（SQLite）
	dbErr error        // 数据库打不开时记下原因，调用歌单相关方法时返回

	bgMu sync.Mutex // 背景图片的复制和删除一次只做一个

	plugins *plugin.Manager // 插件管理器；没有数据文件夹时为 nil
	stream  *stream.Proxy   // 桌面版在线歌曲的播放代理，前端用 /stream?id=... 播放
}

// New 创建内核：建好数据文件夹，打开数据库，并在后台加载插件（不拖慢启动）
func New(opts Options) *Core {
	if opts.Logf == nil {
		opts.Logf = log.Printf
	}
	c := &Core{dataDir: opts.DataDir, version: opts.AppVersion, queue: queue.New(), stream: stream.New(nil)}

	err := errNoDataDir
	if c.dataDir != "" {
		err = os.MkdirAll(c.dataDir, 0o755)
	}
	if err == nil {
		c.startPlugins(opts.Logf)
		c.store, err = store.Open(filepath.Join(c.dataDir, "swiftpaw.db"))
	}
	c.dbErr = err
	return c
}

// Close 关闭数据库和插件运行环境，程序退出时调用
func (c *Core) Close() {
	if c.store != nil {
		c.store.Close()
	}
	if c.plugins != nil {
		c.plugins.Close()
	}
}

// Stream 返回在线歌曲的播放代理（桌面版挂在 /stream 上）
func (c *Core) Stream() http.Handler {
	return c.stream
}

// AppVersion 返回版本号
func (c *Core) AppVersion() string {
	return c.version
}

// startPlugins 创建插件管理器，并在后台加载插件
func (c *Core) startPlugins(logf func(string, ...any)) {
	c.plugins = plugin.NewManager(plugin.Options{
		Dir:        filepath.Join(c.dataDir, "plugins"),
		StatePath:  filepath.Join(c.dataDir, "plugins.json"),
		AppVersion: c.version,
		Logger: func(id, level, message string) {
			logf("[插件 %s] %s: %s", id, level, message)
		},
	})
	// LoadAll 加载时持有管理器的锁，这时调用其它插件方法会等它加载完
	go func() {
		if err := c.plugins.LoadAll(); err != nil {
			logf("加载插件失败：%v", err)
		}
	}()
}

// db 返回数据库；打不开时返回错误，前端会显示出来
func (c *Core) db() (*store.Store, error) {
	if c.store == nil {
		return nil, fmt.Errorf("数据库打不开：%v", c.dbErr)
	}
	return c.store, nil
}
