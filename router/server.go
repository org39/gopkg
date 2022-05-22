package router

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// Routable is an interface for objects that can be routable.
type Routable interface {
	Route(e *echo.Echo) error
}

// Router is a object that can be dispatch a Router.
type Router struct {
	*echo.Echo
}

// Attach attach a Routable to the Router.
func (r *Router) Attach(d Routable) *Router {
	if err := d.Route(r.Echo); err != nil {
		panic(err)
	}
	return r
}

// New creates a new Router.
func New(serverName string) (*Router, error) {
	e := echo.New()

	// tracer middleware is first
	e.Use(otelecho.Middleware(
		serverName,
		otelecho.WithSkipper(
			func(c echo.Context) bool {
				// skip healthcheck endpoint
				return strings.HasPrefix(c.Path(), "/_health")
			},
		),
	))

	// any usefule middleware can be added here
	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.DefaultGzipConfig))

	e.HideBanner = true
	e.HidePort = true
	return wrap(e), nil
}

// Start starts the Router.
func (r *Router) Start(address string) error {
	// logger middleware is always last
	r.Echo.Use(loggerMiddleware())

	return r.Echo.Start(address)
}

func wrap(e *echo.Echo) *Router {
	return &Router{e}
}
