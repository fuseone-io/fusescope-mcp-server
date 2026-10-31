#!/usr/bin/env bash
# Regression test for check-agpl.sh: a clean module passes, a linked collector
# package fails, and a failing go list fails instead of passing vacuously.
set -uo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
stubs=$(mktemp -d)
trap 'rm -rf "$stubs"' EXIT

stub() {
	printf '#!/usr/bin/env bash\n%s\n' "$2" >"$stubs/$1"
	chmod +x "$stubs/$1"
}
stub clean 'echo github.com/fuseone-io/fusescope-mcp-server/pkg/dashboard'
stub linked 'echo github.com/SigNoz/signoz-otel-collector/cmd/signozschemamigrator/schema_migrator'
stub broken 'echo partial; exit 2'

failures=0
expect() {
	local want=$1 go_bin=$2
	(cd "$root" && GO="$go_bin" scripts/check-agpl.sh >/dev/null 2>&1)
	local got=$?
	if { [ "$want" = pass ] && [ "$got" -ne 0 ]; } || { [ "$want" = fail ] && [ "$got" -eq 0 ]; }; then
		echo "FAIL: GO=$go_bin: want $want, exit $got"
		failures=$((failures + 1))
	fi
}

expect pass "$stubs/clean"
expect fail "$stubs/linked"
expect fail "$stubs/broken"

if [ "$failures" -ne 0 ]; then
	exit 1
fi
echo "check-agpl.sh regression tests passed"
