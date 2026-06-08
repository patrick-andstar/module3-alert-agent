package router

import (
	"context"
	"embed"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

//go:embed web/*
var frontendFiles embed.FS

func frontendAsset(name, contentType string) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		data, err := frontendFiles.ReadFile("web/" + name)
		if err != nil {
			c.String(consts.StatusNotFound, "not found")
			return
		}
		c.Data(consts.StatusOK, contentType, data)
	}
}
