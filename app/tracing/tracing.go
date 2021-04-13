package tracing

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/efficientgo/tools/core/pkg/merrors"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/propagation"
	etrace "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

// Option sets the value of an option for a Config.
type Option func(*options)

type options struct {
	newExporters []func() (SpanExporter, error)
	sampler      Sampler
	svcName      string
}

// WithStartedSpanExporter sets the exporter for spans.
// Use WithOTLP or WithPrettyPrinter for simpler options.
func WithStartedSpanExporter(e SpanExporter) Option {
	return func(o *options) {
		o.newExporters = append(o.newExporters, func() (SpanExporter, error) { return e, nil })
	}
}

// WithOTLP sets the gRPC OTLP exporter for spans.
func WithOTLP(opts ...OTLPOption) Option {
	return func(o *options) {
		o.newExporters = append(o.newExporters, func() (SpanExporter, error) {
			e, err := otlp.NewExporter(context.TODO(), otlpgrpc.NewDriver(opts...))
			if err != nil {
				return nil, errors.Wrap(err, "OTLP exporter creation")
			}
			return e, nil
		})
	}
}

// WithPrinter sets the printing exporter for spans.
func WithPrinter(w io.Writer) Option {
	return func(o *options) {
		o.newExporters = append(o.newExporters, func() (etrace.SpanExporter, error) {
			e, err := stdout.NewExporter(
				stdout.WithWriter(w),
			)
			if err != nil {
				return nil, errors.Wrap(err, "pretty print exporter creation")
			}
			return e, nil
		})
	}
}

// WithSampler sets sampler.
func WithSampler(s Sampler) Option {
	return func(o *options) {
		o.sampler = s
	}
}

// WithSvcName sets service name. Usually it comes in format of service:app.
func WithSvcName(s string) Option {
	return func(o *options) {
		o.svcName = s
	}
}

type Provider struct {
	trace.TracerProvider
	propagation.TextMapPropagator
}

func NewProvider(opts ...Option) (*Provider, func() error, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	var closers []func() error
	closeFn := func() error {
		errs := merrors.New()
		for _, cl := range closers {
			errs.Add(cl())
		}
		return errs.Err()
	}

	if len(o.newExporters) == 0 {
		return nil, closeFn, errors.New("no exporters were configured")
	}

	svcName := o.svcName
	if svcName == "" {
		executable, err := os.Executable()
		if err != nil {
			svcName = "unknown_service:go"
		} else {
			svcName = "unknown_service:" + filepath.Base(executable)
		}
	}

	tpOpts := []sdktrace.TracerProviderOption{
		// TODO(bwplotka): Detect process info etc.
		sdktrace.WithResource(resource.NewWithAttributes(attribute.KeyValue{Key: semconv.ServiceNameKey, Value: attribute.StringValue(svcName)})),
	}
	for _, ne := range o.newExporters {
		exporter, err := ne()
		if err != nil {
			errcapture.Do(&err, closeFn, "close")
			return nil, func() error { return nil }, err
		}
		closers = append(closers, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			return exporter.Shutdown(ctx)
		})

		// TODO(bwplotka): Allow different batch options too.
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exporter))
	}

	if o.sampler != nil {
		tpOpts = append(tpOpts, sdktrace.WithSampler(o.sampler))
	} else {
		tpOpts = append(tpOpts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
	}

	p := &Provider{
		TracerProvider: sdktrace.NewTracerProvider(tpOpts...),
		// TODO(bwplotka): Allow different propagations.
		TextMapPropagator: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	}

	// Globals kills people.
	otel.SetTracerProvider(p)
	otel.SetTextMapPropagator(p)

	return p, closeFn, nil
}
