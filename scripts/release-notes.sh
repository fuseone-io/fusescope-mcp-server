#!/usr/bin/env bash
# Prints the CHANGELOG.md section of the given version (without the leading v);
# fails when the section is missing or empty.
set -euo pipefail

version="${1:?usage: release-notes.sh VERSION}"
changelog="${CHANGELOG:-CHANGELOG.md}"

notes=$(awk -v head="## [${version}]" '
	/^## \[/ { p = index($0, head) == 1; next }
	p
' "$changelog" | sed -e '/./,$!d')

if [ -z "${notes//[[:space:]]/}" ]; then
	echo "${GITHUB_ACTIONS:+::error::}${changelog} has no notes for ${version}." >&2
	exit 1
fi
printf '%s\n' "$notes"
