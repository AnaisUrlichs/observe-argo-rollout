package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func DoInSpan(ctx context.Context, spanName string, f func(context.Context, Span), opts ...SpanOption) {
	sctx, span := Start(ctx, spanName, opts...)
	f(sctx, span)
	span.End()
}

func Start(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, Span) {
	return trace.SpanFromContext(ctx).Tracer().Start(ctx, spanName, opts...)
}
