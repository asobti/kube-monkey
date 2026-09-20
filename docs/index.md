---
title: Chaos Monkey for Kubernetes
hide:
  - navigation
  - toc
---

<div class="km-hero" markdown>

# Chaos on a timetable

<p class="km-lede">kube-monkey deletes pods at random on apps that have opted in. It publishes
the day's terminations each morning and runs them during working hours, while the people who
own the service are still at their desks.</p>

[Get started](getting-started.md){ .md-button .md-button--primary }
[Install with Helm](helm-chart.md){ .md-button }

<div class="km-board" markdown>

<div class="km-board-head">
  <b>Today's schedule</b>
  <span>4 terminations</span>
</div>

| Kind | Namespace | Name | Termination time |
| --- | --- | --- | --- |
| v1.Deployment | default | monkey-victim | 10:14:07 |
| v1.Deployment | payments | checkout-api | 11:32:55 |
| v1.StatefulSet | payments | ledger | 13:47:19 |
| v1.DaemonSet | observability | log-shipper | 15:02:41 |

<p class="km-board-foot">kube-monkey writes this to the log at <code>run_hour</code> on each
of its <code>run_days</code>. Every termination lands between <code>start_hour</code> and
<code>end_hour</code>.</p>

</div>

</div>

kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey)
for [Kubernetes](https://kubernetes.io/) clusters. It randomly deletes pods in the cluster,
encouraging and validating the development of failure-resilient services.

<div class="km-split" markdown>
<div markdown>

### Safe by default

[Opt in, never opt out](opting-in.md)
:   Nothing is touched until an app asks for it with a label. A team adopts chaos when it is
    ready, not when the cluster operator flips a switch.

[Office hours only](how-it-works.md)
:   Terminations are scheduled inside a window you choose, on the days you choose. Pods die
    while the people who own them are at their desks.

[Dry run until you say otherwise](configuration.md)
:   Out of the box kube-monkey only logs what it would have killed. You see the blast radius
    on paper before anything real happens.

</div>
<div markdown>

### What you can tune

[Blast radius](opting-in.md#kill-modes)
:   Kill one pod, a fixed number, a fixed percentage, a random percentage up to a cap, or the
    whole app. Set per app, by a label.

[Prometheus metrics](metrics.md)
:   Schedules built, terminations attempted, pods actually deleted. Chart the chaos and alert
    on it like anything else.

[Notifications](notifications.md)
:   Post every attack to a Slack webhook or your own API, with placeholders for the victim,
    the time and the outcome.

</div>
</div>

## Sixty seconds to your first termination

```bash
helm repo add kubemonkey https://asobti.github.io/kube-monkey/charts/repo
helm repo update
helm install kube-monkey kubemonkey/kube-monkey --namespace kube-system
```

That gives you a kube-monkey in dry run mode, touching nothing. Label an app to opt it in:

```yaml
metadata:
  labels:
    kube-monkey/enabled: enabled
    kube-monkey/identifier: monkey-victim
    kube-monkey/mtbf: "2"
```

Then read [Getting started](getting-started.md) for the walk through, including how to
watch a real termination happen in debug mode before you trust it with a live namespace.

## Talk to us

Join [#kube-monkey](https://kubernetes.slack.com/messages/kube-monkey) on Kubernetes Slack,
or open an issue on [GitHub](https://github.com/asobti/kube-monkey/issues).

<div class="km-badges" markdown>

[![Build](https://github.com/asobti/kube-monkey/actions/workflows/go.yml/badge.svg)](https://github.com/asobti/kube-monkey/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docker Pulls](https://img.shields.io/docker/pulls/ayushsobti/kube-monkey?label=Docker%20pulls&logo=docker)](https://hub.docker.com/r/ayushsobti/kube-monkey/)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubemonkey)](https://artifacthub.io/packages/search?repo=kubemonkey)

</div>
