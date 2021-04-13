package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/AnaisUrlichs/observe-argo-rollout/app/exthttp"
	"github.com/AnaisUrlichs/observe-argo-rollout/app/tracing"
	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// TODO(bwplotka): Move those flags out of globals.
	addr               = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	endpoint           = flag.String("endpoint", "http://app.demo.svc.cluster.local:8080/ping", "The address of pong app we can connect to and send requests.")
	pingsPerSec        = flag.Int("pings-per-second", 10, "How many pings per second we should request")
	traceEndpoint      = flag.String("trace-endpoint", "tempo.demo.svc.cluster.local:9091", "The gRPC OTLP endpoint for tracing backend. Hack: Set it to 'stdout' to print traces to the output instead")
	traceSamplingRatio = flag.Float64("trace-sampling-ratio", 1.0, "Sampling ratio")
)

func main() {
	flag.Parse()
	if err := runMain(); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		log.Fatalf("Error: %+v", errors.Wrapf(err, "%s", flag.Arg(0)))
	}
}

func runMain() (err error) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	var tracingProvider *tracing.Provider
	if *traceEndpoint != "" {
		tOpts := []tracing.Option{
			tracing.WithSampler(tracing.TraceIDRatioBasedSampler(*traceSamplingRatio)),
			tracing.WithSvcName("demo:pinger"),
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

	instr := exthttp.NewInstrumentationMiddleware(reg, nil, nil)
	m := http.NewServeMux()
	m.Handle("/metrics", instr.WrapHandler("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	)))
	srv := http.Server{Addr: *addr, Handler: m}

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
	{
		client := &http.Client{
			// Custom HTTP client with metrics and tracing instrumentation.
			Transport: exthttp.NewInstrumentationTripperware(reg, nil, tracingProvider).
				WrapRoundTripper("ping", http.DefaultTransport),
		}

		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			spamPings(ctx, client, *endpoint, *pingsPerSec)
			return nil
		}, func(error) {
			cancel()
		})
	}
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	return g.Run()
}

func spamPings(ctx context.Context, client *http.Client, endpoint string, pingsPerSec int) {
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-time.After(1 * time.Second):
		}

		for i := 0; i < pingsPerSec; i++ {
			wg.Add(1)
			go ping(ctx, client, endpoint, &wg)
		}
	}
}

func ping(ctx context.Context, client *http.Client, endpoint string, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return
	}
	res, err := client.Do(r)
	if err != nil {
		fmt.Println("Failed to send request:", err)
		return
	}
	if res.Body != nil {
		// We don't care about response, just release resources.
		_, _ = io.Copy(ioutil.Discard, res.Body)
		_ = res.Body.Close()
	}
}
