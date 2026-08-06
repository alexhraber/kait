# Operations

## Operational Readiness
- On-call ownership belongs to the team operating the Buildkite queue and node
  pool.
- Dashboards should cover agent starts/exits, health, and job throughput.
- Rollback is an image-tag change and queue drain.
- Capacity guardrails are defined per accelerator class.

## Deployment Model
Kaite runs as a long-lived self-hosted agent or a one-shot Kubernetes Job.
Docker hosts and Kubernetes node pools select the image tag and device
resources; the Buildkite queue selects where work is dispatched.

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
- Publish a new immutable version tag.
- Start one canary agent per hardware queue.
- Verify health, agent registration, and a reference job.
- Roll the queue to the new image after the canary is healthy.
- Repoint the queue to the prior image tag if startup, health, or job behavior
  regresses.

<!-- decapod:codebase-attestation:start -->
## Codebase Attestation

- Repository signal fingerprint: `a053aa0c26e4414a6e960dc383ebef7f73fa571a2621bda5c1e51f4a6041d62e`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
