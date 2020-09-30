#!/bin/sh
set -e

if [ "$(git status --porcelain | wc -l)" -ne "0" ]; then
  echo "Dirty workspace detected. This typically indicates 'go fmt' changed some files?" \
       "\n Run 'make' locally to verify."
  git status --porcelain
  exit 1
else
  exit 0
fi
