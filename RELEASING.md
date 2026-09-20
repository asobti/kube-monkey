# Releasing

The full runbook lives at <https://asobti.github.io/kube-monkey/releasing/>, or in
[`docs/releasing.md`](docs/releasing.md) if you would rather read it here.

In short:

- **New app version:** open a pull request bumping `appVersion` and `version` in
  `helm/kubemonkey/Chart.yaml`, `image.tag` in `helm/kubemonkey/values.yaml`, and the
  versions named in `helm/kubemonkey/README.md` and `docs/helm-chart.md`. Once merged,
  push the tag `v<appVersion>`.
- **Chart-only change:** bump `version` and the `--version` lines, merge, then run the
  **Publish release** workflow by hand on master.
- **Docs change:** nothing to do, merging to master publishes the site.

Check your work before tagging with the same script CI runs:

```bash
RELEASE_TAG=v0.7.0 ./hack/verify-release.sh
```
