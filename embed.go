package claudehaus

import (
	"embed"
	"io/fs"
)

//go:embed all:web
var webFS embed.FS

// TemplatesFS returns the embedded web/templates filesystem rooted at templates/.
func TemplatesFS() fs.FS {
	sub, err := fs.Sub(webFS, "web/templates")
	if err != nil {
		panic(err)
	}
	return sub
}

// StaticFS returns the embedded web/static filesystem rooted at static/.
func StaticFS() fs.FS {
	sub, err := fs.Sub(webFS, "web/static")
	if err != nil {
		panic(err)
	}
	return sub
}
