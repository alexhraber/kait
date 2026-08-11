# Operations

## Operational Readiness
- On-call ownership belongs to the team operating the Buildkite queue and node
  pool.
- Dashboards should cover agent starts/exits, health, and job throughput.
- Rollback is an image-tag change and queue drain.
- Capacity guardrails are defined per accelerator class; accelerator queues are
  inactive until their matching host pools are deliberately enabled.

## Deployment Model
Kaite runs as a long-lived self-hosted agent or a one-shot Kubernetes Job.
Docker hosts and Kubernetes node pools select the image tag and device
resources; the Buildkite queue selects where work is dispatched.

Semantic-tag publication uses the matching native GitHub host for each active
image: CPU is built on `ubuntu-24.04` for `linux/amd64`, and Apple is built on
`ubuntu-24.04-arm` for `linux/arm64`. Verification may use emulation only to
inspect the already-published Apple image; release publication does not rely on
QEMU to build an active image.

Release promotion is a two-stage workflow. A push to `main` runs Release Please,
which creates or updates the release PR for ordinary merged changes. Merging
that release PR creates the semantic tag and GitHub release. Because a tag
created with `GITHUB_TOKEN` does not start a second workflow, the promotion job
explicitly dispatches `release-images.yml` against the created tag. That image
workflow then builds, smoke-tests, and publishes the active images; it never
resolves the release from a moving `main` ref.

The image dependency contract is layered and installed in a fixed order. Every
variant contains the pinned Buildkite agent, Go supervisor, and the selected
hardware manifest. `*-slim` stops after the compact Python foundation; `*-full`
also installs portable data/science foundations, the Hugging Face training
stack, experiment/distributed orchestration, and serving interfaces. CPU and
Apple are the active portable contracts; vendor-specific native extensions
remain operator extras until their inactive host paths are enabled and proven.
Release Please requires the repository permission for `GITHUB_TOKEN` to create
pull requests, in addition to the workflow scopes.

## Container Launch Wrapper
`deploy/docker/run.sh` places runtime flags and environment bindings before the
image reference, then places `KAITE_CONTAINER_COMMAND` after the image. This
keeps agent launches and one-shot commands such as `smoke` valid for Docker
and Kubernetes executor workflows.

## Service Level Objectives
| SLI | Initial target | Measurement |
|---|---|---|
| Agent registration | 99% of valid starts | Buildkite agent state |
| Runtime availability | 99.9% while node is healthy | /healthz and process state |
| Startup diagnosis | under 60 seconds | structured startup event |

## Health Checks
- Liveness: GET /healthz on port 9090.
- Readiness: GET /readyz returns 200 after the Buildkite child starts.
- Dependency health: Buildkite agent logs and child exit status.
- Hardware check: kaite doctor and kaite-hardware.

## Monitoring
| Signal | Source | Backend selection |
|---|---|---|
| Agent starts/exits/running | Kaite /metrics | Prometheus scrape or Splunk collector |
| Runtime counters | DogStatsD | Datadog Agent |
| Lifecycle and job logs | stdout/stderr | Buildkite, Datadog Agent, or OTel collector |

## Logging
Kaite emits structured JSON lifecycle logs to stderr and preserves the
Buildkite agent's stdout/stderr. Datadog and Splunk deployments should collect
container logs with their node agent or OpenTelemetry Collector.

## Secrets
| Secret | Source | Consumer |
|---|---|---|
| Buildkite agent token | Docker secret or Kubernetes Secret | Kaite entrypoint |
| Datadog credentials | Datadog Agent or DaemonSet Secret | Datadog Agent |
| Splunk access token | OpenTelemetry Collector Secret | Splunk Collector |

## Incident Response
- Detection: alert on agent exits, failed readiness, queue age, or hardware
  discovery failures.
- Triage: inspect structured Kaite logs, Buildkite agent status, and node
  device-plugin health.
- Mitigation: drain the queue and roll back to the previous image tag.
- Communication: record the affected hardware queue and observability mode.
- Post-incident: add a reference job or validation case for the failure mode.

## Release image promotion

GitHub Releases are created by Release Please on merge of its release PR. Image
publication is a separate `workflow_dispatch` of `release-images.yml` against
the immutable tag, because tags created with `GITHUB_TOKEN` do not re-trigger
`on: push: tags` workflows. The Release Please job therefore needs
`actions: write` so it can dispatch image builds; without that permission the
tag and release still appear but GHCR stays empty (observed for v0.2.0).

Release Please only opens a release PR when commits since the last tag include
user-facing conventional types (`feat`, `fix`, `perf`). Squash titles of
`chore:` / `docs:` / `ci:` produce a successful Release PR workflow run with
**no** release PR and a skipped image dispatch (observed after #13).

After active images pass verify, the **Annotate GitHub release with image tags**
job appends a Container images section (pull table + example) to the existing
release body. It uses `gh release edit --notes-file` (or `create` only if the
release is missing). It never uses `--generate-notes`. Re-runs are idempotent:
if the section is already present, the job exits 0. The same job runs for both
tag-push and `workflow_dispatch` inputs.

### Public package pulls

GHCR package visibility is **not** inherited from repository visibility. For
anonymous pulls of `ghcr.io/<owner>/kaite:…` on a public project, set the
container package to Public under package settings. Document this for operators
in README/CONTRIBUTING; automation via Packages REST visibility endpoints may
return 404 depending on token scopes.

## Rollout and Recovery
- Publish immutable `<tag>-<hardware>-<variant>` tags; ordinary semantic-tag
  releases contain active `cpu-slim`, `cpu-full`, `apple-slim`, and
  `apple-full` images plus compatibility aliases for the slim images.
- Start one canary agent per hardware queue.
- Verify health, agent registration, and a reference job.
- Roll the queue to the new image after the canary is healthy.
- Repoint the queue to the prior image tag if startup, health, or job behavior
  regresses.

To activate NVIDIA, AMD, or Intel delivery, manually enable the accelerator
workflow path only after registering the corresponding `kaite-*` runner and
vendor device runtime.

<!-- decapod:capability-overlay:background-processing:start -->

## Background Processing Operations Overlay

### Queue Visibility
- Queue depth, processing rate, and latency MUST be monitored
- Dead letter queue MUST be visible and alerted
- Worker health and processing rate metrics required

### Shutdown Behavior
- Graceful shutdown: stop accepting new work, finish current job
- Drain behavior and timeout MUST be selected for the deployment
- Termination and requeue behavior MUST be selected and proven for the deployment

### Worker Health
- Worker liveness and readiness probes
- Queue depth alerts for backpressure detection
- Processing latency percentiles (p50, p95, p99)
<!-- decapod:capability-overlay:background-processing:end -->

<!-- decapod:capability-overlay:persistent-state:start -->

## Persistent State Operations Overlay

### Backup & Recovery
- Backup scope, schedule, retention, and restore evidence MUST be selected for the project
- Recovery point objectives MUST be explicit project decisions, not assumed values
- Recovery time objectives MUST be explicit project decisions, not assumed values
- Restore verification cadence MUST be recorded with the operational proof plan

### Migration Operations
- All schema changes via migration files
- Migration rollback procedures documented
- Zero-downtime migration strategy for production
- Migration health checks and rollback triggers
<!-- decapod:capability-overlay:persistent-state:end -->

<!-- decapod:codebase-attestation:start -->

## Codebase Attestation

- Repository signal fingerprint: `bbcc61486b1c6c224fa8e4057cb8283714a4d1e869c3879e4aa82178d8ca397b`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
