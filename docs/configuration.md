# Configuration reference

kube-monkey is configured by a toml file at `/etc/kube-monkey/config.toml`, by environment
variables, or by both. When you deploy by hand the configmap holding that file has to exist
before the kube-monkey deployment.

If you install with the Helm chart you set these through chart values instead. See
[Helm chart](helm-chart.md) for the mapping.

## Environment variables

Every setting has an environment variable. The name is the toml path in upper case with the
dots replaced by underscores:

| Toml | Environment variable |
| --- | --- |
| `kubemonkey.dry_run` | `KUBEMONKEY_DRY_RUN` |
| `kubemonkey.time_zone` | `KUBEMONKEY_TIME_ZONE` |
| `debug.enabled` | `DEBUG_ENABLED` |
| `metrics.address` | `METRICS_ADDRESS` |

Note there is no shared prefix. Only the `kubemonkey` section starts with `KUBEMONKEY_`,
because that is its section name.

## Scheduling

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `kubemonkey.dry_run` | bool | `true` | Log terminations instead of carrying them out |
| `kubemonkey.time_zone` | string | `America/Los_Angeles` | tzdata timezone the hours below are read in. Note the key is `time_zone`, not `timezone` |
| `kubemonkey.run_hour` | int | `8` | Hour of the weekday when the day's schedule is built. Nothing is terminated at this hour. Must be less than `start_hour`, and in `[0,23]` |
| `kubemonkey.start_hour` | int | `10` | Earliest hour a termination may happen. Must be less than `end_hour`, and in `[0,23]` |
| `kubemonkey.end_hour` | int | `16` | Latest hour a termination may happen. Must be in `[0,23]` |
| `kubemonkey.graceperiod_sec` | int | `5` | Seconds a pod is given to shut down before Kubernetes hard kills it |

Set `start_hour` and `end_hour` to a time when service owners are expected to be available.

`run_hour` is when the day's plan is built, not when pods die. The time between `run_hour`
and `start_hour` is there for you to read the plan and opt anything out of it before the
first termination. See [How it works](how-it-works.md) for what happens in that gap.

## Namespace lists

kube-monkey can be configured with two lists of namespaces:

- a **whitelist**, where only apps in a whitelisted namespace can be touched
- a **blacklist**, where apps in a blacklisted namespace are never touched

An app has to pass both lists. Where the two overlap, the blacklist wins.

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `kubemonkey.whitelisted_namespaces` | list | `[""]` | Namespaces where terminations are allowed |
| `kubemonkey.blacklisted_namespaces` | list | `["kube-system"]` | Namespaces never touched |

To disable the blacklist, provide `[""]` in `blacklisted_namespaces`. The whitelist is off by
default, which is `[""]` in `whitelisted_namespaces` and means every namespace.

### Patterns

Entries in either list are shell-style patterns, so a namespace is matched when it matches
any entry:

| Entry | Matches |
| --- | --- |
| `kube-system` | `kube-system` only |
| `team-?` | `team-a`, `team-b`, but not `team-ab` |
| `*-prod` | `shop-prod`, `checkout-prod` |
| `*` | every namespace |

Namespace names never contain `*` or `?`, so a plain name still matches nothing but itself.

Patterns are checked when the config loads, and a malformed one stops kube-monkey from starting.

!!! info "Patterns need cluster-wide list permission"

    Because the lists hold patterns, kube-monkey lists workloads across the whole cluster and
    applies the lists to the result. It needs permission to list deployments, statefulsets and
    daemonsets cluster-wide, which the Helm chart grants. An install that limits kube-monkey
    to a Role in each namespace will log an error and schedule nothing.

## Overriding the apiserver

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `kubernetes.host` | string | from in-cluster config | Host URL for the Kubernetes apiserver |

Use this when the apiserver address from the in-cluster config does not work for you, for
example because the certificate does not carry the right SAN. Two cases come up often:

- client-go does not support [cluster DNS](https://github.com/kubernetes/client-go/blob/master/rest/config.go#L331)
  explicitly, and carries a `// TODO: switch to using cluster DNS.` note in the code
- on an unauthenticated system you may need to force the http apiserver endpoint

```toml
[kubernetes]
host = "https://your-apiserver-url.com:apiport"
```

## Debug

Debug mode is for confirming an install works. It is not a setting to leave on.

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `debug.enabled` | bool | `false` | Turn debug mode on |
| `debug.schedule_delay` | int | `30` | Seconds after startup before scheduling runs. Lower it to see a schedule sooner |
| `debug.force_should_kill` | bool | `false` | Schedule a termination for every eligible app, so the probability of a kill is 1 |
| `debug.schedule_immediate_kill` | bool | `false` | Schedule terminations in the next 60 seconds instead of between `start_hour` and `end_hour` |

```toml
[debug]
enabled = true
schedule_immediate_kill = true
```

!!! warning

    With `schedule_immediate_kill` on, kube-monkey attacks every 60 seconds and ignores the
    hours you configured.

## Other sections

- [Notifications](notifications.md) covers the `notifications` section
- [Metrics](metrics.md) covers the `metrics` section
- [Logging](logging.md) covers the glog command line flags, which are not part of this file

## Full examples

=== "config.toml"

    ```toml
    [kubemonkey]
    dry_run = true                           # Terminations are only logged
    run_hour = 8                             # Build the day's schedule at 8am. Nothing dies yet
    start_hour = 10                          # Don't schedule any pod deaths before 10am
    end_hour = 16                            # Don't schedule any pod deaths after 4pm
    blacklisted_namespaces = ["kube-system"] # Critical apps live here. Patterns like "*-prod" work too
    whitelisted_namespaces = [""]            # Every namespace. Narrow it with names or patterns like "team-*"
    time_zone = "America/New_York"           # Set tzdata timezone example. Note the field is time_zone not timezone
    ```

=== "Environment variables"

    ```bash
    KUBEMONKEY_DRY_RUN=true
    KUBEMONKEY_RUN_HOUR=8
    KUBEMONKEY_START_HOUR=10
    KUBEMONKEY_END_HOUR=16
    KUBEMONKEY_BLACKLISTED_NAMESPACES=kube-system
    KUBEMONKEY_WHITELISTED_NAMESPACES=
    KUBEMONKEY_TIME_ZONE=America/New_York
    ```

Ready-made configmaps for each of these live in
[`examples/`](https://github.com/asobti/kube-monkey/tree/master/examples).
