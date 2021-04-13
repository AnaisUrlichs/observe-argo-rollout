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

var emptyTraceCtx = trace.SpanContext{}

// InstrumentationTripperware holds necessary metrics to instrument an http.RoundTripper
// and provides necessary behaviors.
type InstrumentationTripperware interface {
	WrapRoundTripper(targetName string, next http.RoundTripper) http.RoundTripper
}

type nopInstrumentationTripperware struct{}

func (ins nopInstrumentationTripperware) WrapRoundTripper(_ string, next http.RoundTripper) http.RoundTripper {
	return next
}

// NewNopInstrumentationTripperware provides a InstrumentationTripperware which does nothing.
func NewNopInstrumentationTripperware() InstrumentationTripperware {
	return nopInstrumentationTripperware{}
}

type instrumentationTripperware struct {
	reg     prometheus.Registerer
	tp      *tracing.Provider
	buckets []float64
}

// NewInstrumentationTripperware provides default InstrumentationTripperware.
// Passing nil as buckets uses the default buckets.
// TODO(bwplotka): Add optional args.
func NewInstrumentationTripperware(reg prometheus.Registerer, buckets []float64, tp *tracing.Provider) InstrumentationTripperware {
	if buckets == nil {
		buckets = []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120, 240, 360, 720}
	}

	return &instrumentationTripperware{reg: reg, buckets: buckets, tp: tp}
}

func (ins *instrumentationTripperware) WrapRoundTripper(targetName string, next http.RoundTripper) http.RoundTripper {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"target": targetName}, ins.reg)
	requestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_client_request_duration_seconds",
			Help:    "Tracks the latencies for HTTP requests.",
			Buckets: ins.buckets,
		},
		[]string{"method", "code"},
	)

	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, []string{"method", "code"},
	)

	requestsInFlight := promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "http_client_requests_inflight",
			Help: "Tracks the number of HTTP requests currently in flight.",
		},
	)

	base := promhttp.InstrumentRoundTripperInFlight(
		requestsInFlight,
		// TODO(bwplotka): Can't use promhttp.InstrumentRoundTripperCounter or promhttp.InstrumentRoundTripperDuration, propose exemplars feature.
		promhttp.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			now := time.Now()
			resp, err := next.RoundTrip(req)
			if err != nil {
				return resp, err
			}

			cntr := requestsTotal.WithLabelValues(strings.ToLower(req.Method), fmt.Sprintf("%d", resp.StatusCode))
			observer := requestDuration.WithLabelValues(strings.ToLower(req.Method), fmt.Sprintf("%d", resp.StatusCode))
			// If we find a TraceID from OpenTelemetry we'll expose it as Exemplar.

			if spanCtx := trace.SpanContextFromContext(req.Context()); spanCtx.HasTraceID() && spanCtx.IsSampled() {
				traceID := prometheus.Labels{"traceID": spanCtx.TraceID().String()}

				cntr.(prometheus.ExemplarAdder).AddWithExemplar(1, traceID)
				observer.(prometheus.ExemplarObserver).ObserveWithExemplar(time.Since(now).Seconds(), traceID)
				return resp, err
			}

			cntr.Inc()
			observer.Observe(time.Since(now).Seconds())
			return resp, err
		}),
	)
	if ins.tp != nil {
		return otelhttp.NewTransport(base, otelhttp.WithTracerProvider(ins.tp), otelhttp.WithPropagators(ins.tp))
	}
	return base
}
