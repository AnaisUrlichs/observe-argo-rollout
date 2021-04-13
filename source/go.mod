module github.com/AnaisUrlichs/observe-argo-rollout/source

go 1.15

require (
	github.com/bwplotka/mimic v0.1.0
	github.com/bwplotka/mimic/lib/abstr/kubernetes v0.0.0-20210402175152-95ed29263523
	github.com/bwplotka/mimic/lib/schemas/prometheus v0.0.0-20210402175152-95ed29263523
	github.com/go-openapi/swag v0.19.15
	github.com/grafana/tempo v0.6.1-0.20210405161918-019d3a4f04c2
	github.com/prometheus/common v0.20.0
	github.com/tdewolff/minify/v2 v2.9.15
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
)

// All of the below replace directives exist due to
//   Cortex -> ETCD -> GRPC requiring 1.29.1
//   Otel Collector -> requiring 1.30.1
//  Once this is merged: https://github.com/etcd-io/etcd/pull/12155 and Cortex revendors we should be able to update everything to current
replace (
	github.com/gocql/gocql => github.com/grafana/gocql v0.0.0-20200605141915-ba5dc39ece85
	github.com/sercand/kuberesolver => github.com/sercand/kuberesolver v2.4.0+incompatible
	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20200520232829-54ba9589114f
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)

replace (
	github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab
	github.com/opentracing-contrib/go-grpc => github.com/pracucci/go-grpc v0.0.0-20201022134131-ef559b8db645
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20210124145330-b5dfa2414b9e
	github.com/satori/go.uuid => github.com/satori/go.uuid v1.2.0
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.2
)

// Pin github.com/go-openapi versions to match Prometheus alertmanager to avoid
// breaking changing affecting the alertmanager.
replace (
	github.com/go-openapi/errors => github.com/go-openapi/errors v0.19.4
	github.com/go-openapi/validate => github.com/go-openapi/validate v0.19.8
)

// Pin github.com/soheilhy/cmux to control grpc required version.
// Before v0.1.5 it contained examples in the root folder that imported grpc without a version,
// and therefore were importing grpc latest (which is problematic because we need <v1.29.1)
replace github.com/soheilhy/cmux => github.com/soheilhy/cmux v0.1.5
