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
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var Tracer trace.Tracer

const serverName = "ExampleService"

func initTracer() *sdktrace.TracerProvider {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serverName),
		),
	)

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	Tracer = tp.Tracer(serverName)
	return tp
}

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Errorf("Error shutting down tracer provider: %v", err)
		}
	}()

	r, err := router.New(serverName)
	if err != nil {
		panic(err)
	}

	hello := &HelloWorldService{Name: "Hatsune Miku"}
	r.Attach(hello)

	log.WithField("port", "8080").Info("Starting server")
	r.Start(":8080")
}

type HelloWorldService struct {
	Name string
}

func (h *HelloWorldService) Route(e *echo.Echo) error {
	e.GET("/hello", h.Hello())
	return nil
}

func (h *HelloWorldService) Hello() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, span := Tracer.Start(c.Request().Context(), "hello")
		defer span.End()

		log.LoggerWithSpan(ctx).WithField("name", h.Name).Info("Hello")
		return c.String(200, fmt.Sprintf("Hello %s Service", h.Name))
	}
}
