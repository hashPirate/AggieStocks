package metrics

import (
	"context"
	"log"
	"net/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

type Registry struct {
	meter              metric.Meter
	EventsProcessed    metric.Int64Counter
	EventsDuplicate    metric.Int64Counter
	HandlerErrors      metric.Int64Counter
	HandlerRetries     metric.Int64Counter
	HandlerLatencyMs   metric.Float64Histogram
}

func NewRegistry() *Registry {
	exporter,err:=prometheus.New()
	if err!=nil {
		log.Fatal(err)
	}
	provider:=metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	http.Handle("/metrics",exporter)
	go func() {
		_=http.ListenAndServe(":9464",nil)
	}()

	meter:=otel.Meter("eon/worker")

	eventsProcessed,_:=meter.Int64Counter("eon_events_processed_total")
	eventsDuplicate,_:=meter.Int64Counter("eon_events_duplicate_total")
	handlerErrors,_:=meter.Int64Counter("eon_handler_errors_total")
	handlerRetries,_:=meter.Int64Counter("eon_handler_retries_total")
	handlerLatency,_:=meter.Float64Histogram("eon_handler_latency_ms")

	return &Registry{
		meter:            meter,
		EventsProcessed:  eventsProcessed,
		EventsDuplicate:  eventsDuplicate,
		HandlerErrors:    handlerErrors,
		HandlerRetries:   handlerRetries,
		HandlerLatencyMs: handlerLatency,
	}
}

func (r *Registry) RecordProcessed(ctx context.Context,consumer,partition string) {
	r.EventsProcessed.Add(ctx,1,metric.WithAttributes(
		attribute.String("consumer",consumer),
		attribute.String("partition",partition),
	))
}

func (r *Registry) RecordDuplicate(ctx context.Context,consumer,partition string) {
	r.EventsDuplicate.Add(ctx,1,metric.WithAttributes(
		attribute.String("consumer",consumer),
		attribute.String("partition",partition),
	))
}

func (r *Registry) RecordHandlerError(ctx context.Context,consumer,partition string) {
	r.HandlerErrors.Add(ctx,1,metric.WithAttributes(
		attribute.String("consumer",consumer),
		attribute.String("partition",partition),
	))
}

func (r *Registry) RecordHandlerRetry(ctx context.Context,consumer,partition string) {
	r.HandlerRetries.Add(ctx,1,metric.WithAttributes(
		attribute.String("consumer",consumer),
		attribute.String("partition",partition),
	))
}

func (r *Registry) RecordHandlerLatency(ctx context.Context,consumer,partition string,ms float64) {
	r.HandlerLatencyMs.Record(ctx,ms,metric.WithAttributes(
		attribute.String("consumer",consumer),
		attribute.String("partition",partition),
	))
}

