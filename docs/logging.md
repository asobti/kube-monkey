# Logging

kube-monkey uses [glog](https://github.com/golang/glog) and supports all of glog's command
line flags.

It logs to stderr, so `kubectl logs` shows everything and nothing is written to disk.

```bash
kubectl logs -f deployment.apps/kube-monkey --namespace kube-system
```

## Verbosity

To change the v level, see `args: ["-v=5"]` in the
[example deployment file](https://github.com/asobti/kube-monkey/tree/master/examples/deployment.yaml).
With the Helm chart, set `args.logLevel`.

The levels are used consistently, so you can pick how much you want to see:

| Level | Shows |
| --- | --- |
| 0 | Nothing |
| 1 | Highest level status info, and errors with terminations |
| 2 | Successful terminations |
| 3 | More detailed schedule status info |
| 4 | Verbose schedule and config info, for debugging |
| 5 | Auto-resolved inconsequential issues |

You can find them in the source with `grep -r 'V([0-9])' *`.

## Timestamps

Every timestamp kube-monkey prints is in `kubemonkey.time_zone`. That covers both the
`I0919 11:32:04.123456` prefix glog puts on each line and the termination times inside the
messages, so the two always agree and you can parse either one.

The prefix carries no year and no zone, so a parser has to know the configured timezone.
The termination times spell theirs out, for example
`09/19/2019 15:32:04 +0200 CEST`, so prefer those if you want the offset in the line.

!!! note "Changing the timezone needs a restart"

    kube-monkey picks up most config changes without a restart, but the timezone of the log
    timestamps is fixed when the process starts.

## Writing log files

To also write glog log files, pass `-log_dir=/path/to/custom/log`.

!!! warning "The image has an empty filesystem"

    Mount a writable volume at that path. Without one the directory cannot be created and
    kube-monkey exits.

With the Helm chart:

```yaml
args:
  logLevel: 5
  logDir: /var/log/kube-monkey

additionalVolumes:
  - name: log
    emptyDir: {}

additionalVolumeMounts:
  - name: log
    mountPath: /var/log/kube-monkey
```

## Further reading

The [Kubernetes logging page](https://kubernetes.io/docs/concepts/cluster-administration/logging/)
and the [community conventions for logging severity](https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md).
