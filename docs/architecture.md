---
layout: default
title: Architecture
description: Kait supervisor, capability contract, deployment, and release architecture.
---

# Architecture

Kait is a thin Go supervisor packaged into hardware-specific Linux images and a
reserved native macOS worker bundle path. Buildkite remains the orchestrator;
Kait owns the reproducible runtime and hardware contract, process lifecycle, identity
validation, health/metrics, and diagnostic subcommands.

## Operating boundary

```text
Docker / Kubernetes / macOS
        │
        ▼
┌───────────────────┐     start / signals      ┌────────────────────┐
│  kait supervisor  │ ───────────────────────► │  buildkite-agent   │
│  + contract model │ ◄─────────────────────── │  + job execution   │
└─────────┬─────────┘     exit status          └─────────┬──────────┘
          │                                              │
          │ identity / tags / diagnostics                 │ jobs, logs, artifacts
          ▼                                              ▼
      hardware                                    Buildkite control plane
```

Buildkite owns pipelines, scheduling, queues, dependencies, dynamic uploads,
gates, retries, artifacts, logs, and job state. Kait does not implement those
workflow semantics. It ensures that an agent selected by a capability tag is a
real, prepared, and validated execution surface.

## One capability model

[`cmd/kait/capability-contract.json`](../cmd/kait/capability-contract.json) is
the authoritative model. It defines:

- hardware classes, execution runtimes, base images where applicable, platforms, Python interpreters, and runner labels;
- workload capabilities, dependency relationships, manifest layers, summaries, and smoke programs;
- public profiles: `slim`, `full`, `data-science`, `training`, `orchestration`, and `serving`.

The embedded supervisor reads that model through `go:embed`. During container
construction or native bundle packaging, `kait contract` resolves the selected
hardware/profile and emits the identity plus the exact ordered requirements.
The Dockerfile installs container manifests and copies the identity to
`/etc/kait/identity.json`; the macOS bundle installs the same identity under
`/Library/Application Support/Kait`. `kait matrix` emits separate container
and native CI/release matrices from the same model.

The resulting flow is:

```text
capability model
  -> profile composition and hardware compatibility
  -> ordered requirement manifests
  -> container image or native bundle and baked identity
  -> startup identity validation
  -> Buildkite agent tags
  -> doctor/smoke proof
  -> generated CI/release matrix
  -> pipeline selectors and documentation
```

`docker-bake.hcl` is the release-facing projection for container rows. The
native macOS packager is the reserved projection for Apple rows. Both consume
the same six profiles; neither defines an alternative capability vocabulary.
Apple is currently inactive, like the other accelerator classes.

## Identity and startup

Official images contain an identity like:

```json
{
  "schema": 3,
  "hardware": "cpu",
  "runtime": "container",
  "accelerator": "cpu",
  "variant": "full",
  "profile": "training",
  "capabilities": ["data-science", "training"],
  "requirements": ["cpu.txt", "slim.txt", "base.txt", "training.txt"]
}
```

The identity file is mandatory. Runtime values can assert or constrain the
baked values, but cannot create a capability claim when the file is absent or
replace one with a different value. The supervisor validates profile,
capability composition, hardware support, execution runtime, accelerator, and
ordered requirement manifests before starting Buildkite. A native Apple
identity is the only identity that can advertise `accelerator=mps`.

OCI labels repeat image inputs for registry inspection; they are not a second
runtime authority.

## Supervisor process model

1. Apply the Intel oneAPI environment when an Intel image provides `setvars.sh`.
2. Resolve a diagnostic command (`contract`, `matrix`, `doctor`, `smoke`, or `hardware`) if requested.
3. Otherwise load and validate the baked identity and runtime configuration.
4. Start optional health/metrics endpoints.
5. Start either `buildkite-agent start` or the explicitly requested command mode.
6. Forward SIGTERM to the single child and propagate its exit status.

