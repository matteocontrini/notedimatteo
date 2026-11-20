// Package web embeds the HTML templates and the assets built by Vite.
//
// The dist directory must exist before compiling: run `npm run go:assets` first.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var templates embed.FS

//go:embed all:dist
var dist embed.FS

func Templates() fs.FS {
	return mustSub(templates, "templates")
}

func Dist() fs.FS {
	return mustSub(dist, "dist")
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
