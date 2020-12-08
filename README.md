# kube-monkey [![Build Status](https://travis-ci.org/asobti/kube-monkey.svg?branch=master)](https://travis-ci.org/asobti/kube-monkey) [![Go Report](https://goreportcard.com/badge/github.com/asobti/kube-monkey)](https://goreportcard.com/report/github.com/asobti/kube-monkey)

kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey) for [Kubernetes](http://kubernetes.io/) clusters. It randomly deletes Kubernetes (k8s) pods in the cluster encouraging and validating the development of failure-resilient services.

Join us at [#kube-monkey](https://kubernetes.slack.com/messages/kube-monkey) on Kubernetes Slack.

---

kube-monkey runs at a pre-configured hour (`run_hour`, defaults to 8 am) on weekdays, and builds a schedule of deployments that will face a random
Pod death sometime during the same day. The time-range during the day when the random pod Death might occur is configurable and defaults to 10 am to 4 pm.

kube-monkey can be configured with a list of namespaces
* to blacklist (any deployments within a blacklisted namespace will not be touched)

To disable the blacklist provide `[""]` in the `blacklisted_namespaces` config.param.

## Opting-In to Chaos

kube-monkey works on an opt-in model and will only schedule terminations for Kubernetes (k8s) apps that have explicitly agreed to have their pods terminated by kube-monkey.

Opt-in is done by setting the following labels on a k8s app:

**`kube-monkey/enabled`**: Set to **`"enabled"`** to opt-in to kube-monkey  
**`kube-monkey/mtbf`**: Mean time between failure (in days). For example, if set to **`"3"`**, the k8s app can expect to have a Pod
killed approximately every third weekday.  
**`kube-monkey/identifier`**: A unique identifier for the k8s apps. This is used to identify the pods
that belong to a k8s app as Pods inherit labels from their k8s app. So, if kube-monkey detects that app `foo` has enrolled to be a victim, kube-monkey will look for all pods that have the label `kube-monkey/identifier: foo` to determine which pods are candidates for killing. The recommendation is to set this value to be the same as the app's name.  
**`kube-monkey/kill-mode`**: Default behavior is for kube-monkey to kill only ONE pod of your app. You can override this behavior by setting the value to:
* `kill-all` if you want kube-monkey to kill **ALL** of your pods regardless of status (including not ready and not running pods). Does not require `kill-value`. **Use this label carefully.**
* `fixed` if you want to kill a specific number of running pods with `kill-value`. If you overspecify, it will kill **all** running pods and issue a warning.
* `random-max-percent` to specify a *maximum* `%` with `kill-value` that can be killed. At the scheduled time, a uniform *random specified* `%` of the running pods will be terminated.
* `fixed-percent` to specify a *fixed* `%` with `kill-value` that can be killed. At the scheduled time, a specified *fixed* `%` of the running pods will be terminated.


**`kube-monkey/kill-value`**: Specify value for kill-mode
* if `fixed`, provide an integer of pods to kill
* if `random-max-percent`, provide a number from `0`-`100` to specify the max `%` of pods kube-monkey can kill
* if `fixed-percent`, provide a number from `0`-`100` to specify the `%` of pods to kill

#### Example of opted-in Deployment killing one pod per purge

```yaml
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: monkey-victim
  namespace: app-namespace
spec:
  template:
    metadata:
      labels:
        kube-monkey/enabled: enabled
        kube-monkey/identifier: monkey-victim
        kube-monkey/mtbf: '2'
        kube-monkey/kill-mode: "fixed"
        kube-monkey/kill-value: '1'
[... omitted ...]
```

For newer versions of kubernetes you may need to add the labels to the k8s app metadata as well.

```yaml
---
apiVersion: extensions/v1beta1
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
  template:
    metadata:
      labels:
        kube-monkey/enabled: enabled
        kube-monkey/identifier: monkey-victim
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
2. For each eligible k8s app, flip a biased coin (bias determined by `kube-monkey/mtbf`) to determine if a pod for that k8s app should be killed today
3. For each victim, calculate a random time when a pod will be killed

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

Configuration keys and descriptions can be found in [`config/param/param.go`](https://github.com/asobti/kube-monkey/blob/master/config/param/param.go)

#### Example config.toml file
```toml
[kubemonkey]
dry_run = true                           # Terminations are only logged
run_hour = 8                             # Run scheduling at 8am on weekdays
start_hour = 10                          # Don't schedule any pod deaths before 10am
end_hour = 16                            # Don't schedule any pod deaths after 4pm
blacklisted_namespaces = ["kube-system"] # Critical apps live here
time_zone = "America/New_York"           # Set tzdata timezone example. Note the field is time_zone not timezone
```

#### Example environment variables
```
KUBEMONKEY_DRY_RUN=true
KUBEMONKEY_RUN_HOUR=8
KUBEMONKEY_START_HOUR=10
KUBEMONKEY_END_HOUR=16
KUBEMONKEY_BLACKLISTED_NAMESPACES=kube-system
KUBEMONKEY_TIME_ZONE=America/New_York
```
#### Example Config to test kube-monkey works by enabling debug mode
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
* `{$date}`: attack's date
* `{$error}`: result's error, if any

```json
  message: '{
            "what": "Kube-monkey attack of {$name} in {$namespace}",
            "who": "{$name}",
            "when": {$timestamp}
           }'
```

The header supports a special placeholder to retrieve the value of an environment variable.
This is useful when calling an API that has a protected endpoint.
A typical scenario will be to pass an API token to the Kube-monkey container, this token is stored in a Kubernetes Secret and you want to pass it via an environment variable.

```json
headers = ["api-key:{$env:API_TOKEN}", "Content-Type:application/json"]
```

`{$env:API_TOKEN}` will be replaced by the environment variable `API_TOKEN` value.

## Deploying

**Manually**
1. First, deploy the expected `kube-monkey-config-map` configmap in the namespace you intend to run kube-monkey in (for example, the `kube-system` namespace). Make sure to define the keyname as `config.toml`

> For example `kubectl create configmap km-config --from-file=config.toml=km-config.toml` or `kubectl apply -f km-config.yaml`

2. Run kube-monkey as a k8s app within the Kubernetes cluster, in a namespace that has permissions to kill Pods in other namespaces (eg. `kube-system`).

See dir [`examples/`](https://github.com/asobti/kube-monkey/tree/master/examples) for example Kubernetes yaml files.

3. You should be able to see debug logs by `kubectl logs -f deployment.apps/kube-monkey --namespace=kube-system`  here the `deployment.apps/kube-monkey` is the k8s deployment for kube-monkey.


**Helm Chart**  
A helm chart is provided that assumes you have already compiled and uploaded the container to your own container repository.  Once uploaded, you need to edit the value of `image.repository` to point at the location of your container, by default it is pointed to `ayushsobti/kube-monkey`.

Helm can then be executed using default values
```bash
helm install --name $release helm/kubemonkey
```
refer [kube-monkey helm chart README.md](https://github.com/asobti/kube-monkey/blob/master/helm/kubemonkey/README.md)

## Logging

kube-monkey uses [glog](github.com/golang/glog) and supports all command-line features for glog. To specify a custom v level or a custom log directory on the pod, see  `args: ["-v=5", "-log_dir=/path/to/custom/log"]` in the [example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml)

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

## Compatibility with Kubernetes

kube-monkey is built using v7.0 of [kubernetes/client-go](https://github.com/kubernetes/client-go). Refer to the
[Compatibility Matrix](https://github.com/kubernetes/client-go#compatibility-matrix) to see which
versions of Kubernetes are compatible.

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
oc run kube-monkey --image=docker.io/ayushsobti/kube-monkey:v0.3.0 --command -- /kube-monkey -v=5 -log_dir=/var/log/kube-monkey
oc volume dc/kube-monkey --add --name=kubeconfigmap -m /etc/kube-monkey -t configmap --configmap-name=kube-monkey-config-map
```

## Ways to contribute

See [How to Contribute](https://github.com/asobti/kube-monkey/blob/master/CONTRIBUTING.md)
