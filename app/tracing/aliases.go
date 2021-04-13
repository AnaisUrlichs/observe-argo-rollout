package tracing

import (
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	etrace "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Span = trace.Span
type SpanOption = trace.SpanOption

type OTLPOption = otlpgrpc.Option

// WithOTLPEndpoint allows one to set the endpoint that the exporter will
// connect to the collector on. If unset, it will instead try to use
// connect to DefaultCollectorHost:DefaultCollectorPort.
func WithOTLPEndpoint(endpoint string) OTLPOption {
	return otlpgrpc.WithEndpoint(endpoint)
}

// WithOTLPInsecure disables client transport security for the exporter's gRPC connection
// just like grpc.WithInsecure() https://pkg.go.dev/google.golang.org/grpc#WithInsecure
// does. Note, by default, client security is required unless WithInsecure is used.
func WithOTLPInsecure() OTLPOption {
	return otlpgrpc.WithInsecure()
}

// WithOTLPDialOption opens support to any grpc.DialOption to be used. If it conflicts
// with some other configuration the GRPC specified via the collector the ones here will
// take preference since they are set last.
func WithOTLPDialOption(opts ...grpc.DialOption) OTLPOption {
	return otlpgrpc.WithDialOption(opts...)
}

// WithOTLPHeaders will send the provided headers with gRPC requests
func WithOTLPHeaders(headers map[string]string) OTLPOption {
	return otlpgrpc.WithHeaders(headers)
}

// WithOTLPTLSCredentials allows the connection to use TLS credentials
// when talking to the server. It takes in grpc.TransportCredentials instead
// of say a Certificate file or a tls.Certificate, because the retrieving
// these credentials can be done in many ways e.g. plain file, in code tls.Config
// or by certificate rotation, so it is up to the caller to decide what to use.
func WithOTLPTLSCredentials(creds credentials.TransportCredentials) OTLPOption {
	return otlpgrpc.WithTLSCredentials(creds)
}

type Sampler = sdktrace.Sampler

// TraceIDRatioBasedSampler samples a given fraction of traces. Fractions >= 1 will
// always sample. Fractions < 0 are treated as zero. To respect the
// parent trace's `SampledFlag`, the `TraceIDRatioBased` sampler should be used
// as a delegate of a `Parent` sampler.
//nolint:golint // golint complains about stutter of `trace.TraceIDRatioBased`
func TraceIDRatioBasedSampler(fraction float64) Sampler {
	return sdktrace.TraceIDRatioBased(fraction)
}

type SpanExporter = etrace.SpanExporter
