package infrastructure

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func InitTracer(
	ctx context.Context,
	serviceName, endpoint string,
	maxQueue, maxBatch int,
	traceRatio float64,
	batchTimeout, exportTimeout time.Duration,
) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}

	result, _ := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))

	bspOptions := []sdktrace.BatchSpanProcessorOption{
		sdktrace.WithMaxQueueSize(maxQueue),
		sdktrace.WithMaxExportBatchSize(maxBatch),
		sdktrace.WithBatchTimeout(batchTimeout),
		sdktrace.WithExportTimeout(exportTimeout),
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, bspOptions...),
		sdktrace.WithResource(result),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(traceRatio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
