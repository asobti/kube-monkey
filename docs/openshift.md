# OpenShift

kube-monkey runs on OpenShift the same way it runs on any other cluster. Install it with
[the Helm chart](helm-chart.md) or [the manifests](manual-install.md) and use `oc` in place
of `kubectl`. The only thing that needs care is the service account.

## Permissions

kube-monkey lists workloads and deletes pods across the whole cluster, so it needs a
ClusterRole. The Helm chart creates its own service account, ClusterRole and binding, so
there is nothing extra to do:

```bash
oc project kube-system
helm install kube-monkey kubemonkey/kube-monkey --namespace kube-system
```

The example manifests carry no RBAC and run as the namespace's `default` service account,
which on OpenShift cannot touch pods in other namespaces. Grant it a role before applying
them:

```bash
oc project kube-system
oc adm policy add-cluster-role-to-user edit -z default --rolebinding-name kube-monkey-edit
oc apply -f examples/configmap.yaml
oc apply -f examples/deployment.yaml
```

`edit` gives away far more than kube-monkey needs. Prefer the chart, or copy the ClusterRole
out of it, if you can.

## DeploymentConfig is not a victim kind

kube-monkey only targets `apps/v1` Deployments, StatefulSets and DaemonSets. OpenShift's
`apps.openshift.io/v1` DeploymentConfig is none of those. It manages ReplicationControllers
directly and never creates a Deployment, so kube-monkey never sees it and the opt-in labels
on it do nothing.

The symptom is a schedule that stays empty, even with debug mode on:

```
Status Update: 0 terminations scheduled today
********** Today's schedule **********
No terminations scheduled
********** End of schedule **********
```

Red Hat deprecated DeploymentConfig in OpenShift 4.14 and points at Deployments instead.
Move the app to a Deployment and the labels start working.
