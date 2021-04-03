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

1. Run bash ./demo.sh
1. Run commands one by one or the one you want using keyboard keys:
 * `enter`: execute command, `enter` again to reveal another command.
 * `q`: quit  
 * `p`: previous command
 * `n`: next command
 * `b`: start from beginning 
 * `n`: start from end 


## Draft

test argo rollout version

```
kubectl argo rollouts version
```

have istioctl installed and install istio

```
istioctl install --set profile=demo
```

```
kubectl get all -n istio-system
```

Install prometheus add on

```
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.9/samples/addons/prometheus.yaml
```

Open prometheus dashboard

```
istioctl dashboard prometheus
```

This will not show anything since nothing is installed yet.

Installing the different resources to configure the application — within manifests/application

```
cd manifests/application
kubectl apply -f ./
```

Now have a look at the rollout 

```
kubectl argo rollouts get rollout rollouts-demo
```

Now deploy a new app version that will trigger the canary deployment
```
kubectl argo rollouts set image rollouts-demo rollouts-demo=argoproj/rollouts-demo:yellow
```