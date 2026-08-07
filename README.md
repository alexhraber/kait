# kaite

[![🦀 Decapod](https://img.shields.io/badge/🦀%20Decapod-v0.96.13-dc2626)](https://github.com/DecapodLabs/decapod)

Kaite is the standard AI runtime for self-hosted Buildkite, packaging the
agent, hardware-specific toolchains, observability, and lifecycle controls
needed to run AI workloads across CPU hosts, GPU nodes, Kubernetes clusters,
and other Buildkite-backed compute environments.

## Use Kaite

Create a Buildkite cluster agent token and run the published CPU image:

~~~bash
BUILDKITE_AGENT_TOKEN=replace-me \
KAITE_HARDWARE=cpu \
KAITE_O11Y=prometheus \
  KAITE_IMAGE=ghcr.io/alexhraber/kaite:cpu \
  ./deploy/docker/run.sh
~~~

Set `KAITE_HARDWARE=apple` for the active arm64 CPU image. The NVIDIA, AMD, and
Intel image contracts remain available for deliberate host testing, but their
automatic CI/release paths and `kaite-nvidia`, `kaite-amd`, and `kaite-intel`
runner labels are currently inactive. The launcher still adds accelerator
device flags when an operator explicitly selects one of those images.

## Image catalog

The release workflow publishes versioned images and stable hardware aliases
to GHCR:

| Image | Platform and runtime contract | Status | Included framework option |
| --- | --- | --- | --- |
| `ghcr.io/alexhraber/kaite:cpu` | Ubuntu 24.04, native `linux/amd64` release image | active | Common Python toolchain |
| `ghcr.io/alexhraber/kaite:apple` | Ubuntu 24.04, `linux/arm64` CPU for Apple Silicon hosts | active | Common Python toolchain |
| `ghcr.io/alexhraber/kaite:nvidia` | CUDA 12.6.3, `linux/amd64` | inactive | PyTorch CUDA 12.6 wheels |
| `ghcr.io/alexhraber/kaite:amd` | ROCm 6.2.4, `linux/amd64` | inactive | PyTorch ROCm 6.2.4 wheels |
| `ghcr.io/alexhraber/kaite:intel` | oneAPI Base Toolkit 2025.0.1 / Ubuntu 22.04, `linux/amd64` | inactive | Python 3.11 + Intel Extension for PyTorch XPU |

Versioned active releases use tags such as `v1.2.3-cpu` (native amd64) and
`v1.2.3-apple` (native arm64).
Accelerator tags are only produced by an explicit opt-in workflow run.
`docker-bake.hcl` is the source of truth for the image matrix and can build
the same targets locally:

~~~bash
make build-plan
make build-cpu
make build-all
# Explicitly opt in to inactive accelerator targets.
make build-all-accelerators
~~~

On a matching accelerator host, run `deploy/docker/smoke.sh` to verify device
visibility and the selected framework without a Buildkite token. The script
uses the same Docker device flags as the agent launcher.

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

Push a semantic version tag such as `v1.2.3` to publish the active CPU and Apple
images and create a GitHub release. The workflow logs into GHCR with the
repository token, publishes provenance and SBOM attestations, and pulls each
published image to run `doctor`. NVIDIA, AMD, and Intel publication is inactive
on tag pushes; a manual workflow run must explicitly set
`enable_accelerators=true` and requires the matching host labels.

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
