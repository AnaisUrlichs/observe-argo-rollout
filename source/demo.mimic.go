// Copyright (c) bwplotka/mimic Authors
// Licensed under the Apache License 2.0.

package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/encoding"
	"github.com/bwplotka/mimic/lib/abstr/kubernetes/volumes"
	"github.com/bwplotka/mimic/lib/schemas/prometheus"
	sdconfig "github.com/bwplotka/mimic/lib/schemas/prometheus/discovery/config"
	"github.com/bwplotka/mimic/lib/schemas/prometheus/discovery/kubernetes"
	"github.com/go-openapi/swag"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	selectorName = "app.kubernetes.io/name"
)

func main() {
	generator := mimic.New()

	// Make sure to generate at the very end.
	defer generator.Generate()

	genPrometheus(generator.With("prometheus"), "prom")
}

func genPrometheus(generator *mimic.Generator, name string) {
	const (
		replicas = 1

		configVolumeName  = "prometheus-config"
		configVolumeMount = "/etc/prometheus"

		httpPort = 9090
	)

	promConfigAndMount := volumes.ConfigAndMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configVolumeName,
			Labels: map[string]string{selectorName: name},
		},
		VolumeMount: corev1.VolumeMount{Name: configVolumeName, MountPath: configVolumeMount},
		Data: map[string]string{
			"prometheus.yaml": EncodeYAML(prometheusConfig()),
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
					Name:       "http",
					Port:       httpPort,
					TargetPort: intstr.FromInt(httpPort),
				},
			},
		},
	}

	prometheusContainer := corev1.Container{
		Name:  "prometheus",
		Image: "prom/prometheus:v2.26.0",
		Args: []string{
			fmt.Sprintf("--config.file=%v/prometheus.yaml", configVolumeMount),
			"--log.level=info",
			"--storage.tsdb.retention.time=2d",
			"--web.enable-lifecycle",
			"--web.enable-admin-api",
			fmt.Sprintf("--web.external-url=https://localhost:%v", httpPort),
		},
		Env: []corev1.EnvVar{
			{Name: "HOSTNAME", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			}},
		},
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt(httpPort),
					Path: "-/ready",
				},
			},
			SuccessThreshold: 3,
		},
		Ports:        []corev1.ContainerPort{{Name: "m-http", ContainerPort: httpPort}},
		VolumeMounts: volumes.VolumesAndMounts{promConfigAndMount.VolumeAndMount()}.VolumeMounts(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: swag.Bool(false),
			RunAsUser:    swag.Int64(1000),
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
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
			Replicas:    swag.Int32(replicas),
			ServiceName: name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{selectorName: name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{prometheusContainer},
					Volumes:    volumes.VolumesAndMounts{promConfigAndMount.VolumeAndMount()}.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{selectorName: name},
			},
		},
	}

	generator.Add(name+".yaml", encoding.GhodssYAML(set, srv, promConfigAndMount.ConfigMap()))
}

