#!/usr/bin/env bash
# Static verification only: never start the server or run application tests.
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Use the installed, pinned toolchain instead of silently downloading another.
export GOTOOLCHAIN=local
export GOWORK=off
export GOFLAGS="${GOFLAGS:+${GOFLAGS} }-mod=readonly"
expected="$(<.go-version)"
actual="$(go env GOVERSION)"
if [[ "$actual" != "go$expected" ]]; then
    printf 'Go %s is required; found %s. Put the pinned Go bin directory on PATH.\n' "$expected" "$actual" >&2
    exit 1
fi

printf '\n==> Shell syntax\n'
for script in scripts/*.sh; do
    [[ -f "$script" ]] || continue
    bash -n "$script"
done

printf '\n==> Module integrity\n'
go mod verify

printf '\n==> Go formatting (read-only)\n'
unformatted="$("$(go env GOROOT)/bin/gofmt" -l ./*.go cmd pkg)"
if [[ -n "$unformatted" ]]; then
    printf 'Run gofmt on these files:\n%s\n' "$unformatted" >&2
    exit 1
fi

printf '\n==> Go vet\n'
go vet ./...

printf '\n==> Go build (compile only)\n'
go build ./...

printf '\nStatic checks passed. No application tests or server were run.\n'
