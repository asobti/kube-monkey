# Opting in to chaos

kube-monkey works on an opt-in model. It only schedules terminations for apps that have
explicitly agreed to have their pods killed.

Opt-in is done with labels on the app.

## The labels

| Label | Required | Meaning |
| --- | --- | --- |
| `kube-monkey/enabled` | yes | Set to `enabled` to opt in |
| `kube-monkey/mtbf` | yes | Mean time between failures |
| `kube-monkey/identifier` | yes | A unique name for the app |
| `kube-monkey/kill-mode` | no | How many pods to kill, defaults to one |
| `kube-monkey/kill-value` | no | The number or percentage the kill mode needs |

### `kube-monkey/enabled`

Set to `"enabled"` to opt in. Any other value, or no label at all, and kube-monkey walks past.

### `kube-monkey/mtbf`

Mean time between failures, as a whole number and a unit: `d` for days, `h` for hours or `m`
for minutes.

Set to `"3d"` and the app can expect to have a pod killed approximately every third weekday.
Set to `"2h"` and it can expect to lose a pod every two hours.

A value with no unit is read as days, so `"3"` and `"3d"` mean the same thing. The shortest
mean time between failures is one minute.

!!! note "Short mtbf values still respect the window"

    All terminations happen inside the daily run window set by `start_hour` and `end_hour`.
    An mtbf shorter than a day packs that day's terminations into that window rather than
    spreading them around the clock.

### `kube-monkey/identifier`

A unique identifier for the app. The recommendation is to use the app's own name.

It also gives you a way to pick the victim's pods by hand. If you repeat this label on the
pod template, kube-monkey looks for pods carrying `kube-monkey/identifier: foo` instead of
using the app's pod selector. Two apps sharing an identifier are then treated as one pool
of pods.

## Kill modes

The default is to kill exactly one pod of your app. `kube-monkey/kill-mode` changes that,
and `kube-monkey/kill-value` supplies the number it needs.

| `kill-mode` | `kill-value` | Result |
| --- | --- | --- |
| unset | not used | One pod |
| `kill-all` | not used | **Every** pod, including ones that are not ready or not running |
| `fixed` | whole number | That many running pods |
| `fixed-percent` | `0` to `100` | That percentage of running pods |
| `random-max-percent` | `0` to `100` | A uniform random percentage of running pods, up to this cap |

With `fixed`, asking for more pods than exist kills all the running pods and logs a warning.

!!! danger "kill-all does what it says"

    `kill-all` takes out the whole app at once, regardless of pod status. Use it deliberately.

## Where to put the labels

All of these labels go on the app's own **`metadata.labels`**. That is the only place
kube-monkey reads them from, both when it builds the daily schedule and when it re-checks
the app at termination time.

You do not need to repeat them on **`spec.template.metadata.labels`**. To find the pods to
kill, kube-monkey falls back to the app's pod selector (`spec.selector`), which every
Deployment, StatefulSet and DaemonSet already has.

## Example: one pod per attack

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
```

## Opting back out

Remove the `kube-monkey/enabled` label, or change its value. Eligibility is checked again at
termination time, so this also cancels attacks already sitting on today's schedule.
