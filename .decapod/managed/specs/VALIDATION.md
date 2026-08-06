# Validation

## Validation Philosophy
Validation is a release gate, not documentation theater.

## Validation Harness
The local proof harness is Go unit tests plus deterministic Docker Bake graph
validation. Image builds are executed on tagged releases and can be run
locally per hardware target.

- Automated tests: GOCACHE=/tmp/kaite-gocache go test ./...
- Static checks: GOCACHE=/tmp/kaite-gocache go vet ./...
- Formatting: gofmt -w cmd/kaite/*.go
- Image contract: docker buildx bake --print
- Governance: decapod validate

## Proof Surfaces
- decapod validate
- GOCACHE=/tmp/kaite-gocache go test ./...
- GOCACHE=/tmp/kaite-gocache go vet ./...
- docker buildx bake --print
- Kubernetes YAML review with a cluster-specific validation tool before apply

## Promotion Gates
| Gate | Command | Evidence |
|---|---|---|
| Architecture and interface drift | decapod validate | Gate output |
| Go tests pass | GOCACHE=/tmp/kaite-gocache go test ./... | CI and local logs |
| Bake graph valid | docker buildx bake --print | CI output |
| Docs/specs current | README and living-spec diff | PR diff |
| Security scan | image scanner on tagged builds | Scanner reports |

## Evidence Artifacts
| Artifact | Path or source | Required for |
|---|---|---|
| Go test output | CI log and local command | Every change |
| Bake graph | GitHub Actions plan job | Image changes |
| Runtime smoke output | Local or CI log | Entrypoint changes |
| Decapod receipt | .decapod/governance/validation.json | Publishable commit |

## Regression Guardrails
- A missing Buildkite token must continue to fail closed.
- Child exit codes must remain observable and non-zero when the job fails.
- Hardware tags must match the image target and KAITE_HARDWARE value.
- O11y mode changes must not require vendor credentials in the Kaite image.

## CI Flow
1. Every pull request renders the Bake graph.
2. Every pull request runs the Go tests through the repository validation job.
3. Version tags publish all hardware image targets to the configured registry.
4. Image scanning and signing remain promotion requirements before production use.

## Failure Recovery
- A Go failure is fixed and rerun with the same cache override.
- A stale spec is refreshed with Decapod, then its authored prose is reviewed.
- A failed image target is isolated by target name and base-image override.
- A missing vendor collector does not block local structured logs or Prometheus.

<!-- decapod:codebase-attestation:start -->
## Codebase Attestation

- Repository signal fingerprint: `a053aa0c26e4414a6e960dc383ebef7f73fa571a2621bda5c1e51f4a6041d62e`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
