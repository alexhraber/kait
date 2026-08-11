# Validation

## Validation Philosophy
Validation is a release gate, not documentation theater.

## Validation Harness
Local proof is the Makefile-backed Go harness plus the Docker Bake graph.
Image builds and `kaite smoke` run on tagged releases (and can be run locally
per hardware target).

- Automated tests: `make test` (or `GOCACHE=… go test ./...`)
- Static checks: `make vet`
- Formatting: `make fmt`
- Local binary: `make build` → `bin/kaite`
- Image contract: `make build-plan` / `docker buildx bake --print`
- Dependency contract: review `requirements/*.txt` install order; native CPU
  and Apple `*-slim`/`*-full` image builds with `kaite smoke`
- Governance: `decapod validate`

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
| Active dependency layers resolve | native CPU/Apple slim/full Docker builds | Image build logs and variant-aware `kaite smoke` |
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
- Hardware and variant tags must match the image target, KAITE_HARDWARE, and
  KAITE_VARIANT values.
- O11y mode changes must not require vendor credentials in the Kaite image.

## CI Flow
1. Every pull request renders the Bake graph.
2. Every pull request runs the Go tests through the repository validation job.
3. A push to `main` creates or updates the Release Please PR.
4. Merging the Release Please PR creates the semantic tag and GitHub release;
   the release workflow explicitly dispatches image publication for that tag.
5. The tagged image workflow builds active `*-slim` and `*-full` targets on
   native hosts, runs the container smoke contract for each variant, and
   publishes versioned and stable GHCR tags.
6. After verify succeeds, the workflow annotates the existing GitHub Release
   with GHCR pull commands (`gh release edit --notes-file`, never
   `--generate-notes`). Re-runs are idempotent when the section already exists.
7. Image scanning and signing remain promotion requirements before production
   use. Operator docs (README, SECURITY.md) state that GHCR package Public
   visibility is required for anonymous pulls and is independent of repo
   visibility.

The release graph is proven by checking both workflow definitions: Release
Please owns the main-to-release-PR and release-PR-to-tag transitions, while
`release-images.yml` owns the immutable tagged build, smoke, GHCR boundary, and
release annotation. The explicit dispatch is required because GitHub does not
recursively trigger workflows from resources created with `GITHUB_TOKEN`.

Supervisor version proof: `cmd/kaite` `version` constant and unit tests must
match the released package version so doctor/smoke/Prometheus labels are not
stale relative to the GitHub/GHCR tag.

The dependency proof must show that the hardware manifest is installed before
the shared AI/ML layers, that the active CPU and Apple manifests resolve Linux
`amd64` and `arm64` wheels, that slim and full image tags are distinct, and that
the variant-aware image smoke command imports the advertised full toolchain.
Accelerator-only packages are not counted as portable proof until their
matching inactive host jobs are deliberately enabled.

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

## Codebase Attestation

- Repository signal fingerprint: `c26fd3e39755202a0870c2ba78cc44f5f53bbdae45cc614ec6a8311547f5b2db`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->

## Codebase Attestation

- Repository signal fingerprint: `c26fd3e39755202a0870c2ba78cc44f5f53bbdae45cc614ec6a8311547f5b2db`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`

## Codebase Attestation

- Repository signal fingerprint: `8e262b3c9e364a0c07874dde13dd5a03128cc9dba98ed8af7e7a7f4b5ee506ee`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
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

- Repository signal fingerprint: `24454fd28087d866086db267cb3430825d2ec92494981796bfd69e715a9cee16`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
