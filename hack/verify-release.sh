#!/usr/bin/env bash
# Checks the chart in helm/kubemonkey is consistent and safe to publish.
# Run it before pushing a release tag. CI runs the same script.
#
# usage: hack/verify-release.sh [--pre-release] [published-chart-dir]
#   --pre-release  only check the chart is self consistent, for PRs where the
#                  release tag does not exist yet
#   RELEASE_TAG    when set, appVersion must match this tag
set -euo pipefail

chart_dir=helm/kubemonkey
pre_release=
if [ "${1:-}" = "--pre-release" ]; then
  pre_release=1
  shift
fi
published_dir=${1:-}

fail() {
  # GitHub turns ::error:: into an annotation. Harmless in a local shell.
  echo "::error::$*" >&2
  exit 1
}

meta=$(helm show chart "$chart_dir")
app_version=$(printf '%s\n' "$meta" | sed -n 's/^appVersion: *"\{0,1\}\([^"]*\)"\{0,1\}$/\1/p')
chart_version=$(printf '%s\n' "$meta" | sed -n 's/^version: *//p')
# Read the tag off the rendered output rather than values.yaml, so the check
# covers what actually ships.
image=$(helm template "$chart_dir" | sed -n 's/^ *image: *"\{0,1\}\([^"]*\)"\{0,1\} *$/\1/p' | head -1)
image_tag=${image##*:}

[ -n "$app_version" ] || fail "could not read appVersion from $chart_dir/Chart.yaml"
[ -n "$chart_version" ] || fail "could not read version from $chart_dir/Chart.yaml"
[ -n "$image_tag" ] || fail "could not read the image tag from the rendered deployment"

# The chart ships the image it pins, so both must name the same app release.
if [ "$image_tag" != "v$app_version" ]; then
  fail "chart pins image $image_tag but appVersion is $app_version, expected v$app_version"
fi

# The install instructions are the first thing users copy, so keep them current.
if ! grep -qF -- "--version $chart_version" "$chart_dir/README.md"; then
  fail "$chart_dir/README.md does not mention --version $chart_version"
fi
if ! grep -qF -- "$image_tag" "$chart_dir/README.md"; then
  fail "$chart_dir/README.md does not mention image tag $image_tag"
fi

helm lint "$chart_dir"

if [ -n "$pre_release" ]; then
  echo "chart $chart_version (app v$app_version) is self consistent"
  exit 0
fi

if [ -n "${RELEASE_TAG:-}" ]; then
  if [ "$RELEASE_TAG" != "v$app_version" ]; then
    fail "tag $RELEASE_TAG does not match appVersion $app_version, expected tag v$app_version"
  fi
else
  # A chart-only release must pin an image that an earlier tag already built.
  if ! git rev-parse -q --verify "refs/tags/v$app_version" >/dev/null; then
    fail "no tag v$app_version exists, so image $image_tag was never built"
  fi
fi

if [ -n "$published_dir" ] && [ -e "$published_dir/kube-monkey-$chart_version.tgz" ]; then
  fail "chart $chart_version is already published, bump version in $chart_dir/Chart.yaml"
fi

echo "chart $chart_version (app v$app_version) is ready to publish"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "chart_version=$chart_version" >> "$GITHUB_OUTPUT"
fi
