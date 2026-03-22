package webserver

import (
	"context"
	"errors"
	"internal/shared/metrics"
	"internal/shared/settings"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

type WebServer struct {
	config settings.HTTPConfig
	router *httprouter.Router
	server *http.Server
}

func NewWebServer(config settings.HTTPConfig) *WebServer {
	router := httprouter.New()
	webServer := &WebServer{
		config: config,
		router: router,
	}

	webServer.server = webServer.buildHTTPServer()

	return webServer
}

func (w *WebServer) buildHTTPServer() *http.Server {
	return &http.Server{
		Addr:         w.config.Addr,
		Handler:      CorrelationID(w.router),
		ReadTimeout:  w.config.ReadTimeoutDuration(),
		WriteTimeout: w.config.WriteTimeoutDuration(),
		IdleTimeout:  w.config.IdleTimeoutDuration(),
	}
}

func (w *WebServer) RegisterRoutes(routes []Route) {
	for _, route := range routes {
		if route.Method == "" || route.Path == "" || route.Handler == nil {
			continue
		}

		handler := route.Handler
		// Instrument all handlers except /metrics to avoid self-observation noise.
		if route.Path != "/metrics" {
			handler = metrics.InstrumentHTTPHandler(route.Method, route.Path, handler)
		}

		w.router.HandlerFunc(route.Method, route.Path, handler)
	}
}

func (w *WebServer) Start() error {
	err := w.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (w *WebServer) Shutdown(ctx context.Context) error {
	return w.server.Shutdown(ctx)
}
