# Project Specs

Canonical path: `.decapod/managed/specs/`.
These files are the project-local contract for humans and agents.

## Snapshot
- Project: **kait**
- Outcome: Self-hosted Buildkite agents with a portable AI/ML runtime image.
- Primary language: Go (stdlib supervisor)
- Surfaces: Docker Bake image matrix, Docker launcher, Kubernetes Job,
  Python requirement layers, Release Please + GHCR CI

## How to use this folder
- [INTENT.md](./INTENT.md): product outcome, non-goals, acceptance criteria.
- [ARCHITECTURE.md](./ARCHITECTURE.md): topology, supervisor layout, ADRs.
- [INTERFACES.md](./INTERFACES.md): env/CLI/metrics contracts and failures.
- [VALIDATION.md](./VALIDATION.md): proof commands and quality gates.
- [SEMANTICS.md](./SEMANTICS.md): process lifecycle and invariants.
- [OPERATIONS.md](./OPERATIONS.md): SLOs, monitoring, rollout.
- [SECURITY.md](./SECURITY.md): trust boundaries and secret handling.

## Canonical `.decapod/` Layout
- `.decapod/data/`: control-plane state (SQLite + ledgers).
- `.decapod/managed/Dockerfile.decapod`: Decapod execution image.
- `.decapod/managed/specs/`: living project specs (this folder).
- Root `Dockerfile`: product application image (what users deploy).
- `.decapod/governance/`: plan, trajectory, validation receipts.
- `.decapod/workspaces/`: isolated todo-scoped git worktrees.

## Contributor checklist
- [x] Intent and acceptance criteria describe the shipping product.
- [x] Architecture map matches `cmd/kait/` file layout and image matrix.
- [x] Interfaces document env vars, metrics endpoints, and image tags.
- [x] Validation gates cover `go test`, `go vet`, bake graph, and CI smoke.
- [ ] Accelerator paths stay inactive until real host runners exist.
- [ ] Proof commands recorded on publishable commits.

<!-- decapod:codebase-attestation:start -->

## Codebase Attestation

- Repository signal fingerprint: `c70f67f5dedfa45dbe5c2c90971c52f165d80c3755cabac1982850ba12279dc0`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
