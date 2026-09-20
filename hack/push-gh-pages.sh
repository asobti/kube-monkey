#!/usr/bin/env bash
#
# Commits and pushes the gh-pages checkout the caller is sitting in.
#
# Usage: push-gh-pages.sh <commit message> [path...]
#
# The docs site and the Helm chart repo share this branch and are published by
# different workflows, so a push can lose a race. Rebase and retry rather than
# failing a release over it.

set -euo pipefail

message=${1:?usage: push-gh-pages.sh <commit message> [path...]}
shift

if [ "$#" -eq 0 ]; then
  set -- .
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

git add -- "$@"

if git diff --cached --quiet; then
  echo "Nothing to publish"
  exit 0
fi

git commit -m "$message"

for attempt in 1 2 3; do
  if git push; then
    exit 0
  fi
  echo "Push rejected on attempt $attempt, rebasing onto the latest gh-pages"
  git pull --rebase
done

echo "Could not push to gh-pages after 3 attempts" >&2
exit 1
