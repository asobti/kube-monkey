# kube-monkey
kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey) for [Kubernetes](http://kubernetes.io/) clusters. It randomly deletes Kubernetes pods in the cluster encouraging and validating the development of failure-resilient services.

---

kube-monkey runs at a pre-configured hour (`run_hour`, defaults to 8am) on weekdays, and builds a schedule of deployments that will face a random
Pod death sometime during the same day. The time-range during the day when the random pod Death might occur is configurable and defaults to 10am to 4pm.

kube-monkey can be configured with a list of namespaces 
* to blacklist (any deployments within a blacklisted namespace will not be touched) 
* to whitelist (only deployments within a whitelisted namespace that are not blacklisted will be scheduled)
The blacklist overrides the whitelist. The config will be populated with default behavior (blacklist `kube-system` and whitelist `default`). To disable either the blacklist or whitelist provide `[""]` to the respective config.param

## Opting-In to Chaos

kube-monkey works on an opt-in model and will only schedule terminations for k8 apps that have explicitly agreed to have their pods terminated by kube-monkey.

Opt-in is done by setting the following labels on a Kubernetes k8 app:

**`kube-monkey/enabled`**: Set to **`"enabled"`** to opt-in to kube-monkey  
**`kube-monkey/mtbf`**: Mean time between failure (in days). For example, if set to **`"3"`**, the k8 app can expect to have a Pod
killed approximately every third weekday.  
**`kube-monkey/identifier`**: A unique identifier for the k8 app (eg. the k8 app's name). This is used to identify the pods 
that belong to a k8 app as Pods inherit labels from their k8 app.  
**`kube-monkey/kill-mode`**: Set this label's value to  
* `"kill-all"` if you want kube-monkey to kill ALL of your pods regardless of status. Does not require kill-value. Default behavior in the absence of this label is to kill only ONE pod. **Use this label carefully.**
* `fixed` if you want to kill a specific number of running pods with kill-value. If you overspecify, it will kill all running pods and issue a warning.
* `random-max-percent` to specify a maximum % with kill-value that can be killed. At the scheduled time, a uniform random specified % of the running pods will be terminated.
**`kube-monkey/kill-value`**: Specify value for kill-mode
* if `fixed`, provide an integer of pods to kill
* if `random-max-percent`, provide a number from 0-100 to specify the max % of pods kube-monkey can kill

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
        kube-monkey/identifier: monkey-victim-pods
        kube-monkey/mtbf: '2'
        kube-monkey/kill-mode: "fixed"
        kube-monkey/kill-value: 1
[... omitted ...]
```

For newer versions of kubernetes you may need to add the labels to the k8 app metadata as well.

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
    kube-monkey/kill-value: 1
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
* Since client-go does not support [cluster dns](https://github.com/kubernetes/client-go/blob/master/rest/config.go#L336) explicitly with a `// TODO: switch to using cluster DNS.` note in the code, you may need to override the apiserver. 
* If you are running an unauthenticated system, you may need to force the http apiserver enpoint. 

#### To override the apiserver specify in the config.toml file
```toml
[kubernetes]
host="https://your-apiserver-url.com"
```

## How kube-monkey works

#### Scheduling time
Scheduling happens once a day on Weekdays - this is when a schedule for terminations for the current day is generated.   
During scheduling, kube-monkey will:  
1. Generate a list of eligible k8 apps (k8 apps that have opted-in and are not blacklisted, if specified, and are whitelisted, if specified)
2. For each eligible k8 app, flip a biased coin (bias determined by `kube-monkey/mtbf`) to determine if a pod for that k8 app should be killed today  
3. For each victim, calculate a random time when a pod will be killed

#### Termination time
This is the randomly generated time during the day when a victim k8 app will have a pod killed.
At termination time, kube-monkey will:
1. Check if the k8 app is still eligible (has not opted-out or been blacklisted or removed from the whitelist since scheduling)
2. Check if the k8 app has updated kill-mode and kill-value
3. Depending on kill-mode and kill-value, execute pods

## Building

Clone the repository and build the container.

```bash
go get github.com/asobti/kube-monkey
cd $GOPATH/src/github.com/asobti/kube-monkey
make container
```

## Configuring
kube-monkey is configured by a toml file placed at `/etc/kube-monkey/config.toml` and expects the configmap to exist before the kubemonkey deployment. 

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

## Deploying

1. First deploy the expected `kube-monkey-config-map` configmap in the namespace you intend to run kube-monkey in (for example, the `kube-system` namespace). Make sure to define the keyname as `config.toml` 

> For example `kubectl create configmap km-config --from-file=config.toml=km-config.toml` or `kubectl apply -f km-config.yaml`

2. Run kube-monkey as a k8 app within the Kubernetes cluster, in a namespace that has permissions to kill Pods in other namespaces (eg. `kube-system`).

See dir [`examples/`](https://github.com/asobti/kube-monkey/tree/master/examples) for example Kubernetes yaml files.

## Logging

kube-monkey uses glog and supports all command-line features for glog. To specify a custom v level or a custom log directory on the pod, see  `args: ["-v=5", "-log_dir=/path/to/custom/log"]` in the [example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml)

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

More resources: See the [k8 logging page](https://kubernetes.io/docs/concepts/cluster-administration/logging/) suggesting [community conventions for logging severity](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md)

## Compatibility with Kubernetes

kube-monkey is built using v6.0 of [kubernetes/client-go](https://github.com/kubernetes/client-go). Refer to the 
[Compatibility Matrix](https://github.com/kubernetes/client-go#compatibility-matrix) to see which 
versions of Kubernetes are compatible.

## Ways to contribute

- Add unit [tests](https://golang.org/pkg/testing/)
- Support more k8 types
  - ~~deployments~~
  - ~~statefulsets~~
  - dameonsets
  - etc
