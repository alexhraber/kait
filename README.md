# kait

[![Release](https://img.shields.io/github/v/release/alexhraber/kait?display_name=tag)](https://github.com/alexhraber/kait/releases/latest)
[![CI](https://github.com/alexhraber/kait/actions/workflows/build-images.yml/badge.svg?branch=main)](https://github.com/alexhraber/kait/actions/workflows/build-images.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GHCR](https://img.shields.io/badge/ghcr.io-alexhraber%2Fkait-blue)](https://github.com/alexhraber/kait/pkgs/container/kait)
[![🦀 Decapod](https://img.shields.io/badge/🦀%20Decapod-v0.98.0-dc2626)](https://github.com/DecapodLabs/decapod)

Self-hosted [Buildkite](https://buildkite.com) agents with a batteries-included
AI/ML execution environment. Choose the work capability and hardware; the
selected Linux container already contains the Buildkite integration, pinned
Python toolchain, hardware contract, diagnostics, and lightweight
observability.

Kait is pronounced “kite.” Its spelling keeps `ai` at the center, reflecting
its focus on AI and machine-learning execution. It also carries a small
linguistic resonance: 愛 (*ai*) is the Japanese kanji for “love”; in Japanese,
the vowels in *ai* are articulated together as /a.i/, which English speakers
commonly perceive as close to “eye.” The connection is a resonance in the
spelling, not a claim that Kait is a Japanese word; spoken aloud, it remains
simply “kite.”

## Quick start

```bash
# Pull the latest data-science environment (stable alias)
docker pull ghcr.io/alexhraber/kait:cpu-slim

# Run as a Buildkite agent
BUILDKITE_AGENT_TOKEN=… \
KAIT_HARDWARE=cpu \
KAIT_VARIANT=slim \
KAIT_O11Y=prometheus \
KAIT_IMAGE=ghcr.io/alexhraber/kait:cpu-slim \
  ./deploy/docker/run.sh
```

For a training, orchestration, or serving job, choose the matching workload
profile. The worker advertises the capabilities baked into that image; the
pipeline does not install them at job start.

Apple is modeled as a native macOS/MPS hardware contract, but it is currently
disabled like the other accelerator classes. On Apple Silicon, use the
ordinary multi-architecture Linux CPU image through Docker or Apple Container
and advertise `kait.hardware=cpu`.

Pin a release for production:

```bash
docker pull ghcr.io/alexhraber/kait:v0.2.1-cpu-slim
```

> **Package visibility:** the container package must be **public** for anonymous
> pulls. If `docker pull` returns 401/404 for a public repo, open
> [Package settings](https://github.com/users/alexhraber/packages/container/package/kait/settings)
> → **Change visibility** → Public.

## Images

Registry: [`ghcr.io/alexhraber/kait`](https://github.com/alexhraber/kait/pkgs/container/kait)

| Tag | Platform (published) | Footprint |
| --- | --- | --- |
| `<hardware>-slim` / `<hardware>-full` | Linux container hardware contract | Compatibility profiles |
| `<hardware>-data-science` | hardware contract | NumPy, pandas, scikit-learn, Jupyter, and hardware-specific PyTorch |
| `<hardware>-training` | hardware contract | Data-science plus Hugging Face and Lightning |
| `<hardware>-orchestration` | hardware contract | Ray, MLflow, and W&B |
| `<hardware>-serving` | hardware contract | FastAPI, Gradio, and Uvicorn |
| `vX.Y.Z-<hardware>-<profile>` | Linux container hardware contract | Immutable release tags |

`slim` is the compact data-science stack. `full` adds Hugging Face training,
Ray/MLflow/W&B, and FastAPI/Gradio serving.

The public capability set is deliberately small:

| Capability | Available in | Contract |
| --- | --- | --- |
| `data-science` | `slim`, `full`, `data-science`, `training` | NumPy, pandas, scikit-learn, Jupyter, and hardware-specific PyTorch |
| `training` | `full`, `training` | Hugging Face and Lightning training tooling; composes data-science |
| `orchestration` | `full`, `orchestration` | Ray plus MLflow and W&B |
| `serving` | `full`, `serving` | FastAPI, Gradio, and Uvicorn |

Every official container records this set in `/etc/kait/identity.json`, exposes it
through `kait doctor`, and advertises it to Buildkite as
`kait.capability.<name>=true`. `slim` and `full` remain compatibility names;
they describe package footprints, while capabilities describe usable work.

The native Apple MPS contract remains modeled for future re-enablement, but is
inactive in CI and release matrices and is not a published execution surface.

Versioned container tags use the six profiles, for example
`v0.5.0-cpu-training`. Apple-specific profiles are currently not released;
Apple Silicon users should use the `cpu-*` multi-architecture images.
Previously published `apple-*` tags remain Linux CPU compatibility artifacts
and are not GPU surfaces. `slim` and `full` remain compatibility profiles,
while workload-specific artifacts have their own identity and smoke proof.

NVIDIA / AMD / Intel bake targets exist for deliberate host testing but are
**inactive** in automatic CI/release. Opt in with `make build-all-accelerators`
or the image workflow’s accelerator input.

```bash
make build-plan   # print bake graph (Docker Buildx)
make build-slim   # Linux CPU slim compatibility profile
make build-full   # Linux CPU full compatibility profile
make build-profiles # all six active Linux container profiles
```

## Runtime

Kait is a small Go supervisor (`cmd/kait`) that validates the baked
hardware/profile/capability contract, starts `buildkite-agent start` (or a
diagnostic command), exposes health/metrics when enabled, and exits with the
child status.

| Variable | Default | Purpose |
| --- | --- | --- |
| `BUILDKITE_AGENT_TOKEN` | — | Cluster agent token (or use `…_TOKEN_FILE`) |
| `KAIT_HARDWARE` | `cpu` | `cpu` · `apple` · `nvidia` · `amd` · `intel` |
| `KAIT_VARIANT` | `slim` | `slim` · `full` |
| `KAIT_PROFILE` | image profile | `slim` · `full` · `data-science` · `training` · `orchestration` · `serving` |
| `KAIT_O11Y` | `none` | `none` · `prometheus` · `datadog` · `splunk` |
| `KAIT_RUN_MODE` | `agent` | `agent` or `command` (+ `KAIT_COMMAND`) |
| `BUILDKITE_AGENT_QUEUE` / `TAGS` | — | Queue and capability targeting |

```bash
kait doctor    # JSON hardware probe
kait smoke     # framework + device check (used by CI)
kait hardware  # accelerator CLI output
```

`kait doctor` reports the baked identity, available capability checks, expected
hardware, detected devices, and whether the host satisfies the hardware
contract. `kait smoke` performs representative bounded programs for every
declared capability and validates accelerator access when the profile includes
the PyTorch data-science contract. A missing baked identity or runtime attempt
to override it fails closed.

On Apple Silicon, run the Linux CPU image and use the CPU hardware selector.
Apple-native doctor/smoke execution is inactive until a matching GPU surface
is deliberately re-enabled.

Deeper layout: [docs/architecture.md](docs/architecture.md) and the
[capability contract](docs/capabilities.md).

Read the [Kait documentation site](https://alexhraber.github.io/kait/) for
the execution-substrate perspective, capability contract, and architecture.

## Deploy

- **Docker:** [`deploy/docker/run.sh`](deploy/docker/run.sh) — local agent launch.
  Use `deploy/docker/smoke.sh` for device/framework checks without a token.
- **Kubernetes:** [`deploy/kubernetes/kait-agent.yaml`](deploy/kubernetes/kait-agent.yaml)
  — one-shot Job. See [`deploy/kubernetes/README.md`](deploy/kubernetes/README.md).
- **Pipeline targeting:** [`examples/pipeline.yml`](examples/pipeline.yml)

```yaml
agents:
  queue: ai
  kait.hardware: cpu
  kait.capability.data-science: "true"
```

Training and serving jobs request `kait.capability.training: "true"` or
`kait.capability.serving: "true"` and can land on a full worker. The old
`kait.variant` selector remains valid for compatibility, but it is not a
workload promise. See [`examples/pipeline.yml`](examples/pipeline.yml) for a
complete set of selectors. Kait reserves `kait.*` agent tags so custom tags
cannot silently contradict the image identity.

Apple Silicon CPU jobs use the same selector as any other CPU worker:
`kait.hardware: cpu`. Apple GPU selectors remain unavailable until the native
hardware contract is deliberately re-enabled.

## Toolchain layers

| Layer | Installed when | Contents |
| --- | --- | --- |
| `slim.txt` + `<hardware>.txt` | Linux `data-science` | NumPy/pandas/sklearn/Jupyter + hardware PyTorch |
| `apple-mps.txt` + `slim.txt` | Reserved native Apple contract | Inactive until Apple GPU execution is re-enabled |
| `base.txt` + `training.txt` | `training` | Hugging Face and Lightning training stack, composed on data-science |
| `orchestration.txt` | `orchestration` | Ray/MLflow/W&B |
| `serving.txt` | `serving` | FastAPI/Gradio/Uvicorn |

Vendor-specific packages (DeepSpeed, bitsandbytes, FlashAttention, vLLM, s3fs, …)
stay out of the defaults. Pass them through `KAIT_EXTRA_PYTHON_PACKAGES` at
build time. Details: [`requirements/README.md`](requirements/README.md).

## Derive an organizational image

Kait is also a base layer. Pin an immutable artifact, then add only the
organization's delta:

```dockerfile
FROM ghcr.io/alexhraber/kait:v0.2.1-cpu-full

COPY internal-requirements.txt /tmp/
RUN /opt/kait/venv/bin/pip install --no-cache-dir -r /tmp/internal-requirements.txt
COPY internal-tools/ /opt/acme-tools/
```

The inherited identity and supervisor remain the source of the advertised
capability contract. If an extension replaces core frameworks or hardware
packages, the organization should rerun `kait doctor`/`kait smoke` and own
the resulting compatibility claim.

## Observability

```bash
KAIT_O11Y=prometheus   # scrape :9090/metrics
KAIT_O11Y=datadog      # DogStatsD via KAIT_DD_* / DD_*
KAIT_O11Y=splunk       # same metrics + standard OTEL_* env
```

Structured JSON logs go to stderr. Child job output stays on the agent’s
stdout/stderr. Vendor credentials belong on the collector or node agent, not
in the image.

## Develop

```bash
make test
make vet
make build          # bin/kait
```

## Releases

Release Please opens a release PR from `main`. Merging it:

1. Creates the semver tag and GitHub release
2. Dispatches `release-images.yml` against that tag (`actions: write` required)
3. Publishes active Linux container profiles to GHCR with provenance and SBOM
4. Annotates the GitHub release with pull/install commands

When a merged change has no conventional release unit, the same workflow opens
an automatic patch release PR so infrastructure, documentation, and identity
changes still reach the versioned image publication path.

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) © Alex H. Raber
