#!/usr/bin/env bash

# Setups done, before setup.
# TODO: Probably for Katacoda demo purposes we maybe need to write this down and allow user to do it one by one.

ARGO_ROLLOUTS_VERSION="v0.10.2"

# Install kubectl argo rollout plugin.
curl -LO https://github.com/argoproj/argo-rollouts/releases/${ARGO_ROLLOUTS_VERSION}/download/kubectl-argo-rollouts-linux-amd64
chmod +x ./kubectl-argo-rollouts-linux-amd64
sudo mv ./kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts

# Verify.
kubectl argo rollouts version

launch.sh

kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://raw.githubusercontent.com/argoproj/argo-rollouts/stable/manifests/install.yaml

