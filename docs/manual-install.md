# Manual install

If you would rather not use [the Helm chart](helm-chart.md), kube-monkey is a single
deployment plus a configmap.

Example manifests for everything below live in
[`examples/`](https://github.com/asobti/kube-monkey/tree/master/examples).

## 1. Create the configmap

Deploy the `kube-monkey-config-map` configmap in the namespace you intend to run kube-monkey
in, for example `kube-system`. The key name has to be `config.toml`.

```bash
kubectl create configmap km-config --from-file=config.toml=km-config.toml
```

or

```bash
kubectl apply -f km-config.yaml
```

The configmap has to exist before the kube-monkey deployment starts.

## 2. Deploy kube-monkey

Run kube-monkey as an app inside the cluster, in a namespace with permission to kill pods in
other namespaces, such as `kube-system`.

```bash
kubectl apply -f examples/deployment.yaml
```

!!! warning "It needs cluster-wide list permission"

    kube-monkey lists deployments, statefulsets and daemonsets across the whole cluster,
    because the namespace lists hold patterns. A Role scoped to a single namespace is not
    enough: kube-monkey will log an error and schedule nothing. See
    [How it works](how-it-works.md#how-it-sees-the-cluster).

## 3. Check the logs

```bash
kubectl logs -f deployment.apps/kube-monkey --namespace=kube-system
```

Here `deployment.apps/kube-monkey` is the deployment for kube-monkey itself.

## Building the image yourself

```bash
git clone https://github.com/asobti/kube-monkey.git
cd kube-monkey
make build
make container
```

Official images are on
[Docker Hub](https://hub.docker.com/r/ayushsobti/kube-monkey/tags/).
