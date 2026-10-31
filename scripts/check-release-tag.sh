#!/usr/bin/env bash
# Fails unless the tag is v followed by a SemVer 2.0.0 version (https://semver.org regex, ERE form).
set -euo pipefail
export LC_ALL=C

tag="${1-}"
num='(0|[1-9][0-9]*)'
pre_id='(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)'
build_id='[0-9a-zA-Z-]+'
semver="^v${num}\.${num}\.${num}(-${pre_id}(\.${pre_id})*)?(\+${build_id}(\.${build_id})*)?\$"

if ! [[ $tag =~ $semver ]]; then
	echo "${GITHUB_ACTIONS:+::error::}Tag '${tag}' is not v<SemVer 2.0.0> (e.g. v1.2.3, v1.2.3-rc.1)." >&2
	exit 1
fi
