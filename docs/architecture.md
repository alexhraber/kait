# Architecture

Kaite is a **thin Go supervisor** packaged into **hardware-specific Linux
images**. Buildkite remains the orchestrator; Kaite owns process lifecycle,
input validation, health/metrics, and a few diagnostic subcommands.

If you only need to run an agent, start with the [root README](../README.md).
This document is the deeper layout for contributors and operators who want to
know how the pieces fit.

## Big picture

```text
Docker / Kubernetes
        │
        ▼
┌───────────────────┐     start / signals      ┌────────────────────┐
│  kaite supervisor │ ───────────────────────► │  buildkite-agent   │
│  (static Go bin)  │ ◄─────────────────────── │  (pinned in image) │
└─────────┬─────────┘     exit status          └─────────┬──────────┘
          │                                              │
          │  /healthz /readyz /metrics                   │  job stdout/stderr
          │  structured JSON logs (stderr)               ▼
          ▼                                    Buildkite cloud (dispatch)
     Prometheus / DogStatsD / OTel collectors
```

Design choices that stay fixed:

| Choice | Why |
| --- | --- |
| Agent-in-image | One tag selects both the Buildkite worker and the AI toolchain |
| Stdlib-only supervisor | No runtime deps to pin or CVEs to chase in the entrypoint |
| Ubuntu/glibc bases | Shared wheel and vendor-runtime contract; musl would be a separate product |
| Collector-friendly o11y | Vendor secrets stay on Datadog/Splunk agents, not in the image |
| Slim vs full variants | Same supervisor; package footprint is a build-time matrix, not a fork |

## Repository map

| Path | Role |
| --- | --- |
| [`cmd/kaite/`](../cmd/kaite/) | Go supervisor and unit tests |
| [`Dockerfile`](../Dockerfile) | Multi-stage image: build supervisor → runtime + agent + venv |
| [`docker-bake.hcl`](../docker-bake.hcl) | Image matrix (hardware × slim/full) |
| [`requirements/`](../requirements/) | Layered Python manifests |
| [`deploy/docker/`](../deploy/docker/) | Local launcher and smoke wrapper |
| [`deploy/kubernetes/`](../deploy/kubernetes/) | One-shot Buildkite agent Job |
| [`examples/`](../examples/) | Pipeline agent-targeting snippets |
| [`.github/workflows/`](../.github/workflows/) | Image CI, Release Please, GHCR publish |

## Supervisor (`cmd/kaite`)

The binary is split by concern so the entrypoint stays readable:

| File | Responsibility |
| --- | --- |
| `main.go` | CLI dispatch (`doctor` / `smoke` / `hardware`) and process bootstrap |
| `config.go` | Env loading and validation (`KAITE_*`, Buildkite token sources) |
| `run.go` | Child process start, SIGTERM, exit-code propagation, agent args |
| `metrics.go` | HTTP `/healthz`, `/readyz`, `/metrics`; DogStatsD client |
| `doctor.go` | Framework/device smoke checks used by CI and operators |
| `hardware.go` | Vendor environment setup (e.g. Intel oneAPI `setvars`) |
| `log.go` | Structured JSON events on stderr |
| `version.go` | Supervisor identity (kept in lockstep with the package release) |

### Process model

1. Optional hardware environment setup (no-op on CPU/Apple).
2. Subcommand path returns immediately (`doctor`, `smoke`, `hardware`).
3. Otherwise load config (exit **2** on invalid input).
4. Start metrics server when `KAITE_METRICS_ADDR` is set or o11y ≠ `none`.
5. Start either `buildkite-agent start` or `sh -lc "$KAITE_COMMAND"`.
6. On SIGINT/SIGTERM, forward SIGTERM to the child and wait.
7. Exit with the child status; shut down the metrics server.

One supervisor process owns one child. There is no in-process job queue —
Buildkite concurrency is the agent’s problem.

### Configuration surface

Runtime is entirely environment-driven. The important knobs:

| Variable | Values | Notes |
| --- | --- | --- |
| `KAITE_HARDWARE` | `cpu` `apple` `nvidia` `amd` `intel` | Selects expected device contract and default tags |
| `KAITE_VARIANT` | `slim` `full` | Must match the image tag footprint |
| `KAITE_O11Y` | `none` `prometheus` `datadog` `splunk` | Metrics/log adapter selection |
| `KAITE_RUN_MODE` | `agent` `command` | Production vs diagnostics |
| `BUILDKITE_AGENT_TOKEN` / `_FILE` | secret | Exactly one required in agent mode |
| `BUILDKITE_AGENT_*` | agent CLI flags | Name, queue, tags, endpoint, disconnect, … |

When `BUILDKITE_AGENT_TAGS` is unset, Kaite supplies:

```text
kaite=true,kaite.hardware=<hw>,kaite.variant=<variant>,kaite.o11y=<o11y>
```

Token values are never written into argv when the env-token path is used; the
file-token path uses Buildkite’s `file://` syntax so the secret stays on disk.

### Health and metrics

| Endpoint | Meaning |
| --- | --- |
| `GET /healthz` | Supervisor process is alive |
| `GET /readyz` | Child process is running |
| `GET /metrics` | Prometheus text exposition |

Prometheus series include hardware/variant/o11y labels and the supervisor
version. Datadog mode emits DogStatsD gauges/counters. Splunk mode exposes the
same scrape endpoint and honors standard `OTEL_*` variables for a collector
sidecar or DaemonSet.

## Image matrix

[`docker-bake.hcl`](../docker-bake.hcl) is the source of truth. Every target
shares one Dockerfile and varies base image, platforms, and build args.

