# Kaite capability contract

Kaite's hardware names describe where an image can run. Its capability names
describe the work the image is prepared to perform. They are separate
contracts:

```text
image build → baked identity → supervisor validation → Buildkite tags → pipeline selector
       └────────────── requirements and smoke checks ────────────────┘
```

## Current capabilities

The first contract is intentionally the set already represented by the
repository's requirement layers:

| Capability | Declared by | Representative proof |
| --- | --- | --- |
| `data-science` | Every official image | NumPy, scikit-learn, and PyTorch imports; selected hardware check |
| `training` | Full images | Accelerate, datasets, Diffusers, Lightning, and Transformers imports |
| `orchestration` | Full images | MLflow, Ray, and W&B imports |
| `serving` | Full images | FastAPI, Gradio, and Uvicorn imports |

These are strong floors, not promises that a model is downloaded, a cluster is
available, or a vendor-specific serving stack is installed. DeepSpeed,
bitsandbytes, FlashAttention, vLLM, s3fs, and similar compatibility-sensitive
packages remain downstream choices.

## Image identity

The Docker build writes `/etc/kaite/identity.json` with:

```json
{
  "schema": 1,
  "hardware": "cpu",
  "variant": "full",
  "capabilities": ["data-science", "training", "orchestration", "serving"]
}
```

The identity is derived from build arguments and is not selected by a pipeline
or changed by an ordinary runtime environment override. `KAITE_HARDWARE`,
`KAITE_VARIANT`, and, when supplied, `KAITE_CAPABILITIES` must agree with the
baked file. A mismatch fails closed before the Buildkite agent starts.

`kaite doctor` reports the identity and detected device commands. `kaite smoke`
executes representative imports for every declared capability and performs the
accelerator check for NVIDIA, AMD, or Intel images. CPU and Apple images do not
claim Apple Metal passthrough; Apple is a Linux/arm64 CPU contract here.

## Buildkite targeting

The supervisor advertises one boolean tag for each declared capability:

```text
kaite=true
kaite.hardware=cpu
kaite.variant=full
kaite.capability.data-science=true
kaite.capability.training=true
kaite.capability.orchestration=true
kaite.capability.serving=true
```

A pipeline requests the capability it needs with ordinary Buildkite matching:

```yaml
steps:
  - label: ":brain: train"
    command: "python train.py"
    agents:
      queue: ai
      kaite.hardware: cpu
      kaite.capability.training: "true"
```

There is no Kaite-specific pipeline parser or scheduler. `kaite.*` is a
reserved tag namespace produced by the image identity. User-supplied custom
tags are still supported, but an attempt to replace a canonical Kaite tag is a
configuration error.

## Compatibility and future scope

`slim` and `full` remain stable image and Bake names. They are compatibility
names for the current package footprints, not the preferred workload selector.
The current full artifact intentionally exposes all four supported capability
tags; this avoids multiplying the release matrix while preserving a coherent
AI/ML floor.

Focused `training` or `serving` artifacts can be added later by reusing the
existing capability composition and identity contract. They should only be
introduced when their smaller dependency sets and release/CI proof are worth a
new artifact. Agentic execution, inference-specific servers, and distributed
training are deferred until Kaite has dependencies and tests that justify
those stronger claims.

## Downstream derivation

Organizations can inherit from an immutable versioned image and add internal
packages, certificates, CLIs, model libraries, security tooling, or
configuration with ordinary Docker. The inherited identity remains the common
Kaite contract. If the derived image replaces a package covered by a
capability, the organization owns the new validation and should not reuse the
official capability claim without re-running the smoke checks.
