package router

import (
	"strings"

	"github.com/HatsuneMiku3939/ocecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
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
func New() (*Router, error) {
	e := echo.New()

	e.Use(loggerMiddleware())
	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.DefaultGzipConfig))
	e.Use(ocecho.OpenCensusMiddleware(
		ocecho.OpenCensusConfig{
			Skipper: func(c echo.Context) bool {
				// skip healthcheck endpoint
				return strings.HasPrefix(c.Path(), "/_health")
			},
			TraceOptions: ocecho.TraceOptions{
				IsPublicEndpoint: false,
				Propagation:      &b3.HTTPFormat{},
				StartOptions:     trace.StartOptions{},
			},
		},
	))

	e.HideBanner = true
	return wrap(e), nil
}

func wrap(e *echo.Echo) *Router {
	return &Router{e}
}
