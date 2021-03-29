#!/usr/bin/env bash

# This scripts assumes you have:
# * Argo plugin 0.10.2 installed https://argoproj.github.io/argo-rollouts/installation/#kubectl-plugin-installation
# * Kubernetes 1.18 running.

# Verify plugin is installed.
kubectl argo rollouts version

kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f manifests/argo-rollouts



