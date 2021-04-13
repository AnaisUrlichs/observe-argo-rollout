// Copyright (c) bwplotka/mimic Authors
// Licensed under the Apache License 2.0.

package main

import (
	"fmt"
	"time"

	"github.com/bwplotka/mimic/lib/abstr/kubernetes/volumes"
	"github.com/go-openapi/swag"
	"github.com/grafana/tempo/modules/frontend"
	"github.com/grafana/tempo/modules/querier"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/cache/memcached"
	"github.com/grafana/tempo/tempodb/backend/cache/redis"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/pool"
	"github.com/grafana/tempo/tempodb/wal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TODO(bwplotka): Instead of importing whole struct, I redefined all here. Reason is:
// * Those structs are not prepared to be marshalled to, e.g `omitempty` are omitted, causing empty strings being marshaled.
//   This is then wrongly parsed by Tempo.
// Long Term Solution: Create tool for getting those structs from exact version, with only deps needed, with omitempty etc?
type TempoConfig struct {
	Target      string `yaml:"target,omitempty"`
	AuthEnabled bool   `yaml:"auth_enabled"`
	HTTPPrefix  string `yaml:"http_prefix,omitempty"`

	Server        TempoServerConfig      `yaml:"server,omitempty"`
	Ingester      TempoIngesterConfig    `yaml:"ingester,omitempty"`
	Distributor   TempoDistributorConfig `yaml:"distributor,omitempty"`
	Compactor     TempoCompactorConfig   `yaml:"compactor,omitempty"`
	StorageConfig TempoStorageConfig     `yaml:"storage,omitempty"`
	Querier       querier.Config         `yaml:"querier,omitempty"`
	Frontend      frontend.Config        `yaml:"query_frontend,omitempty"`
}

type TempoServerConfig struct {
	HTTPListenAddress string `yaml:"http_listen_address,omitempty"`
	HTTPListenPort    int    `yaml:"http_listen_port,omitempty"`
}

// Config for an ingester.
type TempoIngesterConfig struct {
	ConcurrentFlushes    int           `yaml:"concurrent_flushes,omitempty"`
	FlushCheckPeriod     time.Duration `yaml:"flush_check_period,omitempty"`
	FlushOpTimeout       time.Duration `yaml:"flush_op_timeout,omitempty"`
	MaxTraceIdle         time.Duration `yaml:"trace_idle_period,omitempty"`
	MaxBlockDuration     time.Duration `yaml:"max_block_duration,omitempty"`
	MaxBlockBytes        uint64        `yaml:"max_block_bytes,omitempty"`
	CompleteBlockTimeout time.Duration `yaml:"complete_block_timeout,omitempty"`
	OverrideRingKey      string        `yaml:"override_ring_key,omitempty"`
}

// Config for a Distributor.
type TempoDistributorConfig struct {
	//  This receivers node is equivalent in format to the receiver node in the
	//  otel collector: https://github.com/open-telemetry/opentelemetry-collector/tree/master/receiver
	Receivers map[string]interface{} `yaml:"receivers,omitempty"`
}

type TempoStorageConfig struct {
	Trace TempoStorageConfigConfig `yaml:"trace"`
}

type TempoStorageConfigConfig struct {
	Pool  *pool.Config          `yaml:"pool,omitempty"`
	WAL   *wal.Config           `yaml:"wal,omitempty"`
	Block *encoding.BlockConfig `yaml:"block,omitempty"`

	BlocklistPoll            time.Duration `yaml:"blocklist_poll,omitempty"`
	BlocklistPollConcurrency uint          `yaml:"blocklist_poll_concurrency,omitempty"`

	// backends
	Backend string        `yaml:"backend,omitempty"`
	Local   *local.Config `yaml:"local,omitempty"`
	GCS     *gcs.Config   `yaml:"gcs,omitempty"`
	S3      *s3.Config    `yaml:"s3,omitempty"`
	Azure   *azure.Config `yaml:"azure,omitempty"`

	// caches
	Cache     string            `yaml:"cache,omitempty"`
	Memcached *memcached.Config `yaml:"memcached,omitempty"`
	Redis     *redis.Config     `yaml:"redis,omitempty"`
}

type TempoCompactorConfig struct {
	Compactor TempoCompactionConfig `yaml:"compaction"`
}

// CompactorConfig contains compaction configuration options
type TempoCompactionConfig struct {
	ChunkSizeBytes          uint32        `yaml:"chunk_size_bytes,omitempty"`
	FlushSizeBytes          uint32        `yaml:"flush_size_bytes,omitempty"`
	MaxCompactionRange      time.Duration `yaml:"compaction_window,omitempty"`
	MaxCompactionObjects    int           `yaml:"max_compaction_objects,omitempty"`
	MaxBlockBytes           uint64        `yaml:"max_block_bytes,omitempty"`
	BlockRetention          time.Duration `yaml:"block_retention,omitempty"`
	CompactedBlockRetention time.Duration `yaml:"compacted_block_retention,omitempty"`
	RetentionConcurrency    uint          `yaml:"retention_concurrency,omitempty"`
}

func getTempo(name string) (appsv1.StatefulSet, corev1.Service, corev1.ConfigMap) {
	const (
		configVolumeName  = "tempo-config"
		configVolumeMount = "/etc/tempo"
		dataPath          = "/data"
		httpPort          = 9090
		grpcPort          = 9091
	)

	configAndMount := volumes.ConfigAndMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configVolumeName,
			Labels: map[string]string{selectorName: name},
		},
		VolumeMount: corev1.VolumeMount{Name: configVolumeName, MountPath: configVolumeMount},
		Data: map[string]string{
			"tempo.yaml": EncodeYAML(TempoConfig{
				AuthEnabled: false,
				Server: TempoServerConfig{
					HTTPListenPort: httpPort,
				},
				Ingester: TempoIngesterConfig{
					MaxTraceIdle: 10 * time.Second,
					// Normally it should be larger, around 10GB. For quick demo purposes, let's make it small to fit Katacoda environment.
					MaxBlockBytes:    1e6,
					MaxBlockDuration: 5 * time.Minute,
				},
				Distributor: TempoDistributorConfig{
					Receivers: map[string]interface{}{
						"otlp": map[string]interface{}{
							"protocols": map[string]interface{}{
								"grpc": map[string]interface{}{
									"endpoint": fmt.Sprintf("0.0.0.0:%v", grpcPort),
								},
							},
						},
					},
				},
				Compactor: TempoCompactorConfig{
					Compactor: TempoCompactionConfig{
						MaxCompactionRange:      1 * time.Hour,
						MaxBlockBytes:           1e6,
						BlockRetention:          1 * time.Hour,
						CompactedBlockRetention: 10 * time.Minute,
					},
				},
				StorageConfig: TempoStorageConfig{
					Trace: TempoStorageConfigConfig{
						Backend: "local",
						Block: &encoding.BlockConfig{
							BloomFP:              .05,
							IndexDownsampleBytes: 1024 * 1024,
							IndexPageSizeBytes:   250 * 1024,
							Encoding:             backend.EncZstd,
						},
						WAL:   &wal.Config{Filepath: "/data/wal"},
						Local: &local.Config{Path: "/data/blocks"},
						Pool: &pool.Config{
							MaxWorkers: 100,
							QueueDepth: 10000,
						},
					},
				},
			}),
		},
	}

	srv := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{selectorName: name},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{selectorName: name},
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       grpcPort,
					TargetPort: intstr.FromInt(grpcPort),
				},
				{
					Name:       "http",
					Port:       httpPort,
					TargetPort: intstr.FromInt(httpPort),
				},
			},
		},
	}

	dataVM := volumes.VolumeAndMount{
		VolumeMount: corev1.VolumeMount{
			Name:      name,
			MountPath: dataPath,
		},
	}

	container := corev1.Container{
		Name:  "tempo",
		Image: "grafana/tempo:93c378a9",
		Args: []string{
			"-config.file=/etc/tempo/tempo.yaml",
		},
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt(httpPort),
					Path: "/metrics",
				},
			},
			SuccessThreshold: 3,
		},
		Ports: []corev1.ContainerPort{
			{Name: "m-http", ContainerPort: httpPort},
			{Name: "grpc", ContainerPort: grpcPort},
		},
		VolumeMounts: volumes.VolumesAndMounts{configAndMount.VolumeAndMount(), dataVM}.VolumeMounts(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: swag.Bool(false),
			RunAsUser:    swag.Int64(1000),
		},
	}

	set := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{selectorName: name},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    swag.Int32(1),
			ServiceName: name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{selectorName: name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
					Volumes:    volumes.VolumesAndMounts{configAndMount.VolumeAndMount(), dataVM}.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{selectorName: name},
			},
		},
	}
	return set, srv, configAndMount.ConfigMap()
}
