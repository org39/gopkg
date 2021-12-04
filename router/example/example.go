package main

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/org39/gopkg/log"
	"github.com/org39/gopkg/router"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type HelloWorldService struct {
	Name string
}

func (h *HelloWorldService) Route(e *echo.Echo) error {
	e.GET("/hello", h.Hello())
	return nil
}

func (h *HelloWorldService) Hello() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(200, fmt.Sprintf("Hello %s Service", h.Name))
	}
}

var tracer = otel.Tracer("example-service")

func initTracer() *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Errorf("Error shutting down tracer provider: %v", err)
		}
	}()

	r, err := router.New("example-service")
	if err != nil {
		panic(err)
	}

	hello := &HelloWorldService{Name: "Hatsune Miku"}
	r.Attach(hello)

	r.Start(":8080")
}
