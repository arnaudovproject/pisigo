package otelx

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/arnaudovproject/pisigo"
)

func Setup(serviceName string) (func(context.Context) error, error) {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp.Shutdown, nil
}

func Middleware(serviceName string) pisigo.Middleware {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			ctx := propagator.Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))
			ctx, span := tracer.Start(ctx, c.Method()+" "+c.Path(),
				trace.WithAttributes(
					attribute.String("http.method", c.Method()),
					attribute.String("http.route", c.Path()),
					attribute.String("http.client_ip", c.IP()),
				),
			)
			defer span.End()
			c.SetRequest(c.Request().WithContext(ctx))
			err := next(c)
			span.SetAttributes(attribute.Int("http.status_code", c.StatusCode()))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
			if c.StatusCode() >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("status %d", c.StatusCode()))
			}
			return nil
		}
	}
}
