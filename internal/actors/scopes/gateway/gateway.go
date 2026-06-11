package actorgateway

import (
	"context"
	"internal/shared/settings"
	"internal/shared/webserver"
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type Gateway struct {
	cfg       settings.HTTPConfig
	routes    []webserver.Route
	webServer *webserver.WebServer
}

func NewGateway(config settings.HTTPConfig, routes []webserver.Route) actor.Producer {
	return func() actor.Receiver {
		return &Gateway{
			cfg:    config,
			routes: append([]webserver.Route(nil), routes...),
		}
	}
}

func (g *Gateway) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		_ = msg
		g.start(c)
	case actor.Stopped:
		if g.webServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := g.webServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("failed to stop gateway", "error", err)
			}
		}
	}
}

func (g *Gateway) start(ctx *actor.Context) {
	g.webServer = webserver.NewWebServer(g.cfg)
	g.webServer.RegisterRoutes(g.routes)

	go func() {
		err := g.webServer.Start()
		if err != nil {
			slog.Error("failed to start gateway", "error", err)
			ctx.Engine().Poison(ctx.PID())
		}
		slog.Info("gateway stopped")
	}()
}
