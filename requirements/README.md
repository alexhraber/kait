# Kaite Python layers

The Dockerfile installs one of two explicit variants. Every `*-slim` image
installs `slim.txt` and the selected hardware manifest. Every `*-full` image
then adds `base.txt`, `training.txt`, `orchestration.txt`, and `serving.txt`.
Installing the hardware manifest before the full shared layers keeps the
selected CPU, CUDA, ROCm, or XPU PyTorch build authoritative.

The active `cpu-slim`, `cpu-full`, `apple-slim`, and `apple-full` manifests use
the official CPU PyTorch index and contain Linux wheels for their supported
architectures. NVIDIA, AMD, and Intel manifests remain inactive until matching
host runners are enabled.
Packages that require a specific vendor kernel or custom native build, such as
DeepSpeed, bitsandbytes, FlashAttention, s3fs, and vLLM, are intentionally
operator extras through `KAITE_EXTRA_PYTHON_PACKAGES` rather than being
advertised as portable defaults.
