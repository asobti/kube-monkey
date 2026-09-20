[![Documentation](https://img.shields.io/badge/docs-kube--monkey-blue?logo=materialformkdocs&logoColor=white)](https://asobti.github.io/kube-monkey/)
[![Build](https://github.com/asobti/kube-monkey/actions/workflows/go.yml/badge.svg)](https://github.com/asobti/kube-monkey/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docker Pulls](https://img.shields.io/docker/pulls/ayushsobti/kube-monkey?label=Docker%20pulls&logo=docker)](https://hub.docker.com/r/ayushsobti/kube-monkey/)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubemonkey)](https://artifacthub.io/packages/search?repo=kubemonkey)

# kube-monkey

kube-monkey is an implementation of [Netflix's Chaos Monkey](https://github.com/Netflix/chaosmonkey)
for [Kubernetes](https://kubernetes.io/) clusters. It randomly deletes pods in the cluster,
encouraging and validating the development of failure-resilient services.

Apps opt in with a label, terminations happen during the hours and days you pick, and dry
run is the default, so nothing dies until you say so.

**📖 [Documentation](https://asobti.github.io/kube-monkey/)**

## Quick start

```bash
helm repo add kubemonkey https://asobti.github.io/kube-monkey/charts/repo
helm repo update
helm install kube-monkey kubemonkey/kube-monkey --namespace kube-system
```

Opt an app in by labelling it:

```yaml
metadata:
  labels:
    kube-monkey/enabled: enabled
    kube-monkey/identifier: monkey-victim
    kube-monkey/mtbf: "2"
```

That is a kube-monkey in dry run mode and an app that expects to lose a pod on about one run
day in two. See [Getting started](https://asobti.github.io/kube-monkey/getting-started/) for the
walk through, including how to watch a real termination before you trust it with a live
namespace.

## Documentation

| | |
|---|---|
| [Getting started](https://asobti.github.io/kube-monkey/getting-started/) | From install to your first termination |
| [How it works](https://asobti.github.io/kube-monkey/how-it-works/) | Scheduling time and termination time |
| [Opting in to chaos](https://asobti.github.io/kube-monkey/opting-in/) | The labels, and the kill modes |
| [Configuration](https://asobti.github.io/kube-monkey/configuration/) | Every setting and its default |
| [Helm chart](https://asobti.github.io/kube-monkey/helm-chart/) | Values and common installs |
| [Metrics](https://asobti.github.io/kube-monkey/metrics/) | Prometheus endpoint |
| [Notifications](https://asobti.github.io/kube-monkey/notifications/) | Post attacks to Slack or your own API |

## Contributing

See [How to contribute](CONTRIBUTING.md). Join us at
[#kube-monkey](https://kubernetes.slack.com/messages/kube-monkey) on Kubernetes Slack.

## License

Apache License v2.0 - see [LICENSE](LICENSE) for details.
