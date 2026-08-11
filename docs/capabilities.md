---
layout: default
title: Capability contract
description: The workload and hardware capability contract exposed by Kait images.
---

# Kait capability contract

`cmd/kait/capability-contract.json` is Kait's single capability model. The
embedded Go supervisor, Dockerfile, smoke checks, Buildkite tags, and CI/release
matrix all consume the same definitions. A package appearing in a manifest is
not sufficient: the profile must also have a baked identity, a runtime check,
and a schedulable tag.

## Official profiles and capabilities

The public image profiles are `slim`, `full`, and the individual workload
profiles `data-science`, `training`, `orchestration`, and `serving`.

| Profile | Baked workload capabilities | Runtime contract |
| --- | --- | --- |
| `slim` | `data-science` | Compatibility profile for the compact data-science environment. |
| `full` | all four | Compatibility profile composing the complete AI/ML environment. |
| `data-science` | `data-science` | NumPy, pandas, scikit-learn, Jupyter, and hardware-specific PyTorch. |
| `training` | `data-science`, `training` | Data-science foundation plus Hugging Face and Lightning tooling. |
| `orchestration` | `orchestration` | Ray execution with MLflow and Weights & Biases tooling. |
| `serving` | `serving` | FastAPI, Gradio, and Uvicorn application interfaces. |

`training` intentionally composes `data-science`. `orchestration` and `serving`
remain independent so their selectors convey meaningful, smaller environments;
`full` composes all four. The profile is an image identity field, while the
workload capabilities are the scheduler-facing contract.

## Hardware dimension

Hardware remains separate from workload capability:

| Hardware | Runtime contract | Current build/release status |
| --- | --- | --- |
| `cpu` | Ubuntu 24.04, CPU PyTorch, amd64/arm64 | Active |
| `apple` | Ubuntu arm64 CPU execution; no Metal passthrough claim | Active |
| `nvidia` | CUDA 12.6.3 and CUDA PyTorch | Explicit opt-in; matching host required for accelerator smoke |
| `amd` | ROCm 6.2.4 and ROCm PyTorch | Explicit opt-in; matching host required for accelerator smoke |
| `intel` | oneAPI Base Toolkit and XPU PyTorch | Explicit opt-in; matching host required for accelerator smoke |

The model validates every advertised profile/hardware pair structurally. It
does not turn a missing physical accelerator into a passing hardware result:
`kait doctor` reports the host evidence and `kait smoke` fails when a profile
that includes data-science cannot see its required accelerator.

## Baked image identity

Every official image contains `/etc/kait/identity.json`, produced by the
embedded contract resolver during the Docker build:

```json
{
  "schema": 2,
  "hardware": "cpu",
  "variant": "full",
  "profile": "training",
  "capabilities": ["data-science", "training"],
  "requirements": ["cpu.txt", "slim.txt", "base.txt", "training.txt"]
}
```

The supervisor refuses to start without that baked file. `KAIT_HARDWARE`,
`KAIT_VARIANT`, `KAIT_PROFILE`, and `KAIT_CAPABILITIES` may assert or constrain
the identity, but cannot create or replace it. Conflicts fail closed before
the Buildkite agent starts. The resolver also checks that the manifest list
and profile composition agree, preventing an image from claiming packages it
did not install.

## Buildkite targeting

The agent tags derive only from the validated identity:

```text
kait=true
kait.hardware=nvidia
kait.variant=full
kait.profile=training
kait.capability.data-science=true
kait.capability.training=true
```

Pipelines select ordinary Buildkite tags without knowing image names or host
names:

```yaml
steps:
  - label: ":brain: train"
    command: "python train.py"
    agents:
      queue: ai
      kait.hardware: nvidia
      kait.capability.training: "true"
```

`BUILDKITE_AGENT_TAGS` can add organization-specific tags, but the reserved
`kait` namespace cannot be overridden. Dynamic pipeline uploads use the same
selectors because Kait does not assume that the execution graph was known at
image-build time.

## Diagnostics and proof

`kait doctor` reports the image version, profile, baked capabilities, the
available capability checks, expected hardware, detected hardware evidence,
and whether the host satisfies the image's hardware expectation.

`kait smoke` runs the representative checks encoded in the capability model:

- data-science imports and uses NumPy, pandas, scikit-learn, Jupyter, and PyTorch;
- training constructs a Hugging Face Dataset and TrainingArguments and checks a Lightning module;
- orchestration runs a bounded local Ray task, a local MLflow metric, and disabled-mode W&B;
- serving constructs a FastAPI route, Gradio interface, and Uvicorn configuration.

No model download, credential, remote MLflow service, W&B login, or external
experiment is required.

## Direct use and derivation

Use an immutable profile/hardware image directly as a self-hosted Buildkite
worker, or derive from one:

```dockerfile
FROM ghcr.io/alexhraber/kait:<immutable-release>-cpu-training

COPY internal-certificates/ /usr/local/share/ca-certificates/
RUN update-ca-certificates
RUN pip install --no-cache-dir internal-model-tools
```

The inherited identity remains understandable after derivation. If a
downstream image changes dependencies underpinning an advertised capability,
its owner must rerun the relevant doctor/smoke checks and own the resulting
compatibility surface rather than silently retaining an invalid claim.
