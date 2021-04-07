// Copyright (c) The Thanos Authors.
// Licensed under the Apache License 2.0.

package extpromhttp

import (
	"net/http"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/uber/jaeger-client-go"
)

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
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec
	requestsTotal    *prometheus.CounterVec
}

// NewInstrumentationTripperware provides default InstrumentationTripperware.
// Passing nil as buckets uses the default buckets.
func NewInstrumentationTripperware(reg prometheus.Registerer, buckets []float64) InstrumentationTripperware {
	if buckets == nil {
		buckets = []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120, 240, 360, 720}
	}

	ins := instrumentationTripperware{
		requestDuration: promauto.With(reg).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_client_request_duration_seconds",
				Help:    "Tracks the latencies for HTTP requests.",
				Buckets: buckets,
			},
			[]string{"code", "target", "method"},
		),

		requestsTotal: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_client_requests_total",
				Help: "Tracks the number of HTTP requests.",
			}, []string{"code", "target", "method"},
		),

		requestsInFlight: promauto.With(reg).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_client_requests_inflight",
				Help: "Tracks the number of HTTP requests currently in flight.",
			},
			[]string{"target"},
		),
	}
	return &ins
}

func (ins *instrumentationTripperware) WrapRoundTripper(targetName string, next http.RoundTripper) http.RoundTripper {
	return promhttp.InstrumentRoundTripperInFlight(
		ins.requestsInFlight.WithLabelValues("target"),
		promhttp.InstrumentRoundTripperCounter(
			ins.requestsTotal.MustCurryWith(prometheus.Labels{"target": targetName}),
			promhttp.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				now := time.Now()

				res, err := next.RoundTrip(req)
				if err != nil {
					// TODO(bwplotka) Should we bump histogram in this case too?
					return res, err
				}

				observer := ins.requestDuration.WithLabelValues(
					res.Status,
					targetName,
					strings.ToLower(req.Method),
				)
				observer.Observe(time.Since(now).Seconds())

				// If we find a tracingID we'll expose it as Exemplar.
				span := opentracing.SpanFromContext(req.Context())
				if span != nil {
					spanCtx, ok := span.Context().(jaeger.SpanContext)
					if ok {
						observer.(prometheus.ExemplarObserver).ObserveWithExemplar(
							time.Since(now).Seconds(),
							prometheus.Labels{
								"traceID": spanCtx.TraceID().String(),
							},
						)
					}
				}
				return res, err
			}),
		),
	)
}
