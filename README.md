# kaite

[![🦀 Decapod](https://img.shields.io/badge/🦀%20Decapod-v0.96.15-dc2626)](https://github.com/DecapodLabs/decapod)

Self-hosted [Buildkite](https://buildkite.com) agents with a batteries-included
AI/ML runtime. One image ships the agent, a pinned Python toolchain, hardware
contracts, and lightweight observability.

```bash
BUILDKITE_AGENT_TOKEN=… \
KAITE_HARDWARE=cpu KAITE_VARIANT=slim KAITE_O11Y=prometheus \
  ./deploy/docker/run.sh
```

## Images

Published to `ghcr.io/alexhraber/kaite`. Active tags:

| Tag | Platform | Footprint |
| --- | --- | --- |
| `cpu-slim` / `cpu-full` | `linux/amd64`, `linux/arm64` | Agent + CPU PyTorch |
| `apple-slim` / `apple-full` | `linux/arm64` (Apple Silicon hosts) | Same as CPU, arm64-only |

`slim` is the compact data-science stack. `full` adds Hugging Face training,
Ray/MLflow/W&B, and FastAPI/Gradio serving. Versioned tags look like
`v0.2.0-cpu-slim`; unversioned aliases track the latest release.

NVIDIA / AMD / Intel bake targets exist for deliberate host testing but are
**inactive** in CI and automatic releases. Opt in with
`make build-all-accelerators` or the image workflow’s accelerator input.

```bash
make build-plan   # print bake graph
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

Deeper layout (supervisor files, image matrix, deploy paths, failure model):
[docs/architecture.md](docs/architecture.md).

## Deploy

- **Docker:** [`deploy/docker/run.sh`](deploy/docker/run.sh) — local agent launch.
  Use `deploy/docker/smoke.sh` for device/framework checks without a token.
- **Kubernetes:** [`deploy/kubernetes/kaite-agent.yaml`](deploy/kubernetes/kaite-agent.yaml)
  — one-shot Job that claims a Buildkite job and exits. See
  [`deploy/kubernetes/README.md`](deploy/kubernetes/README.md).
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

Vendor-specific or non-portable packages (DeepSpeed, bitsandbytes, FlashAttention,
vLLM, s3fs, …) stay out of the defaults. Pass them through
`KAITE_EXTRA_PYTHON_PACKAGES` at build time. Details in
[`requirements/README.md`](requirements/README.md).

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

Images and releases: Release Please opens a release PR from `main`; merging it
tags, creates a GitHub release, then dispatches the image workflow against that
tag so active GHCR images (CPU/Apple slim+full) publish with provenance and
SBOM. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © Alex H. Raber
