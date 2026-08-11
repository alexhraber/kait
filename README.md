# kaite

[![Release](https://img.shields.io/github/v/release/alexhraber/kaite?display_name=tag)](https://github.com/alexhraber/kaite/releases/latest)
[![CI](https://github.com/alexhraber/kaite/actions/workflows/build-images.yml/badge.svg?branch=main)](https://github.com/alexhraber/kaite/actions/workflows/build-images.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GHCR](https://img.shields.io/badge/ghcr.io-alexhraber%2Fkaite-blue)](https://github.com/alexhraber/kaite/pkgs/container/kaite)
[![🦀 Decapod](https://img.shields.io/badge/🦀%20Decapod-v0.96.18-dc2626)](https://github.com/DecapodLabs/decapod)

Self-hosted [Buildkite](https://buildkite.com) agents with a batteries-included
AI/ML runtime. One image ships the agent, a pinned Python toolchain, hardware
contracts, and lightweight observability.

## Quick start

```bash
# Pull the latest CPU slim image (stable alias)
docker pull ghcr.io/alexhraber/kaite:cpu-slim

# Run as a Buildkite agent
BUILDKITE_AGENT_TOKEN=… \
KAITE_HARDWARE=cpu \
KAITE_VARIANT=slim \
KAITE_O11Y=prometheus \
KAITE_IMAGE=ghcr.io/alexhraber/kaite:cpu-slim \
  ./deploy/docker/run.sh
```

Pin a release for production:

```bash
docker pull ghcr.io/alexhraber/kaite:v0.2.1-cpu-slim
```

> **Package visibility:** the container package must be **public** for anonymous
> pulls. If `docker pull` returns 401/404 for a public repo, open
> [Package settings](https://github.com/users/alexhraber/packages/container/package/kaite/settings)
> → **Change visibility** → Public.

## Images

Registry: [`ghcr.io/alexhraber/kaite`](https://github.com/alexhraber/kaite/pkgs/container/kaite)

| Tag | Platform (published) | Footprint |
| --- | --- | --- |
| `cpu-slim` / `cpu-full` | `linux/amd64` | Agent + CPU PyTorch |
| `apple-slim` / `apple-full` | `linux/arm64` | Same stack for Apple Silicon hosts |
| `vX.Y.Z-<hardware>-<variant>` | as above | Immutable release tags |

`slim` is the compact data-science stack. `full` adds Hugging Face training,
Ray/MLflow/W&B, and FastAPI/Gradio serving.

Versioned tags: `v0.2.1-cpu-slim`, `v0.2.1-apple-full`, …. Unversioned aliases
(`cpu-slim`, …) track the latest successful release.

NVIDIA / AMD / Intel bake targets exist for deliberate host testing but are
**inactive** in automatic CI/release. Opt in with `make build-all-accelerators`
or the image workflow’s accelerator input.

```bash
make build-plan   # print bake graph (Docker Buildx)
make build-slim   # cpu + apple slim
make build-full   # cpu + apple full
```

## Runtime

Kaite is a small Go supervisor (`cmd/kaite`) that validates hardware/variant/o11y
settings, starts `buildkite-agent start` (or a diagnostic command), exposes
health/metrics when enabled, and exits with the child status.

| Variable | Default | Purpose |
| --- | --- | --- |
| `BUILDKITE_AGENT_TOKEN` | — | Cluster agent token (or use `…_TOKEN_FILE`) |
| `KAITE_HARDWARE` | `cpu` | `cpu` · `apple` · `nvidia` · `amd` · `intel` |
| `KAITE_VARIANT` | `slim` | `slim` · `full` |
| `KAITE_O11Y` | `none` | `none` · `prometheus` · `datadog` · `splunk` |
| `KAITE_RUN_MODE` | `agent` | `agent` or `command` (+ `KAITE_COMMAND`) |
| `BUILDKITE_AGENT_QUEUE` / `TAGS` | — | Queue and capability targeting |

```bash
kaite doctor    # JSON hardware probe
kaite smoke     # framework + device check (used by CI)
kaite hardware  # accelerator CLI output
```

Deeper layout: [docs/architecture.md](docs/architecture.md).

## Deploy

- **Docker:** [`deploy/docker/run.sh`](deploy/docker/run.sh) — local agent launch.
  Use `deploy/docker/smoke.sh` for device/framework checks without a token.
- **Kubernetes:** [`deploy/kubernetes/kaite-agent.yaml`](deploy/kubernetes/kaite-agent.yaml)
  — one-shot Job. See [`deploy/kubernetes/README.md`](deploy/kubernetes/README.md).
- **Pipeline targeting:** [`examples/pipeline.yml`](examples/pipeline.yml)

```yaml
agents:
  queue: ai
  kaite.hardware: cpu
  kaite.variant: slim
```

## Toolchain layers

| Layer | Installed when | Contents |
| --- | --- | --- |
| `slim.txt` + `<hardware>.txt` | every image | NumPy/pandas/sklearn/Jupyter + hardware PyTorch |
| `base.txt` `training.txt` `orchestration.txt` `serving.txt` | `*-full` only | broader science, HF stack, Ray/MLflow/W&B, FastAPI/Gradio |

Vendor-specific packages (DeepSpeed, bitsandbytes, FlashAttention, vLLM, s3fs, …)
stay out of the defaults. Pass them through `KAITE_EXTRA_PYTHON_PACKAGES` at
build time. Details: [`requirements/README.md`](requirements/README.md).

## Observability

```bash
KAITE_O11Y=prometheus   # scrape :9090/metrics
KAITE_O11Y=datadog      # DogStatsD via KAITE_DD_* / DD_*
KAITE_O11Y=splunk       # same metrics + standard OTEL_* env
```

Structured JSON logs go to stderr. Child job output stays on the agent’s
stdout/stderr. Vendor credentials belong on the collector or node agent, not
in the image.

## Develop

```bash
make test
make vet
make build          # bin/kaite
```

## Releases

Release Please opens a release PR from `main`. Merging it:

1. Creates the semver tag and GitHub release
2. Dispatches `release-images.yml` against that tag (`actions: write` required)
3. Publishes active CPU/Apple slim+full images to GHCR with provenance and SBOM
4. Annotates the GitHub release with pull commands

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © Alex H. Raber
