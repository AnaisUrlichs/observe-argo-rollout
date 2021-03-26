#!/usr/bin/env bash

# We also don't want to fix index.json and allow-list all files required for so we directly clone repo as first step.
# YOLO!

curl -LO https://github.com/argoproj/argo-rollouts/releases/download/v0.10.2/kubectl-argo-rollouts-linux-amd64
chmod +x ./kubectl-argo-rollouts-linux-amd64
sudo mv ./kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts

cd /root
git clone https://github.com/bwplotka/observe-argo-rollout.git -b /root/demo
cd /root/demo

# Start k8s.
launch.sh

