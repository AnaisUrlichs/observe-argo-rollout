#!/usr/bin/env bash

# We also don't want to fix index.json and allow-list all files required for so we directly clone repo as first step.
# YOLO!

curl -LO https://github.com/argoproj/argo-rollouts/releases/download/v0.10.2/kubectl-argo-rollouts-linux-amd64
chmod +x ./kubectl-argo-rollouts-linux-amd64
sudo mv ./kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts

curl -LO https://github.com/sharkdp/bat/releases/download/v0.18.0/bat-v0.18.0-x86_64-unknown-linux-gnu.tar.gz
tar -xzf bat-v0.18.0-x86_64-unknown-linux-gnu.tar.gz
chmod +x ./bat-v0.18.0-x86_64-unknown-linux-gnu/bat
sudo mv ./bat-v0.18.0-x86_64-unknown-linux-gnu/bat /usr/local/bin/bat


# Start k8s
launch.sh

cd /root
# TODO(bwplotka): Move back to git clone https://github.com/AnaisUrlichs/observe-argo-rollout.git /root/demo
git clone https://github.com/bwplotka/observe-argo-rollout.git /root/demo
cd /root/demo

bash demo.sh
