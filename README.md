# observe-argo-rollout

Demo for _Automating and Monitoring Kubernetes Rollouts with Argo and Prometheus_

## Performing Demo 

The demo can be found on [Katacoda](https://katacoda.com/anaisurlichs/courses/demos/promcon2021).

Alternatively, you can follow the instructios below.

1. Deploy Kubernetes (tested with 1.18)

1. Install argo [rollout kubectl plugin](https://argoproj.github.io/argo-rollouts/installation/#kubectl-plugin-installation)

```
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x ./kubectl-argo-rollouts-linux-amd64
sudo mv ./kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts
```

1. Install bat:

```
curl -LO https://github.com/sharkdp/bat/releases/download/v0.18.0/bat-v0.18.0-x86_64-unknown-linux-gnu.tar.gz
tar -xzf bat-v0.18.0-x86_64-unknown-linux-gnu.tar.gz
chmod +x ./bat-v0.18.0-x86_64-unknown-linux-gnu/bat
sudo mv ./bat-v0.18.0-x86_64-unknown-linux-gnu/bat /usr/local/bin/bat
```

1. Run bash ./demo.sh
1. Run commands one by one or the one you want using keyboard keys:
 * `enter`: execute command, `enter` again to reveal another command.
 * `q`: quit  
 * `p`: previous command
 * `n`: next command
 * `b`: start from beginning 
 * `n`: start from end 

 The structure of the demo can be viewed [here](https://github.com/AnaisUrlichs/observe-argo-rollout/blob/main/demo-arch-all.txt).

## Grafana Dashboard

Once the first Argo rollout is deployed, you can view the metrics of the client and ping-pong app in the Grafana Dashboard. 
Katacoda provides you with a link to Prometheus and Grafana. Simply navigate to Dashboards < Manage < Demo and make sure that the pinger is deployed and running to see metrics.

![Grafana Gif](grafana.gif)


