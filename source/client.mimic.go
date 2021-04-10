package main

import (
	"fmt"

	"github.com/go-openapi/swag"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getPinger(endpoint string, name string) appsv1.Deployment {
	const httpPort = 80

	return appsv1.Deployment{
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
					Containers: []corev1.Container{{
						Name:            "pinger",
						Image:           "anaisurlichs/ping-pong:latest",
						ImagePullPolicy: corev1.PullAlways,
						Command:         []string{"/bin/pinger"},
						Args: []string{
							"-endpoint=" + endpoint,
							fmt.Sprintf("-listen-address=:%v", httpPort),
							"-pings-per-second=10",
						},
						Ports: []corev1.ContainerPort{{Name: "m-http", ContainerPort: httpPort}},
					}},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{selectorName: name},
			},
		},
	}
}
