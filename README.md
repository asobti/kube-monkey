[![Build](https://github.com/asobti/kube-monkey/actions/workflows/go.yml/badge.svg)](https://github.com/asobti/kube-monkey/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/asobti/kube-monkey)](https://goreportcard.com/report/github.com/asobti/kube-monkey)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docker Pulls](https://img.shields.io/docker/pulls/ayushsobti/kube-monkey?label=Docker%20pulls&logo=docker)](https://hub.docker.com/r/ayushsobti/kube-monkey/)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubemonkey)](https://artifacthub.io/packages/search?repo=kubemonkey)

kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey) for [Kubernetes](http://kubernetes.io/) clusters. It randomly deletes Kubernetes (k8s) pods in the cluster encouraging and validating the development of failure-resilient services.

Join us at [#kube-monkey](https://kubernetes.slack.com/messages/kube-monkey) on Kubernetes Slack.

---

kube-monkey runs at a pre-configured hour (`run_hour`, defaults to 8 am) on weekdays, and builds a schedule of deployments that will face a random
Pod death sometime during the same day. The time-range during the day when the random pod Death might occur is configurable and defaults to 10 am to 4 pm.

kube-monkey can be configured with two lists of namespaces
* a whitelist (only apps in a whitelisted namespace can be touched)
* a blacklist (apps in a blacklisted namespace are never touched)

An app has to pass both lists. Where the two overlap the blacklist wins.

Entries in either list are shell-style patterns, so a namespace is matched when it matches any entry:

| Entry | Matches |
| --- | --- |
| `kube-system` | `kube-system` only |
| `team-?` | `team-a`, `team-b`, but not `team-ab` |
| `*-prod` | `shop-prod`, `checkout-prod` |
| `*` | every namespace |

Namespace names never contain `*` or `?`, so a plain name still matches nothing but itself.

Patterns are checked when the config loads, and a malformed one stops kube-monkey from starting.

To disable the blacklist provide `[""]` in the `blacklisted_namespaces` config.param. The whitelist is off by default, which is `[""]`
in `whitelisted_namespaces` and means every namespace.

## Opting-In to Chaos

kube-monkey works on an opt-in model and will only schedule terminations for Kubernetes (k8s) apps that have explicitly agreed to have their pods terminated by kube-monkey.

Opt-in is done by setting the following labels on a k8s app:

**`kube-monkey/enabled`**: Set to **`"enabled"`** to opt-in to kube-monkey  
**`kube-monkey/mtbf`**: Mean time between failure, as a whole number and a unit: `d` for days, `h` for hours or `m` for minutes. For example,
if set to **`"3d"`**, the k8s app can expect to have a Pod killed approximately every third weekday, and if set to **`"2h"`**, it can expect
to lose a Pod every two hours. A value without a unit is read as days, so **`"3"`** and **`"3d"`** mean the same thing. The shortest mean time
between failure is one minute. Note that all terminations happen inside the daily run window (see `start_hour` and `end_hour`), so an mtbf
shorter than a day packs that day's terminations into that window.  
**`kube-monkey/identifier`**: A unique identifier for the k8s app. The recommendation is to set this value to be the same as the app's name.
It also gives you a way to pick the victim's pods by hand: if you repeat this label on the pod template, kube-monkey looks for pods carrying
`kube-monkey/identifier: foo` instead of using the app's own pod selector. Two apps sharing an identifier are then treated as one pool of pods.  
**`kube-monkey/kill-mode`**: Default behavior is for kube-monkey to kill only ONE pod of your app. You can override this behavior by setting the value to:
* `kill-all` if you want kube-monkey to kill **ALL** of your pods regardless of status (including not ready and not running pods). Does not require `kill-value`. **Use this label carefully.**
* `fixed` if you want to kill a specific number of running pods with `kill-value`. If you overspecify, it will kill **all** running pods and issue a warning.
* `random-max-percent` to specify a *maximum* `%` with `kill-value` that can be killed. At the scheduled time, a uniform *random specified* `%` of the running pods will be terminated.
* `fixed-percent` to specify a *fixed* `%` with `kill-value` that can be killed. At the scheduled time, a specified *fixed* `%` of the running pods will be terminated.


**`kube-monkey/kill-value`**: Specify value for kill-mode
* if `fixed`, provide an integer of pods to kill
* if `random-max-percent`, provide a number from `0`-`100` to specify the max `%` of pods kube-monkey can kill
* if `fixed-percent`, provide a number from `0`-`100` to specify the `%` of pods to kill

#### Where to put the labels

All of these labels go on the k8s app's own **`metadata.labels`**. That is the only place kube-monkey reads them from, both when it
builds the daily schedule and when it re-checks the app at termination time.

You do not need to repeat them on **`spec.template.metadata.labels`**. To find the pods to kill, kube-monkey falls back to the app's own
pod selector (`spec.selector`), which every Deployment, StatefulSet and DaemonSet already has.

#### Example of opted-in Deployment killing one pod per purge

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: monkey-victim
  namespace: app-namespace
  labels:
    kube-monkey/enabled: enabled
    kube-monkey/identifier: monkey-victim
    kube-monkey/mtbf: '2'
    kube-monkey/kill-mode: "fixed"
    kube-monkey/kill-value: '1'
spec:
  selector:
    matchLabels:
      app: monkey-victim
  template:
    metadata:
      labels:
        app: monkey-victim
[... omitted ...]
```

### Overriding the apiserver
#### Use cases:
* Since client-go does not support [cluster dns](https://github.com/kubernetes/client-go/blob/master/rest/config.go#L331) explicitly with a `// TODO: switch to using cluster DNS.` note in the code, you may need to override the apiserver.
* If you are running an unauthenticated system, you may need to force the http apiserver endpoint.

#### To override the apiserver specify in the config.toml file
```toml
[kubernetes]
host="https://your-apiserver-url.com:apiport"
```

## How kube-monkey works

#### Scheduling time
Scheduling happens once a day on Weekdays - this is when a schedule for terminations for the current day is generated. During scheduling, kube-monkey will:  
1. Generate a list of eligible k8s apps (k8s apps that have opted-in and are not blacklisted, if specified, and are whitelisted, if specified)
2. For each eligible k8s app, work out how many pods to kill today from `kube-monkey/mtbf`. An app is killed 24h/mtbf times a day, so an mtbf
   of a day or more gives at most one termination and a shorter one gives several
3. For each termination, calculate a random time when a pod will be killed

#### Termination time
This is the randomly generated time during the day when a victim k8s app will have a pod killed.
At termination time, kube-monkey will:
1. Check if the k8s app is still eligible (has not opted-out or been blacklisted or removed from the whitelist since scheduling)
2. Check if the k8s app has updated kill-mode and kill-value
3. Depending on kill-mode and kill-value, execute pods

## Docker Images

Docker images for kube-monkey can be found at [DockerHub](https://hub.docker.com/r/ayushsobti/kube-monkey/tags/)

## Building

Clone the repository and build the container.

```bash
go get github.com/asobti/kube-monkey
cd $GOPATH/src/github.com/asobti/kube-monkey
make build
make container
```

## Configuring
kube-monkey is configured by environment variables or a toml file placed at `/etc/kube-monkey/config.toml` and expects the configmap to exist before the kube-monkey deployment.

Configuration keys and descriptions can be found in [`config/param/param.go`](https://github.com/asobti/kube-monkey/blob/master/internal/pkg/config/param/param.go)

#### Example config.toml file
```toml
[kubemonkey]
dry_run = true                           # Terminations are only logged
run_hour = 8                             # Run scheduling at 8am on weekdays
start_hour = 10                          # Don't schedule any pod deaths before 10am
end_hour = 16                            # Don't schedule any pod deaths after 4pm
blacklisted_namespaces = ["kube-system"] # Critical apps live here. Patterns like "*-prod" work too
whitelisted_namespaces = [""]            # Every namespace. Narrow it with names or patterns like "team-*"
time_zone = "America/New_York"           # Set tzdata timezone example. Note the field is time_zone not timezone
```

#### Example environment variables
```
KUBEMONKEY_DRY_RUN=true
KUBEMONKEY_RUN_HOUR=8
KUBEMONKEY_START_HOUR=10
KUBEMONKEY_END_HOUR=16
KUBEMONKEY_BLACKLISTED_NAMESPACES=kube-system
KUBEMONKEY_WHITELISTED_NAMESPACES=
KUBEMONKEY_TIME_ZONE=America/New_York
```
#### Example Config to test kube-monkey works by enabling debug mode

Note: this will keep attacking pods every 60s regardless of what you configured for the `startHour` and `endHour`.

```toml
[debug]
enabled= true
schedule_immediate_kill= true
```

## Notifications

Kube-monkey supports notifications and can notify an endpoint of your choice after an attack.
It can be a Slack webhook or a custom API.

#### Example Config for posting attack notifications to an HTTP endpoint
```toml
[notifications]
  enabled = true
  reportSchedule = true
  [notifications.attacks]
    endpoint = "http://url1"
    message = "message1"
    headers = ["header1Key:header1Value","header2Key:header2/Value"]
```

#### Placeholders

The message supports the following placeholders:
* `{$name}`: victim's name
* `{$kind}`: victim's kind
* `{$namespace}`: victim's namespace
* `{$timestamp}`: attack's time from Unix epoch in milliseconds
* `{$time}`: attack's time
* `{$date}`: attack's date
* `{$error}`: result's error, if any
* `{$kubemonkeyid}`: kube-monkey id (set using KUBE_MONKEY_ID env variable otherwise empty)

```
  message: '{
            "what": "Kube-monkey(${kubemonkeyid}) attack of {$name} in {$namespace}",
            "who": "{$name}",
            "when": {$timestamp}
           }'
```

The header supports a special placeholder to retrieve the value of an environment variable.
This is useful when calling an API that has a protected endpoint.
A typical scenario will be to pass an API token to the Kube-monkey container, this token is stored in a Kubernetes Secret and you want to pass it via an environment variable.

```
headers = ["api-key:{$env:API_TOKEN}", "Content-Type:application/json"]
```

`{$env:API_TOKEN}` will be replaced by the environment variable `API_TOKEN` value.

Note if the environment variable does not exist, the notification call will NOT be cancelled. The value will resolve to an empty string, and a warning will show up in the logs. 

## Metrics

kube-monkey can serve Prometheus metrics about the schedules it builds and the terminations it runs.
The endpoint is off by default. When it is on, kube-monkey listens on `/metrics` at the configured address.

#### Example config for the metrics endpoint
```toml
[metrics]
  enabled = true
  address = ":8080"  # host:port to listen on, leave the host out to listen on every interface
```

The same settings as environment variables:
```
METRICS_ENABLED=true
METRICS_ADDRESS=:8080
```

#### Exposed metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `kube_monkey_schedule_size` | gauge | | Terminations on the latest schedule |
| `kube_monkey_last_schedule_timestamp_seconds` | gauge | | When the latest schedule was generated |
| `kube_monkey_scheduled_terminations_total` | counter | `kind`, `namespace`, `name` | Terminations kube-monkey has put on a schedule |
| `kube_monkey_terminations_total` | counter | `kind`, `namespace`, `name`, `result` | Terminations kube-monkey has carried out. `result` is `success` or `failure`, where a failure means the whole termination was skipped or errored, for example because the victim opted out after it was scheduled |
| `kube_monkey_pods_terminated_total` | counter | `kind`, `namespace`, `name` | Pods kube-monkey has deleted. Stays at zero in dry run mode because no pod is really deleted |

The standard Go runtime and process metrics are exposed as well.

#### Scraping

To let a Prometheus that discovers pods scrape kube-monkey, annotate the pod template in the
[example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml):

```yaml
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
```

The Helm chart creates a Service for the endpoint when `config.metrics.enabled` is set, and a ServiceMonitor
for the Prometheus Operator when `serviceMonitor.enabled` is set too.

## Deploying

**Manually**
1. First, deploy the expected `kube-monkey-config-map` configmap in the namespace you intend to run kube-monkey in (for example, the `kube-system` namespace). Make sure to define the keyname as `config.toml`

> For example `kubectl create configmap km-config --from-file=config.toml=km-config.toml` or `kubectl apply -f km-config.yaml`

2. Run kube-monkey as a k8s app within the Kubernetes cluster, in a namespace that has permissions to kill Pods in other namespaces (eg. `kube-system`).

See dir [`examples/`](https://github.com/asobti/kube-monkey/tree/master/examples) for example Kubernetes yaml files.

3. You should be able to see debug logs by `kubectl logs -f deployment.apps/kube-monkey --namespace=kube-system`  here the `deployment.apps/kube-monkey` is the k8s deployment for kube-monkey.


**Helm Chart**  

See [How to install kube-monkey with Helm](helm/kubemonkey/README.md).

## Logging

kube-monkey uses [glog](github.com/golang/glog) and supports all command-line features for glog. It logs to stderr, so `kubectl logs` shows everything. To change the v level, see `args: ["-v=5"]` in the [example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml)

To also write glog log files, pass `-log_dir=/path/to/custom/log`. The image runs on an empty filesystem, so mount a writable volume at that path, otherwise the directory cannot be created and kube-monkey exits.

> **Standardized glog levels `grep -r V\([0-9]\) *`**
>
> L0: None
>
> L1: Highest Level current status info and Errors with Terminations
>
> L2: Successful terminations
>
> L3: More detailed schedule status info
>
> L4: Debugging verbose schedule and config info
>
> L5: Auto-resolved inconsequential issues

More resources: See the [k8s logging page](https://kubernetes.io/docs/concepts/cluster-administration/logging/) suggesting [community conventions for logging severity](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md)

## Instructions on how to get this working on OpenShift 3.x

```
git clone https://github.com/asobti/kube-monkey.git
cd examples
oc login http://someserver/ -u system:admin
oc project kube-system
oc create -f configmap.yaml
oc -n kube-system adm policy add-role-to-user -z deployer system:deployer
oc -n kube-system adm policy add-role-to-user -z builder system:image-builder
oc -n kube-system adm policy add-role-to-group system:image-puller system:serviceaccounts:kube-system
oc run kube-monkey --image=docker.io/ayushsobti/kube-monkey:v0.4.0 --command -- /kube-monkey -v=5 -logtostderr=true
oc volume dc/kube-monkey --add --name=kubeconfigmap -m /etc/kube-monkey -t configmap --configmap-name=kube-monkey-config-map
```

### OpenShift 4.x

```
git clone https://github.com/asobti/kube-monkey.git
cd examples
oc login http://someserver/ -u system:admin
oc project kube-system
oc create -f configmap.yaml
oc -n kube-system adm policy add-cluster-role-to-user edit -z default --rolebinding-name kube-monkey-edit
oc run kube-monkey --image=docker.io/ayushsobti/kube-monkey:v0.3.0 --command -- /kube-monkey -v=5 -logtostderr=true
oc set volume dc/kube-monkey --add --name=kubeconfigmap -m /etc/kube-monkey -t configmap --configmap-name=kube-monkey-config-map
```

## Ways to contribute

See [How to Contribute](CONTRIBUTING.md)

## License
This project is licensed under the Apache License v2.0 - see the [LICENSE](LICENSE) file for details.
