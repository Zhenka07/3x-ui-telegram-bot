package observability

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// InitTracer initializes the OpenTelemetry TracerProvider according to configuration.
// If OTelEnabled is false, it returns a no-op tracer and a no-op shutdown function.
func InitTracer(ctx context.Context, cfg Config) (trace.Tracer, func(context.Context) error, error) {
	if !cfg.OTelEnabled {
		noopProvider := noop.NewTracerProvider()
		otel.SetTracerProvider(noopProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return noopProvider.Tracer("3x-ui-admin-bot"), func(context.Context) error { return nil }, nil
	}

	serviceName := cfg.ServiceName
	if strings.TrimSpace(serviceName) == "" {
		serviceName = "3x-ui-admin-bot"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("создание OTel resource: %w", err)
	}

	var spanExporter sdktrace.SpanExporter
	switch strings.ToLower(strings.TrimSpace(cfg.ExporterType)) {
	case "otlp":
		opts := []otlptracegrpc.Option{}
		if cfg.OTLPEndpoint != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint))
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("создание OTLP gRPC экспортера: %w", err)
		}
		spanExporter = exp
	default: // "stdout"
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, nil, fmt.Errorf("создание stdout экспортера: %w", err)
		}
		spanExporter = exp
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer(serviceName)
	return tracer, tp.Shutdown, nil
}

// Tracer returns a named tracer from the global OpenTelemetry provider.
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}
