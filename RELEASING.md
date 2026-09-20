# Releasing

There are two version numbers, and they move independently.

| Version | Where | What it covers | Released by |
|---------|-------|----------------|-------------|
| App version, e.g. `0.6.0` | `appVersion` in `helm/kubemonkey/Chart.yaml` | the binary and the Docker image | pushing the git tag `v0.6.0` |
| Chart version, e.g. `1.8.0` | `version` in `helm/kubemonkey/Chart.yaml` | the Helm chart | a commit on the `gh-pages` branch |

Charts are served from the `gh-pages` branch at
https://asobti.github.io/kube-monkey/charts/repo. Images go to
[ayushsobti/kube-monkey](https://hub.docker.com/r/ayushsobti/kube-monkey).

## Releasing a new app version

Say the new version is `0.7.0`. Open one pull request that changes:

- `helm/kubemonkey/Chart.yaml`: `appVersion` to `"0.7.0"`, and `version` to the
  next chart version
- `helm/kubemonkey/values.yaml`: `image.tag` to `v0.7.0`
- `helm/kubemonkey/README.md`: the `--version` line, the `image.tag` row in the
  table, and the `tag:` line in the values example

The chart version has to move too. The chart now points at a different image, and
publishing never overwrites a version that is already out there.

Once it is merged, tag master:

```bash
git tag v0.7.0
git push origin v0.7.0
```

That builds and pushes the image, then packages the chart and commits it to
`gh-pages`. The chart waits for the image, so it can never go out pointing at an
image that does not exist yet.

## Releasing a chart-only change

Use this when the templates or the default values change but the app does not.

Open a pull request that bumps `version` in `helm/kubemonkey/Chart.yaml` and the
`--version` line in `helm/kubemonkey/README.md`. Leave `appVersion`,
`values.yaml` and the image references alone.

Once it is merged, go to Actions, pick the **Publish release** workflow, and run
it on `master`. It skips the image build and only publishes the chart.

## Checking before you tag

```bash
./hack/verify-release.sh
```

This is the same script CI runs. It checks the chart pins the image its
`appVersion` names, the README matches, the chart lints and renders, and the
chart version is not already published.

Run on its own it expects the tag to already exist. To check a tag you have not
pushed yet:

```bash
RELEASE_TAG=v0.7.0 ./hack/verify-release.sh
```

Every pull request runs the same checks, minus the ones that need the tag.

## If the chart step fails

The image is already pushed by then, so there is no need to tag again. Fix the
chart on master, then run the **Publish release** workflow by hand as above.
