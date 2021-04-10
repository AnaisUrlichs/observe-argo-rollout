# observe-argo-rollout

Demo for _Automating and Monitoring Kubernetes Rollouts with Argo and Prometheus_

## Performing Demo 

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


## Draft

Install Argo Rollouts
```
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://raw.githubusercontent.com/argoproj/argo-rollouts/stable/manifests/install.yaml
```

Install Argo CLI
```
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x ./kubectl-argo-rollouts-linux-amd64
sudo mv ./kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts
```

test Argo Rollouts version
```
kubectl argo rollouts version
```

Install Prometheus and Grafana

```
kubectl apply -n demo -f manifests/generated/monitoring
```

Open Prometheus dashboard
```
 kubectl port-forward -n demo service/prom 9090
```

Open prometheus dashboard
```
 kubectl port-forward -n demo service/grafana 3000
```

Applying Rollout Service, Rollout, and Analysis Template
```
kubectl apply -n demo -f manifests/application/services.yaml
kubectl apply -n demo -f manifests/application/application-rollout.yaml
kubectl apply -n demo -f manifests/application/analysis-template.yaml
```

Now have a look at the rollout 

```
kubectl argo rollouts -n demo get rollout rollouts-demo
```

Now deploy a new app version that will trigger the canary deployment
```
kubectl argo rollouts -n demo set image rollouts-demo rollouts-demo=anaisurlichs/ping-pong:3.0
```