# Python layers

Every image installs `slim.txt` plus the selected hardware manifest
(`cpu.txt`, `apple.txt`, …). Full images then layer on `base.txt`,
`training.txt`, `orchestration.txt`, and `serving.txt`.

These layers are also Kaite's initial capability contract. Slim images declare
`data-science`; full images declare `data-science`, `training`,
`orchestration`, and `serving`. The image writes that declaration to
`/etc/kaite/identity.json`, and the supervisor smoke test imports representative
packages for every declared capability.

Hardware manifests install first so the chosen PyTorch wheel (CPU, CUDA,
ROCm, or XPU) wins over anything later on the default index.

| Manifest | Role |
| --- | --- |
| `slim.txt` | Compact data science + notebook baseline |
| `cpu.txt` / `apple.txt` | Official CPU PyTorch index (active) |
| `nvidia.txt` / `amd.txt` / `intel.txt` | Vendor PyTorch contracts (inactive in CI) |
| `base.txt` | Extra data/science utilities |
| `training.txt` | Hugging Face / Lightning training stack |
| `orchestration.txt` | Ray, MLflow, W&B |
| `serving.txt` | FastAPI, Gradio, Uvicorn |

Capability meanings are intentionally narrow:

| Capability | Promise |
| --- | --- |
| `data-science` | NumPy, pandas, scikit-learn, Jupyter, and the selected PyTorch hardware contract |
| `training` | The framework-neutral Hugging Face, Lightning, and model-training stack |
| `orchestration` | Ray execution plus MLflow and W&B experiment tooling |
| `serving` | FastAPI, Gradio, and Uvicorn application interfaces |

These are package and import contracts, not claims that a model, cluster, or
vendor runtime is configured for a particular organization. Vendor-specific
accelerator packages remain explicit opt-ins.

Packages that need a custom kernel or cloud client (DeepSpeed, bitsandbytes,
FlashAttention, vLLM, s3fs, …) are **not** defaults. Pass them at image build
time with `KAITE_EXTRA_PYTHON_PACKAGES`.
