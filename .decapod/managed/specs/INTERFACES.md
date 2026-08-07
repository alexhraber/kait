# Interfaces

## Contract Principles
- Prefer explicit environment and file contracts over implicit behavior.
- Every failure path maps to a typed, documented runtime error.
- Secrets are supplied by the executor and never baked into an image.

## Runtime Interfaces
Kaite has an environment-driven container interface rather than an HTTP
control-plane API. The runtime exposes health and Prometheus-compatible metrics.

| Interface | Inputs | Outputs | Errors | Ownership |
|---|---|---|---|---|
| Container entrypoint | KAITE_*, BUILDKITE_*, optional mounted token | Buildkite agent process | Exit 2 for invalid config; child exit otherwise | Orchestrator restarts container |
| GET /healthz | HTTP request | 200 while supervisor is alive | 5xx if server unavailable | Kubernetes liveness |
| GET /readyz | HTTP request | 200 after child starts | 503 before child starts | Kubernetes readiness |
| GET /metrics | HTTP request | Prometheus text exposition | 5xx if server unavailable | Prometheus scrape |

## Buildkite Contract
The Buildkite agent is the event consumer. Buildkite owns polling, job
ordering, checkout, command execution, artifact upload, and job status.

| Input | Required | Semantics |
|---|---|---|
| BUILDKITE_AGENT_TOKEN | Agent mode | Cluster token passed only to the child process |
| BUILDKITE_AGENT_TOKEN_FILE | Agent mode alternative | Uses Buildkite file token syntax for a mounted secret |
| BUILDKITE_AGENT_TAGS | Optional | Explicit tags; otherwise Kaite adds hardware and o11y tags |
| BUILDKITE_AGENT_NAME | Optional | Forwarded to Buildkite agent start |
| BUILDKITE_AGENT_CONFIG | Optional | Forwarded config path |
| BUILDKITE_KUBERNETES_EXEC | Optional | Enables Buildkite Kubernetes log/exit transport |

## Inbound Contracts
| Input | Values | Default |
|---|---|---|
| KAITE_HARDWARE | cpu, apple, nvidia, amd, intel | cpu |
| KAITE_O11Y | none, prometheus, datadog, splunk | none |
| KAITE_RUN_MODE | agent, command | agent |
| KAITE_COMMAND | shell command | unset; required in command mode |
| KAITE_METRICS_ADDR | TCP bind address | :9090 when o11y is enabled |

## Outbound Dependencies
| Dependency | Purpose | Configuration | Failure behavior |
|---|---|---|---|
| Buildkite Agent API | Register agent and receive jobs | token, endpoint, tags | Startup failure exits non-zero |
| DogStatsD | Datadog runtime counters | KAITE_DD_AGENT_HOST and KAITE_DD_DOGSTATSD_PORT | UDP metrics are best-effort |
| OpenTelemetry Collector | Splunk metrics/log pipeline | OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_SERVICE_NAME | Local logs and metrics remain available |

## Image Platform Contract

| Target | Base and platform | Status | Hardware behavior |
|---|---|---|---|
| cpu | Ubuntu 24.04; linux/amd64 and linux/arm64 | active | Common CPU toolchain |
| apple | Ubuntu 24.04; linux/arm64 | active | Apple Silicon CPU execution; no Metal device passthrough |
| nvidia | CUDA 12.6.3; linux/amd64 | inactive | NVIDIA runtime and CUDA PyTorch wheels |
| amd | ROCm 6.2.4; linux/amd64 | inactive | AMD runtime and ROCm PyTorch wheels |
| intel | oneAPI Base Toolkit 2025.0.1 / Ubuntu 22.04; linux/amd64 | inactive | Python 3.11, Intel XPU runtime, and extension wheels |

Inactive accelerator targets are available for deliberate local or manual
workflow use. Their matching `kaite-nvidia`, `kaite-amd`, and `kaite-intel`
runner labels do not participate in ordinary CI or semantic-tag releases.

## Data Ownership
- Buildkite is the source of truth for job state, logs, artifacts, and metadata.
- The container owns only process counters and ephemeral local build state.
- Checkpoints and datasets are operator-provided mounts or object-store clients
  configured by the Buildkite workload, not hidden in the supervisor.

## Error Taxonomy
- configuration_failed: invalid hardware, o11y, run mode, or missing token.
- agent_binary_missing: configured Buildkite binary is not present.
- metrics_server_failed: configured metrics address cannot bind.
- process_start_failed: child process cannot be started.
- Child exit status: propagated to the container for orchestrator retry policy.

## Failure Semantics
| Failure class | Retry/backoff | Client contract | Observability |
|---|---|---|---|
| Invalid configuration | No retry until corrected | Exit 2 with a typed log event | stderr and startup event |
| Agent startup failure | Orchestrator policy | Non-zero container exit | stderr and child status |
| Metrics listener failure | No retry inside supervisor | Non-zero container exit | stderr |
| Buildkite job failure | Buildkite policy | Child exit status | Buildkite job log and metric |

## Interface Versioning
- Version strategy: image tags and the runtime version constant.
- Backward compatibility: existing environment names remain stable within a
  major image line; new options are additive.
- Deprecation: announce in README and living specs before removing an input.

<!-- decapod:capability-overlay:public-api:start -->

## Public API Capability Overlay

### API Contract Requirements
- All public endpoints MUST define explicit request/response schemas
- Versioning strategy MUST be documented (URL path or header-based)
- All public endpoints MUST implement idempotency for mutating operations
- Rate limiting and pagination MUST be implemented for list endpoints

### Compatibility Guarantees
- Backward-compatible changes ONLY within a version
- Breaking changes require new version (v1, v2, etc.)
- Deprecation and removal policy MUST be selected for this project and proven against its consumers

### Security Requirements
- All public endpoints MUST implement authentication
- Abuse-control enforcement point MUST be a documented project decision
- Input validation MUST reject malformed requests with typed errors
<!-- decapod:capability-overlay:public-api:end -->

<!-- decapod:codebase-attestation:start -->
## Codebase Attestation

- Repository signal fingerprint: `877e66d244d32c9b5767465870b51ed52a23cf42e56c3c0b99bd519406969527`
- Significant implementation surfaces: `.github/` (3 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
