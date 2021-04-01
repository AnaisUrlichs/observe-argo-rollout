# observe-argo-rollout

Demo for _Automating and Monitoring Kubernetes Rollouts with Argo and Prometheus_

Install the Prometheus Operator

```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

helm repo update

helm install prom prometheus-community/kube-prometheus-stack
```

Install Argo Rollouts to your local cluster

```
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://raw.githubusercontent.com/argoproj/argo-rollouts/stable/manifests/install.yaml
```

Installing the different resources to configure the application — within manifests/application

```
cd manifests/application

kubectl apply -f analysis-template.yaml
kubectl apply -f services.yaml
kubectl apply -f application-rollout.yaml
```

Note that you have to apply the application rollout after you apply the services; otherwise, it will go look for the services and not find it ⇒ the rollout will crash.

Make sure the rollout has happened correctly

```
kubectl argo rollouts get rollout rollouts-demo
```

Access the application

```
kubectl port-forward
```

Update the image

```
kubectl argo rollouts set image rollouts-demo \
  rollouts-demo=argoproj/rollouts-demo:yellow
```

Access the Prometheus port

```
kubectl port-forward service/prom-kube-prometheus-stack-prometheus 9090
```

Access the App

```
kubectl port-forward service/rollouts-demo-stable 8080:80
```