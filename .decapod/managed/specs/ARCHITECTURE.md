# Architecture

## Direction
Infra-oriented container architecture with explicit runtime boundaries and
proof-backed delivery invariants.

## What This Project Is
Kaite is the open-source AI runtime for self-hosted Buildkite. It packages the
agent, standard Python toolchain, hardware runtime, process supervision, and
observability adapters into one deployable image.

## Runtime Boundary
Kaite is a Go supervisor packaged into hardware-specific Linux images. The
supervisor owns process lifecycle, input validation, health endpoints, metrics,
structured logs, and Buildkite agent invocation. Buildkite remains the system
that dispatches and records jobs.

## Principles
- Simplicity: the Buildkite agent remains the job executor.
- Modularity: hardware images vary by base image and build-time packages, not
  by the Buildkite or workload contract.
- Reliability: PID 1 signal handling, health/readiness endpoints, and child
  exit propagation are verified in Go.
- Collector friendliness: vendor credentials live in Datadog/Splunk agents or
  collectors; Kaite exposes stable signals and never owns those secrets.

## Current Facts
- Runtime/languages: Go supervisor, Ubuntu-based Docker images, Python venv.
- Hardware surfaces: Ubuntu CPU, Apple Silicon Linux arm64 CPU, CUDA/NVIDIA,
  ROCm/AMD, and Intel oneAPI.
- Toolchain: NumPy, pandas, scikit-learn, Jupyter, pytest, requests, and PyYAML.
- Product type: self-hosted Buildkite AI runtime.

## Architecture Map
- cmd/kaite/: runtime supervisor and contract tests.
- Dockerfile and docker-bake.hcl: common image implementation and matrix.
- deploy/: Docker and Kubernetes execution templates.
- requirements/: portable Python defaults.
- examples/: Buildkite targeting examples.

The image matrix uses Ubuntu/glibc for every target. The CPU image publishes
linux/amd64 and linux/arm64 variants; the Apple target is an explicit
linux/arm64 CPU alias. NVIDIA, AMD, and Intel targets publish linux/amd64
because their pinned vendor bases and framework wheels are amd64 contracts.
musl is not an equivalent option for these images: it would require separate
Python wheels, vendor libraries, and validation.

Image delivery is host-matrixed. CI builds and runs `kaite smoke` on native
CPU, arm64, NVIDIA, AMD, and Intel hosts; the accelerator runner labels are
explicit inputs to the workflow so a successful build cannot substitute a
device-free boot check for real accelerator validation.

## Strongest Existing Primitives
- Go standard-library supervisor with no runtime library dependencies.
- Buildkite agent start command and its standard environment contract.
- Docker Bake target inheritance for hardware image variants.
- Kubernetes Secret, resource, and pod environment inputs.

## Data Flows
1. Docker or Kubernetes injects environment and device resources.
2. Kaite validates KAITE_HARDWARE, KAITE_O11Y, and Buildkite credentials.
3. Kaite starts buildkite-agent start, which polls Buildkite and executes jobs.
4. Metrics are scraped by Prometheus or forwarded to DogStatsD/OTel
   collectors; logs remain structured on stdout/stderr.

```text
Docker/Kubernetes -> Kaite supervisor -> Buildkite agent -> Buildkite jobs
                         |                 |
                         +-> /metrics     +-> stdout/stderr
                                           +-> DogStatsD/OTel collector
```

## Topology
```text
Docker/Kubernetes -> Kaite -> Buildkite Agent API
        |             |              |
   device nodes   /metrics       Buildkite jobs
        |             |
   GPU runtime   collectors
```

## Store Boundaries
```mermaid
flowchart LR
  E[Executor] --> K[Kaite Container]
  K --> B[Buildkite Agent API]
  K --> M[Metrics and Collectors]
  K --> J[Buildkite Job Workspace]
```

## Happy Path Sequence
Executor injects token and hardware settings -> Kaite validates inputs ->
agent registers with Buildkite -> Buildkite dispatches a job -> job runs in
the selected image -> logs, artifacts, and metrics leave through their
configured systems.

## Error Path
Invalid runtime input fails closed with exit 2. An unavailable agent binary,
metrics listener, or child process returns a non-zero status and is visible in
structured logs. The orchestrator owns restart and rollback.

## Execution Path
- Ingress parse and validation: load KAITE_* and BUILDKITE_* inputs.
- Policy checks: require a token in agent mode and validate the o11y enum.
- Core execution: start and supervise the Buildkite child process.
- Verification: expose health/metrics and propagate the child exit status.

## Concurrency and Runtime Model
- Execution model: one supervisor process and one Buildkite child process.
- Isolation boundaries: container, Buildkite build path, and operator-provided
  device mounts.
- Backpressure strategy: Buildkite queue and agent concurrency settings.
- Shared state synchronization: atomic runtime counters; no shared database.

## Deployment Topology
- Runtime units: one long-lived agent or one-shot Kubernetes Job per hardware
  pool.
- Region/zone model: selected by the operator's Docker host or Kubernetes node
  pool.
- Rollout: publish immutable image tags and roll queues forward.
- Rollback: move the queue back to the prior tag after startup, health, or job
  regressions.

## Data and Contracts
- Inbound contracts: environment variables, mounted token files, and device
  resources.
- Outbound dependencies: Buildkite Agent API, optional DogStatsD, and OTLP
  collectors.
- Ownership: Buildkite owns job state, while the host owns logs, caches,
  checkpoints, and artifacts mounted into the job.
- Evolution: environment names and image tags are compatibility surfaces; any
  change requires README/spec updates and tests.

## ADR Register
| ADR | Title | Status | Rationale | Date |
|---|---|---|---|---|
| ADR-001 | Agent-in-image topology | Accepted | Keep Buildkite dispatch and the AI toolchain in one hardware-selected image | 2026-08-06 |

## Delivery Plan
- Slice 1: supervisor, image matrix, launch templates, and o11y selector.
- Slice 2: framework-specific validated images and reference workflows.
- Slice 3: checkpoint/artifact backends, autoscaling, and collector packaging.

## Risks and Mitigations
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accelerator base tag disappears | Medium | High | Pin tags and allow BASE_IMAGE override in Bake |
| Vendor collector is absent | Medium | Medium | Keep stdout and Prometheus endpoint usable in every mode |
| Build secrets leak into logs | Low | High | Secret-file support and no token logging |

<!-- decapod:codebase-attestation:start -->
## Codebase Attestation

- Repository signal fingerprint: `a053aa0c26e4414a6e960dc383ebef7f73fa571a2621bda5c1e51f4a6041d62e`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
