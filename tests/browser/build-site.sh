#!/usr/bin/env bash
# Regenerate build/ with the Go generator. Runs before the browser tests (npm
# pretest, and again as part of Playwright's webServer command) so the browser
# never inspects a stale site.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"

# The Go toolchain isn't always on a non-login shell's PATH; check the usual
# install locations before giving up.
if ! command -v go >/dev/null 2>&1; then
  for dir in "$HOME/.local/go/bin" /usr/local/go/bin /usr/lib/go/bin; do
    if [ -x "$dir/go" ]; then
      PATH="$PATH:$dir"
      export PATH
      break
    fi
  done
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go not found on PATH — install Go or add it to PATH" >&2
  exit 1
fi

go build -o ssg .

# Generate into an empty build/, the way CI does on a fresh checkout. The
# generator writes over what it produces but never removes what it no longer
# produces, so an incremental build/ can keep serving a page that has been
# renamed or deleted — and the browser tests would happily pass against it.
if [ -d "$root/build" ]; then
  rm -rf "$root/build"
fi

./ssg
