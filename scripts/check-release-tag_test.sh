#!/usr/bin/env bash
# Regression test for check-release-tag.sh: SemVer 2.0.0 tags with a v prefix pass, anything else fails.
set -uo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
failures=0
expect() {
	local want=$1 tag=$2
	"$root/scripts/check-release-tag.sh" "$tag" >/dev/null 2>&1
	local got=$?
	if { [ "$want" = pass ] && [ "$got" -ne 0 ]; } || { [ "$want" = fail ] && [ "$got" -eq 0 ]; }; then
		echo "FAIL: $tag: want $want, exit $got"
		failures=$((failures + 1))
	fi
}

for tag in v0.18.0 v1.0.0 v10.20.30 v1.0.0-rc.1 v1.0.0-alpha v1.0.0-0.3.7 v1.0.0-x.7.z.92 \
	v1.0.0-x-y-z.-- v1.0.0+20130313144700 v1.0.0-beta+exp.sha.5114f85 v1.0.0+21AF26D3---117B344092BD; do
	expect pass "$tag"
done
for tag in 0.18.0 V1.0.0 v1.0 v1 v01.0.0 v1.02.0 v1.0.00 v1.0.0-01 v1.0.0-rc..1 v1.0.0- v1.0.0+ \
	v1.0.0+a..b v1.0.0junk v1.0.0_rc v1.0.0-rc.1/x "v1.0.0 " v1.0.0-é ""; do
	expect fail "$tag"
done

if [ "$failures" -ne 0 ]; then
	exit 1
fi
echo "check-release-tag.sh regression tests passed"
