package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type Options struct {
	AdminToken string
}

func Build() *server.Hertz {
	return BuildWithService(NewMemoryService(map[string]int{
		"critical": 30,
		"high":     60,
		"medium":   180,
		"low":      300,
		"info":     600,
	}))
}

func BuildWithAddr(addr string) *server.Hertz {
	return build(addr, NewMemoryService(map[string]int{
		"critical": 30,
		"high":     60,
		"medium":   180,
		"low":      300,
		"info":     600,
	}), Options{})
}

func BuildWithAddrAndService(addr string, service Service) *server.Hertz {
	return build(addr, service, Options{})
}

func BuildWithService(service Service) *server.Hertz {
	return build("", service, Options{})
}

func BuildWithOptions(addr string, service Service, options Options) *server.Hertz {
	return build(addr, service, options)
}

func build(addr string, service Service, options Options) *server.Hertz {
	var h *server.Hertz
	if addr == "" {
		h = server.Default()
	} else {
		h = server.Default(server.WithHostPorts(addr))
	}

	h.GET("/", frontendAsset("index.html", "text/html; charset=utf-8"))
	h.GET("/styles.css", frontendAsset("styles.css", "text/css; charset=utf-8"))
	h.GET("/app.js", frontendAsset("app.js", "application/javascript; charset=utf-8"))

	h.GET("/healthz", healthz())
	h.POST("/api/client/events", handleEvents(service))

	admin := h.Group("/", authMiddleware(options.AdminToken))
	admin.POST("/api/alerts/query", queryAlerts(service))

	admin.GET("/api/whitelist", listWhitelist(service))
	admin.POST("/api/whitelist", createWhitelist(service))
	admin.PUT("/api/whitelist/:id", updateWhitelist(service))
	admin.DELETE("/api/whitelist/:id", deleteWhitelist(service))

	admin.GET("/api/false-positives", listFalsePositiveRecords(service))
	admin.POST("/api/false-positives", createFalsePositiveRecord(service))
	admin.DELETE("/api/false-positives/:id", deleteFalsePositiveRecord(service))

	return h
}

func authMiddleware(token string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if token == "" {
			c.Next(ctx)
			return
		}
		if string(c.GetHeader("Authorization")) != "Bearer "+token {
			c.AbortWithStatusJSON(consts.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		c.Next(ctx)
	}
}
