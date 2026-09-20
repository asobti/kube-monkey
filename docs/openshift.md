# OpenShift

kube-monkey runs on OpenShift, with a little extra work on the service account permissions.

## OpenShift 4.x

```bash
git clone https://github.com/asobti/kube-monkey.git
cd examples
oc login http://someserver/ -u system:admin
oc project kube-system
oc create -f configmap.yaml
oc -n kube-system adm policy add-cluster-role-to-user edit -z default --rolebinding-name kube-monkey-edit
oc run kube-monkey --image=docker.io/ayushsobti/kube-monkey:v0.3.0 --command -- /kube-monkey -v=5 -logtostderr=true
oc set volume dc/kube-monkey --add --name=kubeconfigmap -m /etc/kube-monkey -t configmap --configmap-name=kube-monkey-config-map
```

## OpenShift 3.x

```bash
git clone https://github.com/asobti/kube-monkey.git
cd examples
oc login http://someserver/ -u system:admin
oc project kube-system
oc create -f configmap.yaml
oc -n kube-system adm policy add-role-to-user -z deployer system:deployer
oc -n kube-system adm policy add-role-to-user -z builder system:image-builder
oc -n kube-system adm policy add-role-to-group system:image-puller system:serviceaccounts:kube-system
oc run kube-monkey --image=docker.io/ayushsobti/kube-monkey:v0.4.0 --command -- /kube-monkey -v=5 -logtostderr=true
oc volume dc/kube-monkey --add --name=kubeconfigmap -m /etc/kube-monkey -t configmap --configmap-name=kube-monkey-config-map
```

!!! note "The image tags above are old"

    These commands were written against the versions named in them. Check
    [Docker Hub](https://hub.docker.com/r/ayushsobti/kube-monkey/tags/) for the current tag
    and substitute it.
