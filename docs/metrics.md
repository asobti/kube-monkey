# Metrics

kube-monkey can serve Prometheus metrics about the schedules it builds and the terminations it
runs. The endpoint is off by default. When it is on, kube-monkey listens on `/metrics` at the
configured address.

## Settings

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `metrics.enabled` | bool | `false` | Serve the metrics endpoint |
| `metrics.address` | string | `:8080` | Host and port to listen on, in `host:port` form. Leave the host out to listen on every interface |

=== "config.toml"

    ```toml
    [metrics]
      enabled = true
      address = ":8080"
    ```

=== "Environment variables"

    ```bash
    METRICS_ENABLED=true
    METRICS_ADDRESS=:8080
    ```

## Exposed metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `kube_monkey_schedule_size` | gauge | | Terminations on the latest schedule |
| `kube_monkey_last_schedule_timestamp_seconds` | gauge | | When the latest schedule was generated |
| `kube_monkey_scheduled_terminations_total` | counter | `kind`, `namespace`, `name` | Terminations kube-monkey has put on a schedule |
| `kube_monkey_terminations_total` | counter | `kind`, `namespace`, `name`, `result` | Terminations kube-monkey has carried out. `result` is `success` or `failure`, where a failure means the whole termination was skipped or errored, for example because the victim opted out after it was scheduled |
| `kube_monkey_pods_terminated_total` | counter | `kind`, `namespace`, `name` | Pods kube-monkey has deleted. Stays at zero in dry run mode because no pod is really deleted |

The standard Go runtime and process metrics are exposed as well.

!!! tip "Dry run tells them apart"

    In dry run mode `kube_monkey_terminations_total` still climbs while
    `kube_monkey_pods_terminated_total` stays at zero. That gap is a good way to confirm
    kube-monkey is scheduling work without touching anything.

## Scraping

To let a Prometheus that discovers pods scrape kube-monkey, annotate the pod template in the
[example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml):

```yaml
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
```

## With the Helm chart

The chart creates a Service for the endpoint when `config.metrics.enabled` is set, and a
ServiceMonitor for the Prometheus Operator when `serviceMonitor.enabled` is set too.

```bash
helm install my-release kubemonkey/kube-monkey \
  --set config.metrics.enabled=true \
  --set serviceMonitor.enabled=true
```

## Example configmap

A ready-made example lives at
[`examples/metrics-configmap.yaml`](https://github.com/asobti/kube-monkey/tree/master/examples/metrics-configmap.yaml).
