# Kube-Monkey Helm Chart

[Kube-Monkey](https://github.com/asobti/kube-monkey) periodically kills pods in your Kubernetes cluster,that are opt-in based on their own rules.

## TL;DR;

TBA to official helm charts

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ cd kube-monkey/helm
$ helm install --name my-release kubemonkey
```

The command deploys kube-monkey on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the my-release deployment:

```console
$ helm delete my-release --purge
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

By default `Kube-Monkey` runs in dry-run mode so it doesn't actually kill anything.
If you're sure you want to use it run `helm` with:

```console
$ helm install stable/chaoskube --set dryRun=false
```

| Parameter                 | Description                                         | Default                          |
|---------------------------|-----------------------------------------------------|----------------------------------|
| `image.repository`        | docker image repo                                   | ayushsobti/kube-monkey           |
| `image.tag`               | docker image tag                                    | v0.3.0                           |
| `replicaCount`            | number of replicas to run                           | 1                                |
| `rbac.enabled`            | rbac enabled or not                                 | true                             |
| `image.tag.IfNotPresent`  | image pull logic                                    | IfNotPresent                     |
| `namespace`               | namespace that kube-monkey will be installed        | default                          |
| `config.dryRun`           | Will not kill pods only logs                        | false                            |
| `config.runHour`          | Schedule start time in 24hr format                  | 8                                |
| `config.startHour`        | Pod killing start time  in 24hr format              | 10                               |
| `config.endHour`          | Pod killing stop time  in 24hr format               | 16                               |
| `config.endHour`      | Set Time Range to avoid actions (Optional)          | "" (Don't skip any times of day) |
| `config.endHour.blacklistedNamespaces`      | Set Days of the Year to avoid actions (Optional)    | "" (Don't skip any days)         |
| `priorityClassName`       | priorityClassName                                   | `nil`                            |
| `rbac.create`             | create rbac service account and roles               | false                            |
| `rbac.serviceAccountName` | name of serviceAccount to use when create is false  | default                          |
| `resources`               | CPU/Memory resource requests/limits                 | `{}`                             |
| `nodeSelector`            | Node labels for pod assignment                      | `{}`                             |
| `tolerations`             | Toleration labels for pod assignment                | `[]`                             |
| `affinity`                | Affinity settings for pod assignment                | `{}`                             |
| `minimumAge`              | Set minimum pod age to filter pod by                | `0s`                             |

Setting label and namespaces selectors from the shell can be tricky but is possible (example with zsh):

```console
$ helm install \
  --set labels='app=mate\,stage!=prod',namespaces='!kube-system\,!production' \
  stable/chaoskube --debug --dry-run | grep -A7 args
    args:
    - --interval=10m
    - --labels=app=foo,stage!=prod
    - --namespaces=!kube-system,!production
    - --timezone=America/New_York
    - --excludedWeekdays="Sat,Tue"
    - --excludedTimesOfDay="12:00-18:00"
    - --excludedDaysOfYear="Apr1,Dec24"
```
