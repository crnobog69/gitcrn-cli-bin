#!/usr/bin/env bash
set -euo pipefail

COMMIT_MSG='❄'
BRANCH='master'
REMOTES=('gitcrn' 'origin')

git add .
if git diff --cached --quiet; then
  echo "Nema izmena za commit. Preskacem commit."
else
  git commit -m "$COMMIT_MSG"
fi

for remote in "${REMOTES[@]}"; do
  if [[ -n "$BRANCH" ]]; then
    git push "$remote" "$BRANCH"
  else
    git push "$remote"
  fi
done
