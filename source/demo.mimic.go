// Copyright (c) bwplotka/mimic Authors
// Licensed under the Apache License 2.0.

package main

import (
	"fmt"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/encoding"
)

const (
	selectorName = "app.kubernetes.io/name"
	namespace    = "demo"
)

func main() {
	generator := mimic.New()

	// Make sure to generate at the very end.
	defer generator.Generate()

	{
		g := generator.With("monitoring")

		promSet, promSrv, promConfigAndMount := getPrometheus("prom")

		g.Add("prom.yaml", encoding.GhodssYAML(promSet, promSrv, promConfigAndMount))
		g.Add("grafana.yaml", encoding.GhodssYAML(
			getGrafana(
				fmt.Sprintf("%s.%s.svc.cluster.local:%d", promSrv.Name, namespace, promSrv.Spec.Ports[0].Port),
				fmt.Sprintf("tempo.%s.svc.cluster.local:9090", namespace),
				"grafana")),
		)
		g.Add("tempo.yaml", encoding.GhodssYAML(getTempo("tempo")))
	}
	{
		g := generator.With("pinger")
		g.Add("pinger.yaml", encoding.GhodssYAML(
			getPinger(fmt.Sprintf("http://app.%s.svc.cluster.local:8080/ping", namespace), "pinger")),
		)
	}
}
