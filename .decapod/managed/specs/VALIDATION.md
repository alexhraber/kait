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
## Codebase Attestation

- Repository signal fingerprint: `dff6fa2af071e116fb753ac55f5d10dd10bba726a15e2018e567c80dba7ffd30`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->

<!-- decapod:capability-overlay:background-processing:start -->

## Background Processing Validation Overlay

### Duplicate Delivery Tests
- Same message delivered multiple times MUST produce same result
- Idempotency key verification
- Verify the declared delivery guarantee; do not claim exactly-once behavior without proof

### Retry Tests
- Configured retry/backoff policy verified
- Configured retry bound or unbounded policy verified
- Poison-work handling verified when the project declares it

### Shutdown Tests
- Graceful drain on signal
- In-flight job completion or safe requeue
- No data loss on forced termination
<!-- decapod:capability-overlay:background-processing:end -->

<!-- decapod:capability-overlay:persistent-state:start -->

## Persistent State Validation Overlay

### Migration Proof Command
- Configure `repo.migration_validation.command` and its arguments as the executable migration proof; file presence is not proof
- The configured command MUST define its working directory, timeout, expected exit code, and evidence output

### Migration Tests
- All migrations MUST have integration tests
- Rollback procedures MUST be tested
- Data integrity checks post-migration

### Persistence Integration Tests
- Repository abstraction tested against real database
- Transaction boundary tests
- Concurrency conflict tests
- Data integrity validation after recovery
<!-- decapod:capability-overlay:persistent-state:end -->

<!-- decapod:capability-overlay:public-api:start -->

## Public API Validation Overlay

### Contract Tests
- All public endpoints MUST have contract tests
- Request/response schema validation on every request
- Compatibility regression tests for each version

### Security Tests
- Authentication bypass tests
- Malformed input handling tests
- Rate limit enforcement tests
- Token expiry/revocation tests
<!-- decapod:capability-overlay:public-api:end -->

<!-- decapod:codebase-attestation:start -->
## Codebase Attestation

- Repository signal fingerprint: `877e66d244d32c9b5767465870b51ed52a23cf42e56c3c0b99bd519406969527`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
