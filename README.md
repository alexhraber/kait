# kaite

[![🦀 Decapod](https://img.shields.io/badge/🦀%20Decapod-v0.96.15-dc2626)](https://github.com/DecapodLabs/decapod)

Kaite is the standard AI runtime for self-hosted Buildkite, packaging the
agent, hardware-specific toolchains, observability, and lifecycle controls
needed to run AI workloads across CPU hosts, GPU nodes, Kubernetes clusters,
and other Buildkite-backed compute environments.

## Use Kaite

Create a Buildkite cluster agent token and run the published CPU image:

~~~bash
BUILDKITE_AGENT_TOKEN=replace-me \
KAITE_HARDWARE=cpu \
KAITE_VARIANT=slim \
KAITE_O11Y=prometheus \
  KAITE_IMAGE=ghcr.io/alexhraber/kaite:cpu-slim \
  ./deploy/docker/run.sh
~~~

Set `KAITE_HARDWARE=apple` and use `kaite:apple-slim` or `kaite:apple-full` for
an active arm64 CPU image. Every image contains the pinned Buildkite agent;
`slim` and `full` select the Python/tooling footprint around that agent. The
NVIDIA, AMD, and Intel image contracts remain available for deliberate host
testing, but their automatic CI/release paths and `kaite-nvidia`, `kaite-amd`,
and `kaite-intel` runner labels are currently inactive. The launcher still adds
accelerator device flags when an operator explicitly selects one of those
images.

## Image catalog

The release workflow publishes versioned images and stable hardware aliases
to GHCR:

| Image | Platform and runtime contract | Status | Included framework option |
| --- | --- | --- | --- |
| `ghcr.io/alexhraber/kaite:cpu-slim` | Ubuntu 24.04, `linux/amd64` and `linux/arm64` | active | Agent plus compact Python and CPU PyTorch |
| `ghcr.io/alexhraber/kaite:cpu-full` | Ubuntu 24.04, `linux/amd64` and `linux/arm64` | active | Agent plus complete portable AI/ML toolchain |
| `ghcr.io/alexhraber/kaite:apple-slim` | Ubuntu 24.04, `linux/arm64` for Apple Silicon hosts | active | Agent plus compact Python and CPU PyTorch |
| `ghcr.io/alexhraber/kaite:apple-full` | Ubuntu 24.04, `linux/arm64` for Apple Silicon hosts | active | Agent plus complete portable AI/ML toolchain |
| `ghcr.io/alexhraber/kaite:nvidia-slim` | CUDA 12.6.3, `linux/amd64` | inactive | Agent plus CUDA PyTorch contract |
| `ghcr.io/alexhraber/kaite:nvidia-full` | CUDA 12.6.3, `linux/amd64` | inactive | Agent plus complete CUDA AI/ML toolchain |
| `ghcr.io/alexhraber/kaite:amd-slim` | ROCm 6.2.4, `linux/amd64` | inactive | Agent plus ROCm PyTorch contract |
| `ghcr.io/alexhraber/kaite:amd-full` | ROCm 6.2.4, `linux/amd64` | inactive | Agent plus complete ROCm AI/ML toolchain |
| `ghcr.io/alexhraber/kaite:intel-slim` | oneAPI Base Toolkit 2025.0.1 / Ubuntu 22.04, `linux/amd64` | inactive | Agent plus Intel XPU contract |
| `ghcr.io/alexhraber/kaite:intel-full` | oneAPI Base Toolkit 2025.0.1 / Ubuntu 22.04, `linux/amd64` | inactive | Agent plus complete Intel AI/ML toolchain |

Versioned releases use the canonical form `kaite:<tag>-<hardware>-<variant>`,
such as `v1.2.3-cpu-slim`, `v1.2.3-cpu-full`, `v1.2.3-apple-slim`, and
`v1.2.3-apple-full`. Stable aliases are `cpu-slim`, `cpu-full`, `apple-slim`,
and `apple-full`; the older `cpu` and `apple` aliases continue to point at
`slim` for compatibility.
Accelerator tags are only produced by an explicit opt-in workflow run.
`docker-bake.hcl` is the source of truth for the image matrix and can build
the same targets locally:

~~~bash
make build-plan
make build-cpu
make build-slim
make build-full
make build-all
make build-all-full
# Explicitly opt in to inactive accelerator targets.
make build-all-accelerators
~~~

On a matching accelerator host, run `deploy/docker/smoke.sh` to verify device
visibility and the selected framework without a Buildkite token. The script
uses the same Docker device flags as the agent launcher.

## AI/ML toolchain

Both variants include the Go supervisor, pinned Buildkite agent, Python
environment, and selected hardware runtime. `*-slim` adds the compact data
science foundation and hardware-specific PyTorch wheels. `*-full` adds the
broader data/science layer, Hugging Face training stack, Ray/MLflow/W&B
orchestration, and FastAPI/Gradio/Uvicorn serving tools. DeepSpeed,
bitsandbytes, FlashAttention, s3fs, and vLLM remain explicit
`KAITE_EXTRA_PYTHON_PACKAGES` operator extras because their native or cloud
client contracts are not portable defaults.

All images intentionally use Ubuntu/glibc so the Buildkite agent, Python
wheels, and vendor runtimes share one Linux base. The Go supervisor is static,
but a musl/Alpine image would be a separate dependency contract and is not
presented as equivalent. Apple Silicon uses Linux arm64 CPU execution; Apple
GPU/Metal acceleration is not available inside an Ubuntu container.

## Buildkite contract

Kaite starts `buildkite-agent start` by default and preserves the normal
Buildkite self-hosted agent workflow: the agent registers with the selected
Buildkite cluster, waits for matching work, and runs each step in the image.
Required inputs are:

- `BUILDKITE_AGENT_TOKEN`, or `BUILDKITE_AGENT_TOKEN_FILE` for a mounted secret.
- `KAITE_VARIANT` (`slim` or `full`) and `KAITE_HARDWARE` select the image
  contract; the image tag should use the matching `<tag>-<hardware>-<variant>`.
- `BUILDKITE_AGENT_QUEUE` for the Buildkite queue and
  `BUILDKITE_AGENT_TAGS` for capability targeting. Kaite supplies `kaite=true`,
  `kaite.hardware`, and `kaite.o11y` tags when none are given.
- `BUILDKITE_AGENT_NAME`, `BUILDKITE_AGENT_CONFIG`,
  `BUILDKITE_AGENT_DISCONNECT_AFTER_JOB`, and the other `buildkite-agent start`
  options accepted by the launcher.

`BUILDKITE_KUBERNETES_EXEC` is only for Buildkite Agent Stack for Kubernetes;
leave it unset for the plain Kubernetes `Job` template in this repository.
See [Buildkite self-hosted agents](https://buildkite.com/docs/agent/self-hosted)
and the [`start` command reference](https://buildkite.com/docs/agent/cli/reference/start)
for the cluster-token, queue, and agent lifecycle contract.

`KAITE_RUN_MODE=command` with `KAITE_COMMAND` is available for diagnostics and
custom executors. `kaite doctor` reports accelerator command availability, and
`kaite smoke` runs the Go-owned toolchain/device check used by the Docker and
Buildkite examples.

## Observability

Choose the runtime adapter with one variable:

~~~bash
KAITE_O11Y=none|prometheus|datadog|splunk
~~~

The supervisor emits structured JSON logs to stderr while child job output
remains on stdout/stderr. Prometheus exposes `/metrics` for scraping. Datadog emits runtime counters through DogStatsD
using `KAITE_DD_AGENT_HOST` and `KAITE_DD_DOGSTATSD_PORT` (or the standard
`DD_*` names). Splunk mode exposes the same metrics and honors
`OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_SERVICE_NAME` for a Splunk
OpenTelemetry Collector. Vendor credentials stay in the collector or node
agent rather than in the Kaite image.

## Kubernetes

[`deploy/kubernetes/kaite-agent.yaml`](deploy/kubernetes/kaite-agent.yaml) is
a one-agent Job template: it registers, runs one matching Buildkite job, and
disconnects so Kubernetes can mark the Job complete. It uses a Secret for the
Buildkite token, exposes metrics for scraping, and leaves accelerator
resource/device settings explicit. CPU and Apple arm64 are the active image
paths; NVIDIA, AMD, and Intel remain explicit operator opt-ins with inactive
automatic CI/release runner labels. Change the image, `KAITE_HARDWARE`, agent
tags, queue, and resource requests for the target hardware. Use a Deployment
or Buildkite Agent Stack for a continuously replenished agent pool.

[`examples/pipeline.yml`](examples/pipeline.yml) shows Buildkite targeting:

~~~yaml
agents:
  queue: ai
  kaite.hardware: cpu
~~~

## Releases

Merging a normal pull request to `main` creates or updates the Release Please
release PR. Merging that PR creates the semantic tag and GitHub release, then
dispatches the image workflow against that immutable tag. The active CPU and
Apple images are built on native hosts, smoke-tested, and uploaded to GHCR with
provenance and SBOM attestations. NVIDIA, AMD, and Intel remain inactive unless
an operator explicitly enables the manual workflow on matching hosts.

The `release-please.yml` job requires the repository setting “Allow GitHub
Actions to create and approve pull requests” enabled for `GITHUB_TOKEN`, along
with the workflow’s contents, issues, and pull-request write permissions.

Image CI is host-matrixed and every active target runs `kaite smoke`: CPU uses
`ubuntu-24.04` and Apple uses native `ubuntu-24.04-arm`. The
`kaite-nvidia`, `kaite-amd`, and `kaite-intel` jobs are retained as explicit
manual opt-ins and remain inactive until matching hosts with Docker and vendor
device runtimes are registered.

## Verification

~~~bash
make test
GOCACHE=/tmp/kaite-gocache go vet ./...
KAITE_RUN_MODE=command KAITE_COMMAND='kaite doctor' KAITE_O11Y=prometheus \
  kaite
~~~
