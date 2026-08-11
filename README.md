# kaite

[![Release](https://img.shields.io/github/v/release/alexhraber/kaite?display_name=tag)](https://github.com/alexhraber/kaite/releases/latest)
[![CI](https://github.com/alexhraber/kaite/actions/workflows/build-images.yml/badge.svg?branch=main)](https://github.com/alexhraber/kaite/actions/workflows/build-images.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GHCR](https://img.shields.io/badge/ghcr.io-alexhraber%2Fkaite-blue)](https://github.com/alexhraber/kaite/pkgs/container/kaite)
[![🦀 Decapod](https://img.shields.io/badge/🦀%20Decapod-v0.96.21-dc2626)](https://github.com/DecapodLabs/decapod)

Self-hosted [Buildkite](https://buildkite.com) agents with a batteries-included
AI/ML execution environment. Choose the work capability and hardware; the
image already contains the Buildkite agent, pinned Python toolchain, hardware
contract, diagnostics, and lightweight observability.

Pronounced “kite” — the ai is just a harder i.

## Quick start

```bash
# Pull the latest data-science environment (stable alias)
docker pull ghcr.io/alexhraber/kaite:cpu-slim

# Run as a Buildkite agent
BUILDKITE_AGENT_TOKEN=… \
KAITE_HARDWARE=cpu \
KAITE_VARIANT=slim \
KAITE_O11Y=prometheus \
KAITE_IMAGE=ghcr.io/alexhraber/kaite:cpu-slim \
  ./deploy/docker/run.sh
```

For a training or serving job, use the full environment and select the
capability in the pipeline. The worker advertises the capabilities baked into
the image; the pipeline does not install them at job start.

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
| `cpu-slim` / `cpu-full` | `linux/amd64` | Agent + CPU PyTorch; data-science or all supported layers |
| `apple-slim` / `apple-full` | `linux/arm64` | Same stack for Apple Silicon hosts |
| `vX.Y.Z-<hardware>-<variant>` | as above | Immutable release tags |

`slim` is the compact data-science stack. `full` adds Hugging Face training,
Ray/MLflow/W&B, and FastAPI/Gradio serving.

The current capability set is deliberately small:

| Capability | Available in | Contract |
| --- | --- | --- |
| `data-science` | slim and full | NumPy, pandas, scikit-learn, Jupyter, and hardware-specific PyTorch |
| `training` | full | Hugging Face, Lightning, and model-training frameworks |
| `orchestration` | full | Ray plus MLflow and W&B |
| `serving` | full | FastAPI, Gradio, and Uvicorn |

Every official image records this set in `/etc/kaite/identity.json`, exposes it
through `kaite doctor`, and advertises it to Buildkite as
`kaite.capability.<name>=true`. `slim` and `full` remain compatibility names;
they describe package footprints, while capabilities describe usable work.

Versioned tags: `v0.2.1-cpu-slim`, `v0.2.1-apple-full`, …. Unversioned aliases
(`cpu-slim`, …) track the latest successful release.
Capability aliases such as `cpu-training` and `cpu-serving` point at the
corresponding full artifact; they improve discoverability without creating
additional images or a new matrix dimension.

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

`kaite doctor` reports the baked identity and detected devices. `kaite smoke`
validates representative imports for each declared capability and performs
the hardware check when an accelerator is expected. Runtime attempts to
override the baked hardware, variant, or capability set fail closed.

Deeper layout: [docs/architecture.md](docs/architecture.md) and the
[capability contract](docs/capabilities.md).

Read the [Kaite documentation site](https://alexhraber.github.io/kaite/) for
the execution-substrate perspective, capability contract, and architecture.

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
  kaite.capability.data-science: "true"
```

Training and serving jobs request `kaite.capability.training: "true"` or
`kaite.capability.serving: "true"` and can land on a full worker. The old
`kaite.variant` selector remains valid for compatibility, but it is not a
workload promise. See [`examples/pipeline.yml`](examples/pipeline.yml) for a
complete set of selectors. Kaite reserves `kaite.*` agent tags so custom tags
cannot silently contradict the image identity.

## Toolchain layers

| Layer | Installed when | Contents |
| --- | --- | --- |
| `slim.txt` + `<hardware>.txt` | `data-science` | NumPy/pandas/sklearn/Jupyter + hardware PyTorch |
| `base.txt` + `training.txt` | `training` | Hugging Face and Lightning training stack |
| `orchestration.txt` | `orchestration` | Ray/MLflow/W&B |
| `serving.txt` | `serving` | FastAPI/Gradio/Uvicorn |

Vendor-specific packages (DeepSpeed, bitsandbytes, FlashAttention, vLLM, s3fs, …)
stay out of the defaults. Pass them through `KAITE_EXTRA_PYTHON_PACKAGES` at
build time. Details: [`requirements/README.md`](requirements/README.md).

## Derive an organizational image

Kaite is also a base layer. Pin an immutable artifact, then add only the
organization's delta:

```dockerfile
FROM ghcr.io/alexhraber/kaite:v0.2.2-cpu-full

COPY internal-requirements.txt /tmp/
RUN /opt/kaite/venv/bin/pip install --no-cache-dir -r /tmp/internal-requirements.txt
COPY internal-tools/ /opt/acme-tools/
```

The inherited identity and supervisor remain the source of the advertised
capability contract. If an extension replaces core frameworks or hardware
packages, the organization should rerun `kaite doctor`/`kaite smoke` and own
the resulting compatibility claim.

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
