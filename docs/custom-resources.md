# Custom resources

Out of the box kube-monkey kills pods belonging to Deployments, StatefulSets and DaemonSets.

Plenty of apps are not run by any of those. An operator takes a custom resource, such as a
CloudNativePG `Cluster` or a Strimzi `Kafka`, and creates the pods itself. kube-monkey never
sees those pods, because nothing it looks at owns them.

Listing a custom resource in the config fixes that. kube-monkey then reads the resource, the
same way it reads a Deployment, and kills the pods the operator created for it.

## Configuring a resource

Each entry names one resource and says how to find its pods.

```toml
[[kubemonkey.custom_resources]]
group = "postgresql.cnpg.io"
version = "v1"
resource = "clusters"
pod_label = "cnpg.io/cluster"
```

| Key | Required | Meaning |
| --- | --- | --- |
| `group` | yes | The API group, e.g. `postgresql.cnpg.io` |
| `version` | yes | The API version, e.g. `v1` |
| `resource` | yes | The lowercase plural resource name, e.g. `clusters` |
| `pod_label` | no | The label the operator puts on the pods it creates |

Repeat the block for each resource you want covered.

!!! warning "`resource` is the plural name, not the kind"

    It is `clusters`, not `Cluster`. This is the name in the URL the API server serves the
    resource on, which is what `kubectl api-resources` prints in its first column.

### Finding the pods

`pod_label` is the important one, and it is what makes this work at all.

An operator does **not** copy the labels you put on a custom resource onto the pods it
creates. So the `kube-monkey/identifier` label that kube-monkey normally uses to find pods
is not on them. Instead, operators label every pod with the name of the resource it belongs
to. CloudNativePG uses `cnpg.io/cluster`, Strimzi uses `strimzi.io/cluster`, and so on.

Tell kube-monkey which label that is, and it looks for pods carrying
`<pod_label>: <name of the resource>`.

Check what your operator uses before you set it:

```console
$ kubectl get pods -n app --show-labels
NAME          READY   STATUS    LABELS
pg-cluster-1  1/1     Running   cnpg.io/cluster=pg-cluster,...
```

Leave `pod_label` out only if you have asked the operator to pass your own labels down to
the pods, and `kube-monkey/identifier` is among them. Without either, kube-monkey finds no
pods and quietly kills nothing.

## Giving kube-monkey permission

kube-monkey can only read a resource it has RBAC for. The Helm chart adds a rule for every
resource you list, so this is handled if you configure them through
`config.customResources`:

```yaml
config:
  customResources:
    - group: postgresql.cnpg.io
      version: v1
      resource: clusters
      podLabel: cnpg.io/cluster
```

Installing by hand means adding the rule yourself:

```yaml
- apiGroups:
  - postgresql.cnpg.io
  resources:
  - clusters
  verbs:
  - get
  - list
  - watch
```

Without it kube-monkey logs an error on each scheduling run and schedules nothing for that
resource.

A resource whose CRD is not installed is skipped quietly, so one config can cover a fleet of
clusters that do not all run the same operators.

## Opting in

The labels are the same ones every other app uses, on the custom resource's own
`metadata.labels`. See [Opting in to chaos](opting-in.md).

```yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: pg-cluster
  namespace: app
  labels:
    kube-monkey/enabled: enabled
    kube-monkey/identifier: pg-cluster
    kube-monkey/mtbf: '2'
    kube-monkey/kill-mode: "fixed"
    kube-monkey/kill-value: '1'
spec:
  instances: 3
```

With the config above, kube-monkey schedules this cluster about every other run day and
kills one of the pods labelled `cnpg.io/cluster: pg-cluster`.

!!! danger "The operator will rebuild what you kill"

    That is the point, and it is also the risk. Killing the primary of a database cluster
    triggers a failover. Start in `dry_run` mode, and start with `kill-mode: fixed` and a
    `kill-value` of `1`.