There is no in-process job queue. One supervisor owns one Buildkite child;
Buildkite owns concurrency and execution graph behavior.

## Buildkite metadata

The supervisor derives reserved tags from the validated identity:

```text
kait=true
kait.hardware=cpu
kait.variant=full
kait.profile=training
kait.o11y=prometheus
kait.capability.data-science=true
kait.capability.training=true
```

Organization tags remain supported. Attempts to override any `kait` or
`kait.*` tag fail before the agent starts. A pipeline therefore selects a
capability without knowing the host name or image implementation:

```yaml
agents:
  queue: ai
  kait.hardware: nvidia
  kait.capability.training: "true"
```

Static and dynamically uploaded Buildkite jobs use the same ordinary tag
matching. Kait does not need to know the graph at image-build time.

## Hardware matrix

| Hardware | Base/runtime | Platforms | Status |
| --- | --- | --- | --- |
| CPU | Ubuntu 24.04 + CPU PyTorch when required | amd64, arm64 | Active |
| Apple | Native macOS arm64 + Metal/MPS PyTorch when required | darwin/arm64 | Inactive; reserved native worker bundle |
| NVIDIA | CUDA 12.6.3 + CUDA PyTorch when required | amd64 | Explicit opt-in |
| AMD | ROCm 6.2.4 + ROCm PyTorch when required | amd64 | Explicit opt-in |
| Intel | oneAPI Base Toolkit + XPU PyTorch when required | amd64 | Explicit opt-in |

Every hardware class is structurally modeled against every public profile.
Accelerator profiles are not considered physically proven merely because their
container or bundle builds: `kait smoke` requires the matching device for
profiles that include the data-science PyTorch contract.

## Diagnostics and observability

`kait doctor` reports the image version, profile, baked capabilities, available
checks, expected hardware, detected evidence, and a `satisfied` hardware
result. `kait smoke` runs representative bounded programs for every advertised
capability and validates the accelerator relationship when required.

The supervisor exposes `/healthz`, `/readyz`, and `/metrics`, emits structured
JSON events on stderr, and optionally sends DogStatsD metrics. Collector
credentials remain outside the image.

## Deployment and release

The Docker launcher chooses `hardware-profile` tags and forwards the profile
assertion. Kubernetes uses the same identity-derived tags and profile values.
Buildkite jobs run in the selected official container; the native worker bundle
path is reserved while Apple is inactive. Jobs do not pull or rebuild Kait
inside each step.

CI asks `kait matrix --active-only --runtime container` for the active Linux CPU
images. The opt-in accelerator job asks for inactive hardware rows only when
matching runner labels are intentionally enabled. Release publishes immutable
container `<version>-<hardware>-<profile>` tags. Apple MPS bundles remain
prepared in source but are not built or released while Apple is inactive;
Apple Silicon CPU users use the multi-architecture CPU images.

## Downstream derivation

Organizations can inherit an immutable profile image and add certificates,
internal packages, CLIs, configuration, or integrations. If a downstream
change alters dependencies or runtime assumptions behind a Kait capability,
the downstream owner must rerun the corresponding doctor/smoke checks and own
the resulting compatibility surface.

## Repository map

| Path | Responsibility |
| --- | --- |
| `cmd/kait/` | Supervisor, contract resolver, matrix generator, diagnostics, tests |
| `cmd/kait/capability-contract.json` | Authoritative hardware/profile/capability model |
| `Dockerfile` | Shared image construction and identity baking |
| `docker-bake.hcl` | Release-facing hardware/profile targets and aliases |
| `requirements/` | Leaf dependency manifests selected by the model |
| `deploy/` | Docker and Kubernetes direct-use paths |
| `deploy/macos/` | Native Apple MPS bundle, installation, and direct-use paths |
| `examples/` | Heterogeneous Buildkite selector example |
| `.github/workflows/` | Generated-matrix image CI and release publication |
| `docs/` | Operator and architectural contract documentation |