func EncodeYAML(in ...interface{}) string {
	cfgBytes, err := ioutil.ReadAll(encoding.YAML(in))
	mimic.PanicIfErr(err)
	return string(cfgBytes)
}
func prometheusConfig() prometheus.Config {
	c := prometheus.Config{
		GlobalConfig: prometheus.GlobalConfig{
			ExternalLabels: map[model.LabelName]model.LabelValue{
				"cluster": "demo",
			},
			ScrapeInterval: model.Duration(15 * time.Second),
		},
		ScrapeConfigs: []*prometheus.ScrapeConfig{
			{
				JobName: "kube-api",
				Scheme:  "https",
				ServiceDiscoveryConfig: sdconfig.ServiceDiscoveryConfig{
					KubernetesSDConfigs: []*kubernetes.SDConfig{{Role: kubernetes.RoleEndpoint}},
				},
				HTTPClientConfig: config.HTTPClientConfig{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					TLSConfig: config.TLSConfig{
						CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
					},
				},
				RelabelConfigs: []*prometheus.RelabelConfig{
					{
						SourceLabels: model.LabelNames{"__meta_kubernetes_namespace", "__meta_kubernetes_service_name", "__meta_kubernetes_endpoint_port_name"},
						Action:       prometheus.RelabelKeep,
						Regex:        prometheus.MustNewRegexp("default;kubernetes;https"),
					},
				},
			},
			{
				JobName: "kube-nodes",
				Scheme:  "https",
				ServiceDiscoveryConfig: sdconfig.ServiceDiscoveryConfig{
					KubernetesSDConfigs: []*kubernetes.SDConfig{{Role: kubernetes.RoleNode}},
				},
				HTTPClientConfig: config.HTTPClientConfig{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					TLSConfig: config.TLSConfig{
						CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
					},
				},
				RelabelConfigs: []*prometheus.RelabelConfig{
					{
						Action: prometheus.RelabelLabelMap,
						Regex:  prometheus.MustNewRegexp("__meta_kubernetes_node_label_(.+)"),
					},
					{
						TargetLabel: model.AddressLabel,
						Replacement: "kubernetes.default.svc:443",
					},
					{
						SourceLabels: model.LabelNames{"__meta_kubernetes_node_name"},
						Regex:        prometheus.MustNewRegexp("(.+)"),
						TargetLabel:  model.MetricsPathLabel,
						Replacement:  "/api/v1/nodes/${1}/proxy/metrics",
					},
				},
			},
			{
				// This is required for Kubernetes 1.7.3 and later, where cAdvisor metrics
				// (those whose names begin with 'container_') have been removed from the
				// Kubelet metrics endpoint.  This job scrapes the cAdvisor endpoint to
				// retrieve those metrics.
				//
				// In Kubernetes 1.7.0-1.7.2, these metrics are only exposed on the cAdvisor
				// HTTP endpoint; use "replacement: /api/v1/nodes/${1}:4194/proxy/metrics"
				// in that case (and ensure cAdvisor's HTTP server hasn't been disabled with
				// the --cadvisor-port=0 Kubelet flag).
				//
				// This job is not necessary and should be removed in Kubernetes 1.6 and
				// earlier versions, or it will cause the metrics to be scraped twice.
				JobName: "kube-cadvisor",
				Scheme:  "https",
				ServiceDiscoveryConfig: sdconfig.ServiceDiscoveryConfig{
					KubernetesSDConfigs: []*kubernetes.SDConfig{{Role: kubernetes.RoleNode}},
				},
				HTTPClientConfig: config.HTTPClientConfig{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					TLSConfig: config.TLSConfig{
						CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
					},
				},
				RelabelConfigs: []*prometheus.RelabelConfig{
					{
						Action: prometheus.RelabelLabelMap,
						Regex:  prometheus.MustNewRegexp("__meta_kubernetes_node_label_(.+)"),
					},
					{
						TargetLabel: model.AddressLabel,
						Replacement: "kubernetes.default.svc:443",
					},
					{
						SourceLabels: model.LabelNames{"__meta_kubernetes_node_name"},
						Regex:        prometheus.MustNewRegexp("(.+)"),
						TargetLabel:  model.MetricsPathLabel,
						Replacement:  "/api/v1/nodes/${1}/proxy/metrics/cadvisor",
					},
				},
			},
			{
				JobName: "kube-pods",
				ServiceDiscoveryConfig: sdconfig.ServiceDiscoveryConfig{
					KubernetesSDConfigs: []*kubernetes.SDConfig{{Role: kubernetes.RolePod}},
				},
				RelabelConfigs: []*prometheus.RelabelConfig{
					{
						// To have pod port scraped, container has to have port name started with m-.
						SourceLabels: model.LabelNames{"__meta_kubernetes_pod_container_port_name"},
						Action:       prometheus.RelabelKeep,
						Regex:        prometheus.MustNewRegexp("m-.+"),
					},
					{
						SourceLabels: model.LabelNames{"__meta_kubernetes_pod_label_" + selectorName},
						Action:       prometheus.RelabelReplace,
						TargetLabel:  "job",
					},
					{
						SourceLabels: model.LabelNames{"__meta_kubernetes_pod_name"},
						Action:       prometheus.RelabelReplace,
						TargetLabel:  "pod",
					},
					{
						SourceLabels: model.LabelNames{"__meta_kubernetes_namespace"},
						Action:       prometheus.RelabelReplace,
						TargetLabel:  "namespace",
					},
				},
			},
		},
	}
	return c
}
