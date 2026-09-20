# Getting started

This page takes you from nothing to watching kube-monkey delete a pod on purpose.

## Kubernetes compatibility

kube-monkey talks to the cluster through
[client-go](https://github.com/kubernetes/client-go). Each release is built against one
version of it, pinned in `go.mod`. The client-go
[compatibility matrix](https://github.com/kubernetes/client-go#compatibility-matrix) is the
formal answer to which Kubernetes versions that covers: the matching minor version, give or
take one.

The real range is much wider, because kube-monkey only uses APIs that went stable years ago.

| API | Used for | Stable since |
| --- | --- | --- |
| `apps/v1` | Finding the Deployments, StatefulSets and DaemonSets that opted in | 1.9 |
| `core/v1` | Listing and deleting pods | 1.0 |
| `rbac.authorization.k8s.io/v1` | The ClusterRole and binding the chart installs | 1.8 |

There are no custom resources and no beta APIs, so 1.9 is the floor, and the Helm chart
refuses to install below it. Anything between that floor and the version kube-monkey is built
against is untested rather than unsupported. If an old cluster does give you trouble you will
see it as a failure to list or delete pods in the log, so
[open an issue](https://github.com/asobti/kube-monkey/issues) with the version and the error.

## 1. Install kube-monkey

kube-monkey runs as a normal workload inside the cluster. It needs to live in a namespace
allowed to delete pods in other namespaces, so `kube-system` is the usual home.

```bash
helm repo add kubemonkey https://asobti.github.io/kube-monkey/charts/repo
helm repo update
helm install kube-monkey kubemonkey/kube-monkey --namespace kube-system
```

The defaults are deliberately harmless. `dryRun` is on, so kube-monkey builds a schedule
and logs what it would have killed without deleting anything.

If you would rather write the manifests yourself, see [Manual install](manual-install.md).

## 2. Opt an app in

kube-monkey ignores everything that has not asked to be included. Add these labels to a
Deployment, StatefulSet or DaemonSet:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: monkey-victim
  namespace: app-namespace
  labels:
    kube-monkey/enabled: enabled
    kube-monkey/identifier: monkey-victim
    kube-monkey/mtbf: "2"
```

That reads as: this app is in, call it `monkey-victim`, and expect to lose a pod on about
one run day in two. Run days are weekdays unless you change `run_days`.

The labels go on the app's own `metadata.labels`. See [Opting in to chaos](opting-in.md)
for the full set, including the kill modes that let you take out more than one pod.

## 3. Let it through the namespace lists

By default the blacklist holds `kube-system` and the whitelist is empty, which means every
namespace. If you have narrowed the whitelist, your app's namespace has to be on it.

```toml
[kubemonkey]
whitelisted_namespaces = ["app-namespace"]
```

See [Configuration](configuration.md#namespace-lists) for how the two lists interact.

## 4. Watch it work

Terminations normally happen at a random moment inside the daily window, which is not much
use when you are trying to confirm the install. Debug mode ignores the window and kills on a
short repeating cycle:

```bash
helm upgrade kube-monkey kubemonkey/kube-monkey --namespace kube-system \
  --set config.debug.enabled=true \
  --set config.debug.schedule_immediate_kill=true
```

Each round waits `config.debug.schedule_delay` seconds, builds a schedule, then kills each
victim at a random point in the next 60 seconds. Raise the delay if you want longer between
rounds, for example to give pods time to come back:

```bash
helm upgrade kube-monkey kubemonkey/kube-monkey --namespace kube-system \
  --set config.debug.schedule_delay=300
```

Follow the logs:

```bash
kubectl logs -f deployment.apps/kube-monkey --namespace kube-system
```

With `dryRun` still on you will see the terminations it intends to carry out. Turn the
handle when you believe it:

```bash
helm upgrade kube-monkey kubemonkey/kube-monkey --namespace kube-system \
  --set config.dryRun=false
```

!!! warning "Turn debug back off"

    Debug mode attacks in a loop and ignores `start_hour` and `end_hour`. It is a tool for
    the first ten minutes, not a setting to leave on.

## 5. Move to a real schedule

Once you trust it, drop debug mode and pick the hours that suit the people on call:

```toml
[kubemonkey]
dry_run = false
run_days = ["mon", "tue", "wed", "thu", "fri"]  # Weekdays only
run_hour = 8      # Build the day's schedule at 8am. Nothing dies yet
start_hour = 10   # No terminations before 10am
end_hour = 16     # No terminations after 4pm
time_zone = "Europe/Lisbon"
```

## Where to go next

- [How it works](how-it-works.md) for what happens at scheduling time and termination time
- [Configuration](configuration.md) for every setting and its default
- [Metrics](metrics.md) to chart the terminations
- [Notifications](notifications.md) to post each attack to Slack or your own API

## Docker images

Images are published to
[Docker Hub](https://hub.docker.com/r/ayushsobti/kube-monkey/tags/).
