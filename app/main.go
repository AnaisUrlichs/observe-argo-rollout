package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AnaisUrlichs/observe-argo-rollout/app/exthttp"
	"github.com/AnaisUrlichs/observe-argo-rollout/app/tracing"
	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"go.opentelemetry.io/otel/attribute"
)

var (
	latDecider *latencyDecider

	// TODO(bwplotka): Move those flags out of globals.
	addr               = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	appVersion         = flag.String("set-version", "first", "Injected version to be presented via metrics.")
	lat                = flag.String("latency", "90%500ms,10%200ms", "Encoded latency and probability of the response in format as: <probability>%<duration>,<probability>%<duration>....")
	successProb        = flag.Float64("success-prob", 100, "The probability (in %) of getting a successful response")
	traceEndpoint      = flag.String("trace-endpoint", "tempo.demo.svc.cluster.local:9091", "The gRPC OTLP endpoint for tracing backend. Hack: Set it to 'stdout' to print traces to the output instead")
	traceSamplingRatio = flag.Float64("trace-sampling-ratio", 1.0, "Sampling ratio")
)

type latencyDecider struct {
	latencies     []time.Duration
	probabilities []float64 // Sorted ascending.
}

func newLatencyDecider(encodedLatencies string) (*latencyDecider, error) {
	l := latencyDecider{}

	s := strings.Split(encodedLatencies, ",")
	// Be smart, sort while those are encoded, so they are sorted by probability number.
	sort.Strings(s)

	cumulativeProb := 0.0
	for _, e := range s {
		entry := strings.Split(e, "%")
		if len(entry) != 2 {
			return nil, errors.Errorf("invalid input %v", encodedLatencies)
		}
		f, err := strconv.ParseFloat(entry[0], 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse probabilty %v as float", entry[0])
		}
		cumulativeProb += f
		l.probabilities = append(l.probabilities, f)

		d, err := time.ParseDuration(entry[1])
		if err != nil {
			return nil, errors.Wrapf(err, "parse latency %v as duration", entry[1])
		}
		l.latencies = append(l.latencies, d)
	}
	if cumulativeProb != 100 {
		return nil, errors.Errorf("overall probability has to equal 100. Parsed input equals to %v", cumulativeProb)
	}
	fmt.Println("Latency decider created:", l)
	return &l, nil
}

func (l latencyDecider) AddLatency(ctx context.Context) {
	_, span := tracing.Start(ctx, "addingLatencyBasedOnProbability")
	defer span.End()

	n := rand.Float64() * 100
	span.SetAttributes(attribute.Array("latencyProbabilities", l.probabilities))
	span.SetAttributes(attribute.Float64("lucky%", n))

	for i, p := range l.probabilities {
		if n <= p {
			span.SetAttributes(attribute.String("latencyIntroduced", l.latencies[i].String()))
			<-time.After(l.latencies[i])
			return
		}
	}
}

func handlerPing(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.Start(r.Context(), "pingHandler")
	defer span.End()

	latDecider.AddLatency(ctx)

	tracing.DoInSpan(ctx, "writeStatusBasedOnSuccessProbability", func(ctx context.Context, span tracing.Span) {
		n := rand.Float64() * 100
		span.SetAttributes(attribute.Float64("successProbability", *successProb))
		span.SetAttributes(attribute.Float64("lucky%", n))

		if n <= *successProb {
			w.WriteHeader(200)
			_, _ = fmt.Fprintln(w, "pong")
		} else {
			w.WriteHeader(500)
		}

		if span.SpanContext().HasTraceID() && span.SpanContext().IsSampled() {
			_, _ = fmt.Fprintf(w, "BTW, here is (sampled) trace ID: %v", span.SpanContext().TraceID().String())
		}
	})
}

func main() {
	flag.Parse()
	if err := runMain(); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		log.Fatalf("Error: %+v", errors.Wrapf(err, "%s", flag.Arg(0)))
	}
}

func runMain() (err error) {
	latDecider, err = newLatencyDecider(*lat)
	if err != nil {
		return err
	}

	version.Version = *appVersion
	version.BuildUser = "Anaïs"

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		version.NewCollector("app"),
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	var tracingProvider *tracing.Provider
	if *traceEndpoint != "" {
		tOpts := []tracing.Option{
			tracing.WithSampler(tracing.TraceIDRatioBasedSampler(*traceSamplingRatio)),
			tracing.WithSvcName("demo:app"),
		}
		switch *traceEndpoint {
		case "stdout":
			tOpts = append(tOpts, tracing.WithPrinter(os.Stdout))
		default:
			tOpts = append(tOpts, tracing.WithOTLP(
				tracing.WithOTLPInsecure(),
				tracing.WithOTLPEndpoint(*traceEndpoint),
			))
		}
		tp, closeFn, err := tracing.NewProvider(tOpts...)
		if err != nil {
			return err
		}
		tracingProvider = tp
		defer errcapture.Do(&err, closeFn, "close tracers")
		fmt.Println("Tracing enabled", *traceEndpoint)
	}

	m := http.NewServeMux()
	m.Handle("/metrics", exthttp.NewInstrumentationMiddleware(reg, nil, nil).
		WrapHandler("/metrics", promhttp.HandlerFor(
			reg,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics to support exemplars.
				EnableOpenMetrics: true,
			},
		)))
	m.HandleFunc("/ping", exthttp.NewInstrumentationMiddleware(reg, nil, tracingProvider).
		WrapHandler("/ping", http.HandlerFunc(handlerPing)))
	srv := http.Server{Addr: *addr, Handler: m}

	// Setup multiple 2 jobs. One is for serving HTTP requests, second to listen for Linux signals like Ctrl+C.
	g := &run.Group{}
	g.Add(func() error {
		fmt.Println("HTTP Server listening on", *addr)
		if err := srv.ListenAndServe(); err != nil {
			return errors.Wrap(err, "starting web server")
		}
		return nil
	}, func(error) {
		if err := srv.Close(); err != nil {
			fmt.Println("Failed to stop web server:", err)
		}
	})
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	return g.Run()
}
