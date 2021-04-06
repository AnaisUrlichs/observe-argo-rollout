package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"syscall"
	"time"

	"github.com/AnaisUrlichs/observe-argo-rollout/app/extpromhttp"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

var (
	// TODO(bwplotka): Move those flags out of globals.
	addr       = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	appVersion = flag.String("set-version", "first", "Injected version to be presented via metrics.")
	lat        = flag.Int("lat", 0, "The latency of the response in milliseconds")
	prob       = flag.Int("prob", 100, "The probability (in %) of getting a successful response")
)

func handlerPing(w http.ResponseWriter, r *http.Request) {
	n := rand.Intn(100)

	<-time.After(time.Duration(*lat) * time.Millisecond)
	if n <= *prob {
		w.Write([]byte("pong"))
	} else {
		w.WriteHeader(500)
	}
}

func main() {
	flag.Parse()

	version.Version = *appVersion
	version.BuildUser = "Probably Anaïs"

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		version.NewCollector("app"),
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	instr := extpromhttp.NewInstrumentationMiddleware(reg, nil)
	m := http.NewServeMux()
	m.Handle("/metrics", instr.NewHandler("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	)))
	m.HandleFunc("/ping", instr.NewHandler("/ping", http.HandlerFunc(handlerPing)))
	srv := http.Server{Addr: *addr, Handler: m}

	// Setup multiple 2 jobs. One is for serving HTTP requests, second to listen for Linux signals like Ctrl+C.
	g := &run.Group{}
	g.Add(func() error {
		fmt.Println("ping listening on localhost:8080")
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
	if err := g.Run(); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		log.Fatalf("Error: %+v", errors.Wrapf(err, "%s failed", flag.Arg(0)))
	}
}
