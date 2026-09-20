# Helm chart

The chart is published from this repository to a Helm repo served next to this site.

## Add the repository

```bash
helm repo add kubemonkey https://asobti.github.io/kube-monkey/charts/repo
helm repo update
```

## Install

```bash
helm install my-release kubemonkey/kube-monkey
```

This deploys kube-monkey with the default configuration, which is dry run mode with no
whitelisted namespaces. It kills nothing until you tell it to.

Pin the chart version if you want a repeatable install:

```bash
helm install my-release kubemonkey/kube-monkey --version 1.9.0
```

## Uninstall

```bash
helm delete my-release
```

This removes every Kubernetes component associated with the chart and deletes the release.

## Common changes

By default kube-monkey runs in dry run mode, so it does not actually kill anything. When you
are confident, turn it off:

```bash
helm install my-release kubemonkey/kube-monkey --set config.dryRun=false
```

By default there are no whitelisted namespaces. Name the ones you want in scope:

```bash
helm install my-release kubemonkey/kube-monkey \
  --set config.dryRun=false \
  --set config.whitelistedNamespaces="{namespace1,namespace2,namespace3}"
```

To see kube-monkey kill pods immediately, in debug mode:

```bash
helm install my-release kubemonkey/kube-monkey \
  --set config.dryRun=false \
  --set config.whitelistedNamespaces="{namespace1,namespace2,namespace3}" \
  --set config.debug.enabled=true \
  --set config.debug.schedule_immediate_kill=true
```

To change when kube-monkey wakes up, and when it starts and stops killing pods:

```bash
helm install my-release kubemonkey/kube-monkey \
  --set config.dryRun=false \
  --set config.whitelistedNamespaces="{namespace1,namespace2,namespace3}" \
  --set config.runDays="{mon,wed,fri}" \
  --set config.runHour=10 \
  --set config.startHour=11 \
  --set config.endHour=17
```

To check the values that actually reached the configmap:

```bash
helm get manifest my-release
```

## Values

| Parameter | Description | Default |
|---|---|---|
| `image.repository` | docker image repo | `ayushsobti/kube-monkey` |
| `image.tag` | docker image tag | `v0.6.0` |
| `image.pullPolicy` | image pull logic | `IfNotPresent` |
| `replicaCount` | number of replicas to run | `1` |
| `config.dryRun` | will not kill pods, only logs behaviour | `true` |
| `config.runDays` | days of the week the schedule is built on | `[mon, tue, wed, thu, fri]` |
| `config.runHour` | schedule start time in 24hr format | `8` |
| `config.startHour` | pod killing start time in 24hr format | `10` |
| `config.endHour` | pod killing stop time in 24hr format | `16` |
| `config.whitelistedNamespaces` | pods in this namespace that opt in will be killed | |
| `config.blacklistedNamespaces` | pods in this namespace will not be killed | `kube-system` |
| `config.timeZone` | time zone in tzdata format | `America/New_York` |
| `config.debug.enabled` | debug mode, needed to see debugging behaviour | `false` |
| `config.debug.schedule_immediate_kill` | immediate pod kill matching other rules apart from time | `false` |
| `config.notifications.enabled` | enables reporting of attacks to an HTTP endpoint | `false` |
| `config.notifications.proxy` | notifications proxy URL | |
| `config.notifications.attacks` | HTTP collector as (endpoint,message,headers) where attacks are reported | |
| `config.metrics.enabled` | serves Prometheus metrics on /metrics and creates a Service for them | `false` |
| `config.metrics.port` | port the metrics endpoint listens on | `8080` |
| `serviceMonitor.enabled` | creates a ServiceMonitor for the Prometheus Operator, needs `config.metrics.enabled` | `false` |
| `serviceMonitor.interval` | how often Prometheus scrapes the metrics | `30s` |
| `serviceMonitor.scrapeTimeout` | how long Prometheus waits for a scrape | `10s` |
| `serviceMonitor.additionalLabels` | extra labels on the ServiceMonitor, to match your Prometheus selector | `{}` |
| `args.logLevel` | go log level | `5` |
| `args.logDir` | writes log files to this directory, empty means stderr only | |
| `args.extraArgs` | extra command line flags for kube-monkey | `[]` |
| `additionalVolumes` | extra volumes on the pod | `[]` |
| `additionalVolumeMounts` | extra volume mounts on the container | `[]` |
| `podSecurityContext` | security context for the pod | `{}` |

These map onto the settings described in the
[configuration reference](configuration.md), which is the place to look for what each one
actually does.

## Using a values file

Rather than a long line of `--set` flags, edit `values.yaml` and install from it:

```yaml
replicaCount: 1
image:
  repository: ayushsobti/kube-monkey
  tag: v0.6.0
  pullPolicy: IfNotPresent
config:
  dryRun: false
  runDays: [ "mon", "tue", "wed", "thu", "fri" ]
  runHour: 8
  startHour: 10
  endHour: 16
  blacklistedNamespaces: [ "kube-system" ]
  whitelistedNamespaces: [ "namespace1", "namespace2" ]
  timeZone: America/New_York
args:
  logLevel: 5
```

```bash
helm install my-release kubemonkey/kube-monkey --namespace kube-monkey -f values.yaml
```

## Running as a non-root user

Nothing beyond `podSecurityContext` is needed while logs go to stderr:

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1001
  runAsGroup: 1001
  fsGroup: 1001
```

If you also set `args.logDir`, add a writable volume at that path as shown in
[Logging](logging.md). Without it the user cannot create the directory and kube-monkey will
not start.
