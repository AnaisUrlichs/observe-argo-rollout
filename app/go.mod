module github.com/AnaisUrlichs/observe-argo-rollout/app

go 1.15

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.0 // indirect
	github.com/efficientgo/tools/core v0.0.0-20210326193628-425a09c04e05
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.18.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.19.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/exporters/otlp v0.19.0
	go.opentelemetry.io/otel/exporters/stdout v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
	google.golang.org/grpc v1.36.0
)
