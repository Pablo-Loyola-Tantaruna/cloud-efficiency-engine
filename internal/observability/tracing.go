package observability

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.37.0"
)

func InitTracing(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	if !tracingEnabled() {
		return func(context.Context) error { return nil }, nil
	}

	if configuredName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); configuredName != "" {
		serviceName = configuredName
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTEL resource: %w", err)
	}

	tpr := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.ParentBased(
				sdktrace.TraceIDRatioBased(traceSampleRatio()),
			),
		),
	)

	otel.SetTracerProvider(tpr)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tpr.Shutdown, nil
}

func tracingEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_ENABLED")))
	if value == "true" || value == "1" || value == "yes" {
		return true
	}
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
}

func traceSampleRatio() float64 {
	value := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
	if value == "" {
		return 0.10
	}
	ratio, err := strconv.ParseFloat(value, 64)
	if err != nil || ratio <= 0 || ratio > 1 {
		return 0.10
	}
	return ratio
}
