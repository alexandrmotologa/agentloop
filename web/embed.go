package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFS returns the filesystem pointing to the built web dist directory.
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
