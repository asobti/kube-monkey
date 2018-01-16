## kube-monkey
kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey) for [Kubernetes](http://kubernetes.io/) 
clusters. It randomly deletes Kubernetes pods in the cluster encouraging and validating the development of failure-resilient
services.

--

kube-monkey runs at a pre-configured hour (`run_hour`, defaults to 8am) on weekdays, and builds a schedule of deployments that will face a random
Pod death sometime during the same day. The time-range during the day when the random pod Death might occur is configurable and
defaults to 10am to 4pm.

kube-monkey can be configured with a list of namespaces to blacklist - any deployments within a blacklisted namespace will not 
be touched.

## Opting-In to Chaos

kube-monkey works on an opt-in model and will only schedule terminations for Deployments that have explicitly agreed 
to have their pods terminated by kube-monkey.

Opt-in is done by setting the following labels on a Kubernetes Deployment:

**`kube-monkey/enabled`**: Set to **`"enabled"`** to opt-in to kube-monkey  
**`kube-monkey/mtbf`**: Mean time between failure (in days). For example, if set to **`"3"`**, the Deployment can expect to have a Pod
killed approximately every third weekday.  
**`kube-monkey/identifier`**: A unique identifier for the deployment (eg. the deployment's name). This is used to identify the pods 
that belong to a Deployment as Pods inherit labels from their Deployment.  
**`kube-monkey/kill-all`**: Set this label's value to `"kill-all"` if you want kube-monkey to kill ALL of your pods. Default behavior in the absence of this label is to kill only ONE pod. **Use this label carefully.**


#### Example of opted-in Deployment

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
[... omitted ...]
```

## How kube-monkey works

#### Scheduling time
Scheduling happens once a day on Weekdays - this is when a schedule for terminations for the current day is generated.   
During scheduling, kube-monkey will:  
1. Generate a list of eligible deployments (deployments that have opted-in and are not blacklisted)  
2. For each eligible deployment, flip a biased coin (bias determined by `kube-monkey/mtbf`) to determine if a pod for that deployment should be killed today  
3. For each victim, calculate a random time when a pod will be killed

#### Termination time
This is the randomly generated time during the day when a victim Deployment will have a pod killed.   
At termination time, kube-monkey will:  
1. Check if the deployment is still eligible (has not opted-out or been blacklisted since scheduling)  
2. Get a list of running pods for the deployment  
3. Select one random pod and delete it  

## Building

Clone the repository and build the container.

```
$ go get github.com/asobti/kube-monkey
$ cd $GOPATH/src/github.com/asobti/kube-monkey
$ make container
```

## Configuring
kube-monkey is configured by a toml file placed at `/etc/kube-monkey/config.toml`.  
Configuration keys and descriptions can be found in [`config/param/param.go`](https://github.com/asobti/kube-monkey/blob/master/config/param/param.go)

#### Example config file

```toml
[kubemonkey]
dry_run = true                           # Terminations are only logged
run_hour = 8                             # Run scheduling at 8am on weekdays
start_hour = 10                          # Don't schedule any pod deaths before 10am
end_hour = 16                            # Don't schedule any pod deaths after 4pm
blacklisted_namespaces = ["kube-system"] # Critical deployments live here
```

## Deploying

Run kube-monkey as a Deployment within the Kubernetes cluster, in a namespace that has permissions to kill Pods
in other namespaces (eg. `kube-system`).

See dir [`examples/`](https://github.com/asobti/kube-monkey/tree/master/examples) for example Kubernetes yaml files.

## Logging

kube-monkey uses glog and supports all command-line features for glog. To specify a custom v level or a custom log directory on the pod, see  `args: ["-v=5", "-logs_dir=/path/to/custom/log"]` in the [example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml)

## Compatibility with Kubernetes

kube-monkey is built using v1.5 of [kubernetes/client-go](https://github.com/kubernetes/client-go). Refer to the 
[Compatibility Matrix](https://github.com/kubernetes/client-go#compatibility-matrix) to see which 
versions of Kubernetes are compatible.

## To do

- Add tests
- Use a logging library like [glog](https://github.com/golang/glog)
