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

## Rollout and Recovery
- Publish a new immutable version tag; ordinary semantic-tag releases contain
  the active CPU and Apple images only.
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

- Repository signal fingerprint: `877e66d244d32c9b5767465870b51ed52a23cf42e56c3c0b99bd519406969527`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
