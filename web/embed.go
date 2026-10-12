// Package web embeds the built single-page app (web/dist) into the binary.
// Run `pnpm build` (or `make build`) first; without it the server explains
// how to build the frontend.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend rooted at dist/.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