```text
                    slim                          full
              ┌─────────────────┐          ┌─────────────────┐
  cpu         │ amd64 + arm64   │          │ amd64 + arm64   │   active
  apple       │ arm64 only      │          │ arm64 only      │   active
  nvidia      │ amd64 (CUDA)    │          │ amd64 (CUDA)    │   inactive CI
  amd         │ amd64 (ROCm)    │          │ amd64 (ROCm)    │   inactive CI
  intel       │ amd64 (oneAPI)  │          │ amd64 (oneAPI)  │   inactive CI
              └─────────────────┘          └─────────────────┘
```

Tag shapes:

- Versioned: `kaite:<semver>-<hardware>-<variant>` (e.g. `v0.2.0-cpu-slim`)
- Stable aliases: `cpu-slim`, `cpu-full`, `apple-slim`, `apple-full`
- Compatibility: bare `cpu` / `apple` still point at slim

### Build layers inside the image

1. **Builder stage** — `golang:1.23` cross-compiles a static `kaite` binary.
2. **Runtime base** — Ubuntu or vendor image (`BASE_IMAGE`).
3. **System packages** — bash, curl, git, tini, Python venv tooling.
4. **Buildkite agent** — pinned release, SHA256-verified install under `/buildkite`.
5. **Python venv** — layered requirements (see below).
6. **Entrypoint** — `tini` → `/usr/local/bin/kaite`.

### Python requirement layers

Install order matters so the hardware PyTorch wheel wins:

| Variant | Manifests (in order) |
| --- | --- |
| slim | `slim.txt` → `<hardware>.txt` |
| full | slim set → `base.txt` → `training.txt` → `orchestration.txt` → `serving.txt` |

Non-portable packages (DeepSpeed, bitsandbytes, FlashAttention, vLLM, s3fs, …)
are **not** defaults. Operators pass them with `KAITE_EXTRA_PYTHON_PACKAGES` at
build time. See [`requirements/README.md`](../requirements/README.md).

## Deploy paths

### Docker launcher

[`deploy/docker/run.sh`](../deploy/docker/run.sh) is a thin wrapper:

- Requires a Buildkite token (or command override) in agent mode
- Forwards `BUILDKITE_*` / `KAITE_*` / o11y env
- Adds device flags for nvidia (`--gpus`), amd/intel (`/dev/dri`, …)
- Optionally runs a container command (`KAITE_CONTAINER_COMMAND=smoke`)

[`deploy/docker/smoke.sh`](../deploy/docker/smoke.sh) sets that command so hosts
can verify devices and frameworks without a cluster token.

### Kubernetes Job

[`deploy/kubernetes/kaite-agent.yaml`](../deploy/kubernetes/kaite-agent.yaml) is a
**one-shot** agent: register → claim one matching job → disconnect → Job
completes. Continuous pools need a Deployment or Buildkite Agent Stack.

Secrets stay in a Kubernetes Secret (`buildkite-agent`); the template never
embeds a token. Prometheus scrape annotations are ready for cluster scrapers.

Do **not** set `BUILDKITE_KUBERNETES_EXEC` on this plain Job — that flag belongs
to Agent Stack for Kubernetes only.

### Pipeline targeting

Jobs select Kaite agents with tags, not image pulls inside the step:

```yaml
agents:
  queue: ai
  kaite.hardware: cpu
  kaite.variant: slim
```

The image already is the job environment; the step command runs inside it.

## Observability model

```text
KAITE_O11Y=none        → no metrics server (unless KAITE_METRICS_ADDR set)
KAITE_O11Y=prometheus  → bind :9090, scrape /metrics
KAITE_O11Y=datadog     → DogStatsD to KAITE_DD_* / DD_*
KAITE_O11Y=splunk      → metrics + OTEL_* for collector export
```

Always:

- Supervisor events → **stderr** as JSON (`component=kaite`)
- Child/job output → agent **stdout/stderr** (Buildkite job logs)

Never:

- Vendor API keys in the image
- Tokens printed into logs or agent argv (env-token path)

## Failure and exit codes

| Situation | Exit | Notes |
| --- | --- | --- |
| Invalid config / missing token | 2 | Fail closed before starting work |
| Agent binary missing, metrics bind fail, start fail | 1 | Structured error event on stderr |
| Child / Buildkite job failure | child’s code | Orchestrator decides retry |
| Clean shutdown after SIGTERM | 0 if child exits 0 | SIGTERM forwarded once |

## Release topology

```text
PR → main
      │
      ▼
Release Please PR (changelog + version bump)
      │ merge
      ▼
semver tag + GitHub Release
      │ dispatch
      ▼
release-images.yml  →  native CPU + Apple hosts
                      →  kaite smoke per active target
                      →  GHCR (versioned + alias tags, provenance, SBOM)
```

Accelerator jobs stay behind an explicit workflow input so a green release
cannot queue on hosts that do not exist. Local bake mirrors the same matrix
via `make build-*` (requires Docker Buildx; Podman-only hosts may lack bake).

## Non-goals (current slice)

- Apple Metal / GPU passthrough inside Ubuntu containers
- Framework-specific image forks beyond slim/full + hardware
- Embedding checkpoint stores or object-store credentials in the supervisor
- Replacing Buildkite as the job dispatcher

## Related docs

- [README](../README.md) — quick start and env table
- [CONTRIBUTING](../CONTRIBUTING.md) — local build and PR expectations
- [requirements/README](../requirements/README.md) — Python layer details
- [deploy/kubernetes/README](../deploy/kubernetes/README.md) — cluster Job notes
