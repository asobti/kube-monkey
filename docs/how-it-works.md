# How kube-monkey works

kube-monkey runs on a two-stage cycle. Once a day it decides what will die, and then it
waits for each of those moments to arrive.

## Scheduling time

Scheduling happens once a day on weekdays, at `run_hour`. This is when the schedule of
terminations for the current day is generated. During scheduling, kube-monkey will:

1. Generate a list of eligible apps. An app is eligible when it has opted in, is not in a
   blacklisted namespace, and is in a whitelisted namespace if a whitelist is set.
2. For each eligible app, work out how many pods to kill today from `kube-monkey/mtbf`.
   An app is killed `24h / mtbf` times a day, so an mtbf of a day or more gives at most one
   termination, and a shorter one gives several.
3. For each termination, calculate a random time during the day when a pod will be killed.

## Termination time

This is the randomly generated moment when a victim app has a pod killed. At termination
time, kube-monkey will:

1. Check the app is still eligible. It may have opted out, been blacklisted, or dropped off
   the whitelist since the schedule was built.
2. Re-read the app's `kube-monkey/kill-mode` and `kube-monkey/kill-value`.
3. Delete pods according to that kill mode and value.

Because both checks happen again at termination time, removing the `kube-monkey/enabled`
label is enough to call off an attack that is already on today's schedule.

## Which pods get picked

kube-monkey finds the victim's pods through the app's own pod selector, `spec.selector`,
which every Deployment, StatefulSet and DaemonSet already has.

You can override that. If you repeat `kube-monkey/identifier` on the pod template, kube-monkey
looks for pods carrying that label instead of using the selector. Two apps sharing an
identifier are then treated as one pool of pods.

## How it sees the cluster

Namespace lists hold shell-style patterns, so kube-monkey cannot ask the API server for one
namespace at a time. It lists workloads across the whole cluster and applies the lists to the
result.

That means it needs permission to list deployments, statefulsets and daemonsets cluster-wide.
The Helm chart grants this. An install that limits kube-monkey to a Role in each namespace
will log an error and schedule nothing.

## The daily window

Every termination lands inside the window between `start_hour` and `end_hour`. Set that
window to hours when the people who own the services are around to notice.

An mtbf shorter than a day does not spread terminations across the clock. It packs that day's
terminations into the same window.
