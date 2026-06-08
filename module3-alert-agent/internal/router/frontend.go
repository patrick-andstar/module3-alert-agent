package router

import (
	"context"
	"embed"
	"io/fs"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

//go:embed web/*
var frontendFiles embed.FS

// embeddedFS is a pre-extracted sub-filesystem rooted at web/
var embeddedFS fs.FS

func init() {
	var err error
	embeddedFS, err = fs.Sub(frontendFiles, "web")
	if err != nil {
		panic("failed to extract web/ sub filesystem: " + err.Error())
	}
}

func serveFrontend(_ context.Context, c *app.RequestContext) {
	path := strings.TrimPrefix(string(c.Path()), "/")
	if path == "" {
		path = "index.html"
	}

	// Try exact file match
	data, err := fs.ReadFile(embeddedFS, path)
	if err == nil {
		contentType := contentTypeByPath(path)
		c.Data(consts.StatusOK, contentType, data)
		return
	}

	// Fallback to index.html for SPA routing
	data, err = fs.ReadFile(embeddedFS, "index.html")
	if err != nil {
		c.String(consts.StatusNotFound, "not found")
		return
	}
	c.Data(consts.StatusOK, "text/html; charset=utf-8", data)
}

func contentTypeByPath(path string) string {
	ext := ""
	if parts := strings.Split(path, "."); len(parts) > 0 {
		ext = "." + parts[len(parts)-1]
	}
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	default:
		return "application/octet-stream"
	}
}
