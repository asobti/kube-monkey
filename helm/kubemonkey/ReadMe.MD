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
If you're confident you want to use it in real run `helm` with:

```console
$ helm install --name my-release kubemonkey --set config.dryRun=false
```

By default `Kube-Monkey` runs in without any white listed namespace mode so it doesn't actually kill anything.
If you're confident you want to enable it in real, run `helm` with:

```console
$ helm install --name my-release kubemonkey --set config.dryRun=false --set config.whitelistedNamespaces=namespace1,namespace2,namespace3
```

| Parameter                 | Description                                         | Default                          |
|---------------------------|-----------------------------------------------------|----------------------------------|
| `image.repository`        | docker image repo                                   | ayushsobti/kube-monkey           |
| `image.tag`               | docker image tag                                    | v0.3.0                           |
| `replicaCount`            | number of replicas to run                           | 1                                |
| `rbac.enabled`            | rbac enabled or not                                 | true                             |
| `image.tag.IfNotPresent`  | image pull logic                                    | IfNotPresent                     |
| `namespace`               | namespace that kube-monkey will be installed        | default                          |
| `config.dryRun`           | will not kill pods only logs                        | false                            |
| `config.runHour`          | schedule start time in 24hr format                  | 8                                |
| `config.startHour`        | pod killing start time  in 24hr format              | 10                               |
| `config.endHour`          | pod killing stop time  in 24hr format               | 16                               |
| `config.whitelistedNamespaces`| pods in this namespace that opt-in will be killed|                                 |
| `config.endHour.blacklistedNamespaces`| pods in this namespace will not be killed| kube-system                     |
| `config.timeZone`         | time zone in DZ format                              | `America/New_York`               |
| `config.debug.enabled`    | debug mode,need to be enabled to see debuging behaviour| false                         |
| `config.debug.schedule_immediate_kill` | immediate pod kill matching other rules apart from time| true             |
| `args.logLevel`           | go log level                                        | 5                                |
| `args.logDir`             | log directory                                       | /var/log/kube-monkey             |

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
