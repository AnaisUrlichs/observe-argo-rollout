// Copied from https://github.com/thanos-io/thanos/blob/30162377d15ef0b8b7c71081f22ceb7ab3ef0285/pkg/extprom/http/instrument_server.go

// Copyright (c) The Thanos Authors.
// Licensed under the Apache License 2.0.

package exthttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AnaisUrlichs/observe-argo-rollout/app/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentationMiddleware holds necessary metrics to instrument an http.Server
// and provides necessary behaviors.
type InstrumentationMiddleware interface {
	// WrapHandler wraps the given HTTP handler for instrumentation.
	WrapHandler(handlerName string, handler http.Handler) http.HandlerFunc
}

type nopInstrumentationMiddleware struct{}

func (ins nopInstrumentationMiddleware) WrapHandler(_ string, handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
}

// NewNopInstrumentationMiddleware provides a InstrumentationMiddleware which does nothing.
func NewNopInstrumentationMiddleware() InstrumentationMiddleware {
	return nopInstrumentationMiddleware{}
}

type instrumentationMiddleware struct {
	reg     prometheus.Registerer
	tp      *tracing.Provider
	buckets []float64
}

// NewInstrumentationMiddleware provides default InstrumentationMiddleware.
// Passing nil as buckets uses the default buckets.
func NewInstrumentationMiddleware(reg prometheus.Registerer, buckets []float64, tp *tracing.Provider) InstrumentationMiddleware {
	if buckets == nil {
		buckets = []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120, 240, 360, 720}
	}

	return &instrumentationMiddleware{reg: reg, buckets: buckets, tp: tp}
}

// WrapHandler wraps the given HTTP handler for instrumentation. It
// registers four metric collectors (if not already done) and reports HTTP
// metrics to the (newly or already) registered collectors: http_requests_total
// (CounterVec), http_request_duration_seconds (Histogram),
// http_request_size_bytes (Summary), http_response_size_bytes (Summary). Each
// has a constant label named "handler" with the provided handlerName as
// value. http_requests_total is a metric vector partitioned by HTTP method
// (label name "method") and HTTP status code (label name "code").
func (ins *instrumentationMiddleware) WrapHandler(handlerName string, handler http.Handler) http.HandlerFunc {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": handlerName}, ins.reg)

	requestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Tracks the latencies for HTTP requests.",
			Buckets: ins.buckets,
		},
		[]string{"method", "code"},
	)
	requestSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "Tracks the size of HTTP requests.",
		},
		[]string{"method", "code"},
	)
	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, []string{"method", "code"},
	)
	responseSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "Tracks the size of HTTP responses.",
		},
		[]string{"method", "code"},
	)
	// TODO(bwplotka): Add exemplars everywhere when supported: https://github.com/prometheus/client_golang/issues/854
	base := promhttp.InstrumentHandlerRequestSize(
		requestSize,
		promhttp.InstrumentHandlerCounter(
			requestsTotal,
			promhttp.InstrumentHandlerResponseSize(
				responseSize,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					now := time.Now()

					wd := &responseWriterDelegator{w: w}
					handler.ServeHTTP(wd, r)

					observer := requestDuration.WithLabelValues(strings.ToLower(r.Method), wd.Status())
					// If we find a TraceID from OpenTelemetry we'll expose it as Exemplar.
					if spanCtx := trace.SpanContextFromContext(r.Context()); spanCtx.HasTraceID() && spanCtx.IsSampled() {
						traceID := prometheus.Labels{"traceID": spanCtx.TraceID().String()}

						observer.(prometheus.ExemplarObserver).ObserveWithExemplar(time.Since(now).Seconds(), traceID)
						return
					}

					observer.Observe(time.Since(now).Seconds())
					return

				}),
			),
		),
	)

	if ins.tp != nil {
		return otelhttp.NewHandler(
			base,
			handlerName,
			otelhttp.WithTracerProvider(ins.tp),
			otelhttp.WithPropagators(ins.tp),
		).ServeHTTP
	}
	return base.ServeHTTP
}

// responseWriterDelegator implements http.ResponseWriter and extracts the statusCode.
type responseWriterDelegator struct {
	w          http.ResponseWriter
	written    bool
	statusCode int
}

func (wd *responseWriterDelegator) Header() http.Header {
	return wd.w.Header()
}

func (wd *responseWriterDelegator) Write(bytes []byte) (int, error) {
	return wd.w.Write(bytes)
}

func (wd *responseWriterDelegator) WriteHeader(statusCode int) {
	wd.written = true
	wd.statusCode = statusCode
	wd.w.WriteHeader(statusCode)
}

func (wd *responseWriterDelegator) StatusCode() int {
	if !wd.written {
		return http.StatusOK
	}
	return wd.statusCode
}

func (wd *responseWriterDelegator) Status() string {
	return fmt.Sprintf("%d", wd.StatusCode())
}
