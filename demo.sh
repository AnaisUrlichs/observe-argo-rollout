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

p "# Install Prometheus ${NAMESPACE}"
r "kubectl apply -n ${NAMESPACE} -f manifests/generated/monitoring"

r "kubectl get -n ${NAMESPACE} po"

# Last entry to run navigation mode.
navigate