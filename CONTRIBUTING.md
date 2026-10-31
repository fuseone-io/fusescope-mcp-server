# Contributing to SigNoz MCP Server

Thanks for contributing to SigNoz MCP Server.

## Development workflow

1. Fork the repository and create a feature branch.
2. Make your changes with tests where applicable.
3. Run local checks before opening a PR.
4. Open a pull request with a clear summary and validation steps.

## Releases

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yaml`. The tagged commit must be on `main`;
set the version in `manifest.json` and `server.json` and add its `CHANGELOG.md` section first. The
workflow reruns `make ci` and fails otherwise, and its publishing jobs wait for approval in the
`release` environment.
It publishes:

- GoReleaser binaries for Linux, macOS and Windows (amd64, arm64) with SPDX SBOMs, a
  `checksums.txt` signed with cosign, and build provenance, on a GitHub release whose notes are
  the CHANGELOG section.
- `ghcr.io/fuseone-io/fusescope-mcp-server:X.Y.Z` (and `:latest` for non-prereleases) for
  linux/amd64 and linux/arm64, signed with cosign, with build provenance and an SPDX SBOM
  attestation per platform; the SBOMs are also attached to the release.

Verify an image with:

```sh
cosign verify ghcr.io/fuseone-io/fusescope-mcp-server:X.Y.Z \
  --certificate-identity-regexp '^https://github.com/fuseone-io/fusescope-mcp-server/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
gh attestation verify oci://ghcr.io/fuseone-io/fusescope-mcp-server:X.Y.Z -R fuseone-io/fusescope-mcp-server
```

Publishing to the MCP Registry is not set up yet.

## Required sync for MCP changes

If your PR adds, removes, or renames MCP tools/resources/config behavior, update docs and metadata in the same PR:

- `README.md` (tool list and parameter references)
- `manifest.json` (`tools` names/descriptions)
- Relevant files in `docs/` when user-facing behavior changes

This prevents drift between implementation, manifest metadata, and public docs.

## Suggested validation

Run `make ci` before pushing. It runs everything the PR gate runs except the live e2e suite, and each step is also its own target (see the `CI` section of the `Makefile`). For a quicker loop, run what is relevant for your change:

```bash
go test ./...
```

For documentation-only changes, at minimum run `make check-repo-docs`, which validates plans and flags stale repo paths in docs. Mention what was validated in the PR.

## Testing across external contracts

This server depends on external parties — it consumes the SigNoz backend / query-builder (QB) API (upstream) and produces tool outputs that MCP clients and custom agents and clients consume (downstream). Fixture-based unit tests only prove our code matches our *assumption* of those contracts; they do not catch the contract drifting out from under us (a renamed field, a changed QB response envelope, a new output shape). When you parse an upstream response or shape a tool output:

- **Pin the contract, and test against reality where you can.** Beyond fixture unit tests, add a periodic/integration test against a live instance (or a recorded real response) so upstream drift fails a test, not a user.
- **When tests can't catch it, observability must.** If a break only manifests against real data, add a metric or WARN log that fires when the contract appears violated (e.g. a passthrough that found rows but could not locate the expected field), so silent degradation is detectable in production.
- **Fail open, but never fail silent.** Pair every fail-open cross-boundary parse with a detectable signal.

## Pull request checklist

- [ ] Code/tests updated as needed
- [ ] README/docs updated for user-facing changes
- [ ] `manifest.json` updated for MCP tool metadata changes
- [ ] Validation commands and results included in PR description
