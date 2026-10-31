# Plan: Release workflow

Status: Done
Issue:
PR: https://github.com/fuseone-io/fusescope-mcp-server/pull/8

## Context

The fork removed SigNoz's release workflows, so FuseScope had no way to ship binaries or an
image. The first FuseScope release is v0.18.0 (upstream stopped at 0.17.0). A release job holds
`contents: write`, `packages: write` and an OIDC token, so a mutable action or an unreviewed
commit behind a tag could publish signed artifacts.

## Approach

A `v<semver>` tag runs `.github/workflows/release.yaml`, modelled on the FuseScope backend's
release workflow:

- `preflight` (read-only): the tag must be `v` + SemVer 2.0.0 and the tagged commit on `main`; `make ci` (the PR gate minus
  e2e, including `check-agpl`); `manifest.json` and `server.json` must carry the tag's version and
  `CHANGELOG.md` its section.
- `binaries`: GoReleaser builds linux/darwin/windows × amd64/arm64 archives with SPDX SBOMs, a
  cosign-signed `checksums.txt` and build provenance; release notes are the CHANGELOG section.
- `image`: multi-arch `ghcr.io/fuseone-io/fusescope-mcp-server:<version>`, `:latest` only for
  non-prereleases.
- `sign`: cosign signs the image recursively, attaches build provenance, and attests a syft SPDX
  SBOM per platform manifest; the SBOMs are also attached to the release.

- `approve-release`: a no-op job in the `release` environment; `binaries` and `image` wait for it.

## Files to Modify

- `.github/workflows/release.yaml` — new release workflow.
- `.github/workflows/*.y*ml` — actions pinned by commit SHA.
- `.github/dependabot.yml` — weekly grouped github-actions updates.
- `.goreleaser.yaml` — windows, zip, NOTICE, SBOMs, checksum signing, no generated changelog.
- `Dockerfile` — cross-compile for `TARGETARCH`, pinned alpine, labels on the final image.
- `scripts/release-notes.sh` — prints a version's CHANGELOG section.
- `scripts/check-release-tag.sh`, `scripts/check-release-tag_test.sh`, `Makefile`,
  `.github/workflows/ci.yaml` — SemVer tag check and its test target.
- `CONTRIBUTING.md`, `CHANGELOG.md`, `manifest.json`, `server.json` — release docs and 0.18.0.

## Key Decisions

### 2026-10-31 — Keyless cosign

- Decision: sign the image and `checksums.txt` with cosign keyless via GitHub OIDC.
- Rationale: matches the backend; no signing key to store or rotate.

### 2026-10-31 — SBOMs as attestations

- Decision: one SPDX SBOM per archive (GoReleaser) and per image platform (syft), the latter
  attested with `cosign attest --type spdxjson` on each platform manifest and attached to the
  release.
- Rationale: a pull resolves to a platform manifest, so that is where verifiers look.
- Alternatives rejected: one SBOM on the index digest, which mixes platforms.

### 2026-10-31 — SHA-pinned actions

- Decision: every action is pinned by commit SHA with its version as a comment; Dependabot bumps
  them weekly.
- Rationale: release jobs hold write and OIDC permissions; a moved tag must not change what runs.

### 2026-10-31 — Only commits on main publish

- Decision: `preflight` fails unless the tagged commit is an ancestor of `origin/main`, and reruns
  `make ci`. The trigger glob only approximates semver, so `scripts/check-release-tag.sh` checks
  the tag against the SemVer 2.0.0 regex (pre-release and build metadata allowed, leading zeros
  rejected).
- Rationale: any collaborator can push a tag; this keeps unreviewed commits out of signed
  releases.

### 2026-10-31 — `release` environment

- Decision: one `approve-release` job, with no permissions, runs in the `release` environment;
  `binaries` and `image` need it and `sign` needs both. No other job uses the environment.
- Rationale: required reviewers gate publishing on one human approval, given before anything is
  published, so no approval wait can leave a pushed image unsigned. The environment and the tag
  ruleset are repo settings, configured outside this PR.
- Alternatives rejected: the environment on each publishing job, which asks twice and can stall
  `sign` after the image is pushed.

## Reference Links

- [Backend release workflow](https://github.com/fuseone-io/fusescope/blob/main/.github/workflows/release.yaml)
- [GoReleaser signing](https://goreleaser.com/customization/sign/)
- [Dependabot for GitHub Actions](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/keeping-your-actions-up-to-date-with-dependabot)

## Verification

- `goreleaser check` and `goreleaser release --snapshot --clean --skip=publish,sign` (v2.19.0):
  six archives, six SBOMs, `checksums.txt`.
- `actionlint` with shellcheck on all workflows.
- `scripts/check-release-tag_test.sh` (also `make check-release-tag`, in `make ci` and the CI
  matrix): SemVer spec samples pass; leading zeros, missing parts, empty identifiers, `V`/no
  prefix, junk suffixes and non-ASCII fail.
- `docker buildx build --platform linux/amd64`: runs as uid 1001, carries the OCI and MCP
  registry labels, answers MCP `initialize` over stdio. The arm64 builder stage produces a
  linux/arm64 binary; the full arm64 image was not built locally (no QEMU).
- `make check-agpl`, `make check-repo-docs`; the PR gate is green.
- Unexercised until the first tag: cosign keyless signing, the provenance and SBOM attestations,
  the GHCR push, and the release environment gate.

## Outcome

Shipped the release workflow and 0.18.0 metadata. Repo settings, outside this PR:

- before the first tag, create the `release` environment with required reviewers, limited to `v*`
  tags (an undefined environment is created on first use without protection), and add a tag
  ruleset restricting who can create, update or delete `v*` tags;
- after the first release, make the `fusescope-mcp-server` GHCR package public.

MCP Registry publishing is not set up.
