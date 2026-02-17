#!/usr/bin/env bash
set -euo pipefail

BRANCH='master'
REMOTES=('gitcrn' 'origin')

for remote in "${REMOTES[@]}"; do
  if [[ -n "$BRANCH" ]]; then
    git pull "$remote" "$BRANCH"
  else
    git pull "$remote"
  fi
done
