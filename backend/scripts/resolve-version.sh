#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
BACKEND_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
REPO_DIR="$(CDPATH= cd -- "$BACKEND_DIR/.." && pwd)"
VERSION="$(tr -d '\r\n' < "$BACKEND_DIR/cmd/server/FORK_VERSION")"

if ! printf '%s\n' "$VERSION" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$'; then
  echo 'Invalid FORK_VERSION' >&2
  exit 1
fi

# Upstream v* tags never select the fork version.
if command -v git >/dev/null 2>&1; then
  TAGS="$(git -C "$REPO_DIR" tag --points-at HEAD --list 'tr-v*' 2>/dev/null || true)"
  for TAG in $TAGS; do
    if [ "$TAG" != "tr-v$VERSION" ]; then
      echo "Fork tag $TAG does not match FORK_VERSION $VERSION" >&2
      exit 1
    fi
  done
fi

printf '%s\n' "$VERSION"
