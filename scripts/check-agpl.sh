#!/usr/bin/env bash
# Fails when the module links a package from the AGPL-3.0 collector or schema
# migrator; this server is Apache-2.0. GO selects the go binary (tests stub it).
set -euo pipefail

go_bin="${GO:-go}"

if ! deps=$("$go_bin" list -deps -test ./...); then
	echo "${GITHUB_ACTIONS:+::error::}go list failed; cannot check for AGPL collector packages." >&2
	exit 1
fi

linked=$(grep -E '^github\.com/(SigNoz/signoz|fuseone-io/fusescope)-otel-collector(/|$)' <<<"$deps" || true)
if [ -n "$linked" ]; then
	printf '%s\n' "$linked"
	echo "${GITHUB_ACTIONS:+::error::}The server links AGPL collector packages." >&2
	exit 1
fi
