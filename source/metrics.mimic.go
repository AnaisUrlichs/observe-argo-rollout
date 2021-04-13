// Copyright (c) bwplotka/mimic Authors
// Licensed under the Apache License 2.0.

package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
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

func getPrometheus(name string) (appsv1.StatefulSet, corev1.Service, corev1.ConfigMap) {
	const (
		configVolumeName  = "prometheus-config"
		configVolumeMount = "/etc/prometheus"
		dataPath          = "/data"
		httpPort          = 9090
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
			Type:     corev1.ServiceTypeNodePort,
			Selector: map[string]string{selectorName: name},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       httpPort,
					TargetPort: intstr.FromInt(httpPort),
					NodePort:   30555,
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
		Name: "prometheus",
		// Image: Prometheus master + https://github.com/prometheus/prometheus/pull/8712
		Image: "bplotka/prometheus:2.26.0-exemplars-metrics",
		Args: []string{
			fmt.Sprintf("--config.file=%v/prometheus.yaml", configVolumeMount),
			"--log.level=info",
			"--storage.tsdb.retention.time=2d",
			fmt.Sprintf("--storage.tsdb.path=%s", dataPath),
			"--web.enable-lifecycle",
			"--web.enable-admin-api",
			// Give me all the features!
			"--enable-feature=promql-at-modifier",
			"--enable-feature=promql-negative-offset",
			"--enable-feature=exemplar-storage",
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
		VolumeMounts: volumes.VolumesAndMounts{promConfigAndMount.VolumeAndMount(), dataVM}.VolumeMounts(),
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
			Replicas:    swag.Int32(1),
			ServiceName: name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{selectorName: name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
					Volumes:    volumes.VolumesAndMounts{promConfigAndMount.VolumeAndMount(), dataVM}.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{selectorName: name},
			},
		},
	}

	return set, srv, promConfigAndMount.ConfigMap()
}

func getGrafana(promURL, tempoURL string, name string) (appsv1.Deployment, corev1.Service, corev1.ConfigMap, corev1.ConfigMap, corev1.ConfigMap) {
	const (
		configVolumeName  = "grafana-config"
		configVolumeMount = "/etc/grafana"
		httpPort          = 3000
	)

	var (
		// Grafana has those hardcoded or something...
		// https://grafana.com/tutorials/provision-dashboards-and-data-sources/
		datVolumeMount  = filepath.Join(configVolumeMount, "provisioning", "datasources")
		dashVolumeMount = filepath.Join(configVolumeMount, "provisioning", "dashboards")
	)

	cfgCM := volumes.ConfigAndMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configVolumeName,
			Labels: map[string]string{selectorName: name},
		},
		VolumeMount: corev1.VolumeMount{Name: configVolumeName, MountPath: configVolumeMount},
		Data: map[string]string{
			"grafana.ini": `
[paths]
provisioning = /etc/grafana/provisioning
data = /var/lib/grafana
logs = /var/log/grafana
[auth.basic]
enabled = false
[auth.anonymous]
# enable anonymous access
enabled = true
org_role = Admin
[analytics]
reporting_enabled = false
check_for_updates = false
[users]
default_theme = dark
[security]
allow_embedding = true`,
		},
	}

	dsCM := volumes.ConfigAndMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configVolumeName + "ds",
			Labels: map[string]string{selectorName: name},
		},
		VolumeMount: corev1.VolumeMount{Name: configVolumeName + "ds", MountPath: datVolumeMount},
		Data: map[string]string{
			"datasource.yml": `apiVersion: 1

datasources:
- name: Prometheus
  type: prometheus
  uid: Prom1
  access: proxy
  orgId: 1
  url: http://` + promURL + `
  isDefault: true
  editable: true
  httpMethod: POST
  version: 1
  jsonData:
    exemplarTraceIdDestinations:
      # Field with internal link pointing to data source in Grafana.
      # datasourceUid value can be anything, but it should be unique across all defined data source uids.
      - datasourceUid: tempo1
        name: traceID

- name: Tempo
  type: tempo
  uid: tempo1
  # access: proxy
  orgId: 1
  url: http://` + tempoURL + `
  editable: true
`,
		},
	}

	dshCM := volumes.ConfigAndMount{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configVolumeName + "dsh",
			Labels: map[string]string{selectorName: name},
		},
		VolumeMount: corev1.VolumeMount{Name: configVolumeName + "dsh", MountPath: dashVolumeMount},
		Data: map[string]string{
			"dashboard.yml": `apiVersion: 1

providers:
- name: 'Demo'
  orgId: 1
  folder: ''
  type: file
  disableDeletion: false
  editable: true
  options:
    path: /etc/grafana/provisioning/dashboards
`,
			"demo.json": demoDashboard(),
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
			Type:     corev1.ServiceTypeNodePort,
			Selector: map[string]string{selectorName: name},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       httpPort,
					TargetPort: intstr.FromInt(httpPort),
					NodePort:   30556,
				},
			},
		},
	}

	container := corev1.Container{
		Name:            "grafana",
		Image:           "grafana/grafana:7.5.0",
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt(httpPort),
					Path: "/robots.txt",
				},
			},
			SuccessThreshold: 3,
		},
		Args:         []string{"--config=" + filepath.Join(configVolumeMount, "grafana.ini")},
		Ports:        []corev1.ContainerPort{{Name: "m-http", ContainerPort: httpPort}},
		VolumeMounts: volumes.VolumesAndMounts{cfgCM.VolumeAndMount(), dsCM.VolumeAndMount(), dshCM.VolumeAndMount()}.VolumeMounts(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot: swag.Bool(true),
			RunAsUser:    swag.Int64(65534),
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
		},
	}

	dpl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{selectorName: name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: swag.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{selectorName: name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
					Volumes:    volumes.VolumesAndMounts{cfgCM.VolumeAndMount(), dsCM.VolumeAndMount(), dshCM.VolumeAndMount()}.Volumes(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{selectorName: name},
			},
		},
	}

	return dpl, srv, cfgCM.ConfigMap(), dsCM.ConfigMap(), dshCM.ConfigMap()
}

func EncodeYAML(in ...interface{}) string {
	cfgBytes, err := ioutil.ReadAll(encoding.YAML(in...))
	mimic.PanicIfErr(err)
	return string(cfgBytes)
}

func prometheusConfig() prometheus.Config {
	c := prometheus.Config{
		GlobalConfig: prometheus.GlobalConfig{
			ExternalLabels: map[model.LabelName]model.LabelValue{
				"cluster": "demo",
			},
			// For demo purposes do scrapes much more often. Normally 15s is ok.
			ScrapeInterval: model.Duration(5 * time.Second),
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
						SourceLabels: model.LabelNames{"__meta_kubernetes_pod_label_app_kubernetes_io_name"},
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
