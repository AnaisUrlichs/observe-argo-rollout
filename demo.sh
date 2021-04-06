#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Import bash library: https://github.com/bwplotka/demo-nav
. "${DIR}/demo-nav.sh"

clear

# Tutorial for this script.
# `r` registers command to be invoked.
#
# First argument specifies what should be printed.
# Second argument specifies what will be actually executed.
#
# NOTE: Use `'` to quote strings inside command.

# 'rc' is like r but does not leave executed command printed.
# 'p' print line with next command, useful for comments.
# See example https://github.com/bwplotka/demo-nav/blob/master/example/demo-example.sh

# This scripts assumes you have:
# * Argo plugin 0.10.2 installed https://argoproj.github.io/argo-rollouts/installation/#kubectl-plugin-installation
# * Kubernetes 1.18 running.

p "# Verify plugin is installed"
r "kubectl argo rollouts version"

NAMESPACE="demo"

p "# Purge Demo Resources (e.g for demo restart)."
r "kubectl delete namespace ${NAMESPACE}"
r "kubectl create namespace ${NAMESPACE}"

p "# Install argo rollout controller with CRDs to ${NAMESPACE}"
r "kubectl apply -n ${NAMESPACE} -f manifests/argo-rollouts"

p "# Install monitoring resources ${NAMESPACE}"
r "kubectl apply -n ${NAMESPACE} -f manifests/generated/monitoring"
r "kubectl get -n ${NAMESPACE} po"

# Cat/tail/head some part of deployment manifest to how things are configured?
p "# Rollout anaisurlichs/ping-pong:initial"
r "kubectl apply -n ${NAMESPACE} -f manifests/application"

p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

p "# Start client 'pinger'"
r "kubectl apply -n ${NAMESPACE} -f manifests/generated/pinger"
r "kubectl get -n ${NAMESPACE} po"

p "# Rollout anaisurlichs/ping-pong:error"
r "kubectl argo rollouts set image app app=anaisurlichs/ping-pong:error"

# TODO(bwplotka): Wait?
p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

p "# Rollout anaisurlichs/ping-pong:slow"
r "kubectl argo rollouts set image app app=anaisurlichs/ping-pong:slow"

# TODO(bwplotka): Wait?
p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

p "# Rollout anaisurlichs/ping-pong:best"
r "kubectl argo rollouts set image app app=anaisurlichs/ping-pong:best"

# TODO(bwplotka): Wait?
p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

# Last entry to run navigation mode.
navigate