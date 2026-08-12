# Python layers

`cmd/kait/capability-contract.json` is the authoritative composition model.
The Dockerfile asks the embedded `kait contract` command to resolve a container
profile into an ordered manifest list, writes the resulting identity, and
installs only those manifests. The reserved native Apple path uses the same
resolver when re-enabled. The same model drives runtime smoke checks,
Buildkite tags, and the CI/release matrix.

The public profiles are:

| Profile | Capabilities | Ordered manifests |
| --- | --- | --- |
| `slim` | `data-science` | hardware PyTorch, then `slim.txt` |
| `full` | all four capabilities | hardware PyTorch, `slim.txt`, `base.txt`, `training.txt`, `orchestration.txt`, `serving.txt` |
| `data-science` | `data-science` | hardware PyTorch, then `slim.txt` |
| `training` | `data-science`, `training` | hardware PyTorch, `slim.txt`, `base.txt`, `training.txt` |
| `orchestration` | `orchestration` | `orchestration.txt` |
| `serving` | `serving` | `serving.txt` |

The `slim` and `full` names remain compatibility profiles. Workload selectors
should use `kait.capability.<name>=true`; the profile tag remains available for
image and operator inspection.

## Capability promises

| Capability | Promise and representative proof |
| --- | --- |
| `data-science` | NumPy, pandas, scikit-learn, Jupyter, and the selected hardware PyTorch contract; smoke constructs representative data objects and validates the accelerator relationship. |
| `training` | Hugging Face Dataset/TrainingArguments and Lightning classes without model downloads or network-backed experiments. It intentionally composes `data-science`. |
| `orchestration` | A local Ray task, a file-backed MLflow run, and disabled-mode W&B initialization. |
| `serving` | FastAPI route construction, a Gradio interface, and a Uvicorn configuration. |

Hardware manifests install only when the resolved profile requires the
data-science PyTorch contract. The hardware class is still part of the baked
identity for every profile, even when a serving or orchestration profile does
not need a vendor Python wheel.

`cpu.txt`, `apple-mps.txt`, `apple.txt`, `nvidia.txt`, `amd.txt`, and `intel.txt`
preserve the vendor-specific PyTorch choices. `apple-mps.txt` is reserved for
the inactive native macOS arm64 contract; it is validated with
`torch.backends.mps` when that path is re-enabled. `apple.txt` remains the
manifest name used by already-published Linux Apple CPU images. Apple, NVIDIA,
AMD, and Intel remain explicit accelerator opt-ins until matching Buildkite
hosts are deliberately enabled for hardware smoke proof.

Packages that need a custom kernel or cloud client (DeepSpeed, bitsandbytes,
FlashAttention, vLLM, s3fs, and similar) are not defaults. Pass them at image
build time with `KAIT_EXTRA_PYTHON_PACKAGES`; downstream owners must rerun the
corresponding `kait doctor` and `kait smoke` checks if those packages change a
Kait capability contract.
