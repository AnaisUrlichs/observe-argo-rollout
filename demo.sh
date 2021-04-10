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

# Service commands, comment out if needed.
#p "# Verify plugin is installed"
#r "kubectl argo rollouts version"
#p "# Purge Demo Resources (e.g for demo restart)."
#r "kubectl delete namespace ${NAMESPACE}"

NAMESPACE="demo"

# https://asciiflow.com/#/share/eJzNWMFKAzEQ%2FZUhB2mhBxVR9CJFRQQFqXrbS1zHtrDdrLOptIgg4tGDh1J78OjRk3gSv8YvMaV11bpJN93UGkI32WTfm5kkb0IvWMgbyNZYBc%2BRJGxiQ7ASC3gbSb298FjLY2ury0slj7VVa3FlWbUktqTqeAzGlvfO7XvnylG987wwC%2BW1YWgWENvET3nI9RAuovSot%2BRhekugi9TwfeFopwSbPK4dC04ncdH0iSYyRnz7oX%2BD33G3KL389tiXFOzumyvs7pvCdqsdP22fAHnUOfXG7bnKvudSZqbaZ4BzwVeOItjlEkO%2FvW7Jd4iNSJj5ylQVUBFBIJryi2%2BLCEhxrn%2Fj2yfRQFnDZuzUP4vafbXxvnBI3MeBGmqtKSSug351C3soqe7HJSgHKn%2FXw2pxUu83RChJUSIVZ7ObrPmmpaB65KcpqIMB88u8X0r7rAuOKcqGaNp%2B2Z%2Bzp%2B40VQQengCJpvp1h97PJCSO1X4GwrMmxjJOnfe3%2BS3B1uePu9kMZrT6RytPNztfX7O%2FQUzYVY%2F7mzTfH%2F%2BRtzrV6M1mMJPVecuk53u8fuVAT%2BYszEPU1xDX6IkmRUgQoy9SxM8SXXO8X8Yd%2BPx%2BjQ6ODUIavrNcnPDvq4VT0f3sa85%2FnjqaVYelgj7WzxEGlzRjjA584hHC8BIGc7DVUvfagFOc6T8C69j1PHbJLj8AMPc2MQ%3D%3D)

r "kubectl create namespace ${NAMESPACE}"

p "# Install monitoring resources ${NAMESPACE}. It's critical to have 👀  first!"
p "$(cat demo-arch-monitoring.txt)"
r "kubectl apply -n ${NAMESPACE} -f manifests/generated/monitoring"
r "kubectl get -n ${NAMESPACE} po"

p "# Install argo rollout controller with CRDs to ${NAMESPACE}"
r "kubectl apply -n ${NAMESPACE} -f manifests/argo-rollouts"

p "# Argo Rollout defines Rollout CR that replaces Deployment object and allows to deploy our anaisurlichs/ping-pong:initial"
r "bat -r 1:38 manifests/application/app.yaml"

p "# It references 'low-error-low-latency Analysis template:"
r "bat manifests/application/analysis-template.yaml"

p "# Rollout anaisurlichs/ping-pong:initial managed by Argo Rollout"
p "$(cat demo-arch-apps.txt)"
r "kubectl apply -n ${NAMESPACE} -f manifests/application"

p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

p "# Start client 'pinger'"
p "$(cat demo-arch-all.txt)"
r "kubectl apply -n ${NAMESPACE} -f manifests/generated/pinger"
r "kubectl get -n ${NAMESPACE} po"

p "# Rollout anaisurlichs/ping-pong:errors"
r "kubectl argo rollouts -n ${NAMESPACE} set image app app=anaisurlichs/ping-pong:errors"

p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

p "# Rollout anaisurlichs/ping-pong:slow"
r "kubectl argo rollouts -n ${NAMESPACE} set image app app=anaisurlichs/ping-pong:slow"

p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

p "# Rollout anaisurlichs/ping-pong:best"
r "kubectl argo rollouts -n ${NAMESPACE} set image app app=anaisurlichs/ping-pong:best"

p "# Ensure initial Rollout happened correctly"
r "kubectl argo rollouts -n ${NAMESPACE} get rollout app"

# Last entry to run navigation mode.
# True means we will continue from last step.
navigate true