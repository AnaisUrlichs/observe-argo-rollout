package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
	latDecider *latencyDecider

	// TODO(bwplotka): Move those flags out of globals.
	addr        = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	appVersion  = flag.String("set-version", "first", "Injected version to be presented via metrics.")
	lat         = flag.String("latency", "90%500ms,10%200ms", "Encoded latency and probability of the response in format as: <probability>%<duration>,<probability>%<duration>....")
	successProb = flag.Float64("success-prob", 100, "The probability (in %) of getting a successful response")
)

type latencyDecider struct {
	latencies   []time.Duration
	probability []float64 // Sorted ascending.
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
		l.probability = append(l.probability, f)

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

func (l latencyDecider) AddLatency() {
	n := rand.Float64() * 100

	for i, p := range l.probability {
		if n <= p {
			<-time.After(l.latencies[i])
			return
		}
	}
}

func handlerPing(w http.ResponseWriter, _ *http.Request) {
	latDecider.AddLatency()

	if rand.Float64()*100 <= *successProb {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("pong"))
		return
	}
	w.WriteHeader(500)
}

func main() {
	var err error
	flag.Parse()

	latDecider, err = newLatencyDecider(*lat)
	if err != nil {
		log.Fatal(err)
	}

	version.Version = *appVersion
	version.BuildUser = "Anaïs"

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
