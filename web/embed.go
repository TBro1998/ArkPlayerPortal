// Package web 打包进程序的网页（Vite 的构建结果 web/dist）。没有构建时 dist 里只有 .gitkeep，
// 玩家站照常运行，网页显示「还没有构建」
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS 网页的静态文件，根目录是 dist
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
