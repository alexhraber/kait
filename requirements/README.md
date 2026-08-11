# Python layers

Every image installs `slim.txt` plus the selected hardware manifest
(`cpu.txt`, `apple.txt`, …). Full images then layer on `base.txt`,
`training.txt`, `orchestration.txt`, and `serving.txt`.

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

Packages that need a custom kernel or cloud client (DeepSpeed, bitsandbytes,
FlashAttention, vLLM, s3fs, …) are **not** defaults. Pass them at image build
time with `KAITE_EXTRA_PYTHON_PACKAGES`.
