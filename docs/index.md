---
title: Chaos Monkey for Kubernetes
hide:
  - navigation
  - toc
---

<div class="km-hero" markdown>

# kube-monkey

Your services claim to be resilient. kube-monkey finds out. It deletes pods at random
during working hours, on apps that have opted in, so the failures happen while you are
awake to watch them.

[Get started](getting-started.md){ .md-button .md-button--primary }
[Install with Helm](helm-chart.md){ .md-button }

</div>

<div class="km-badges" markdown>

[![Build](https://github.com/asobti/kube-monkey/actions/workflows/go.yml/badge.svg)](https://github.com/asobti/kube-monkey/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/asobti/kube-monkey)](https://goreportcard.com/report/github.com/asobti/kube-monkey)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docker Pulls](https://img.shields.io/docker/pulls/ayushsobti/kube-monkey?label=Docker%20pulls&logo=docker)](https://hub.docker.com/r/ayushsobti/kube-monkey/)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubemonkey)](https://artifacthub.io/packages/search?repo=kubemonkey)

</div>

kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey)
for [Kubernetes](https://kubernetes.io/) clusters. It randomly deletes pods in the cluster,
encouraging and validating the development of failure-resilient services.

<div class="grid cards" markdown>

-   :material-hand-back-right:{ .lg .middle } __Opt-in, never opt-out__

    ---

    Nothing is touched until an app asks for it with a label. A team adopts chaos when
    it is ready, not when the cluster operator flips a switch.

    [:octicons-arrow-right-24: Opting in to chaos](opting-in.md)

-   :material-clock-outline:{ .lg .middle } __Office hours only__

    ---

    Terminations are scheduled inside a window you choose, on weekdays. Pods die while
    the people who own them are at their desks.

    [:octicons-arrow-right-24: How it works](how-it-works.md)

-   :material-eye-outline:{ .lg .middle } __Dry run by default__

    ---

    Out of the box kube-monkey only logs what it would have killed. You see the blast
    radius on paper before anything real happens.

    [:octicons-arrow-right-24: Configuration](configuration.md)

-   :material-target:{ .lg .middle } __Pick the blast radius__

    ---

    Kill one pod, a fixed number, a fixed percentage, a random percentage up to a cap,
    or the whole app. Per app, set by a label.

    [:octicons-arrow-right-24: Kill modes](opting-in.md#kill-modes)

-   :material-chart-line:{ .lg .middle } __Prometheus metrics__

    ---

    Schedules built, terminations attempted, pods actually deleted. Chart the chaos and
    alert on it like anything else.

    [:octicons-arrow-right-24: Metrics](metrics.md)

-   :material-bell-outline:{ .lg .middle } __Tell your team__

    ---

    Post every attack to a Slack webhook or your own API, with placeholders for the
    victim, the time and the outcome.

    [:octicons-arrow-right-24: Notifications](notifications.md)

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
