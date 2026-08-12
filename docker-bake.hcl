variable "REGISTRY" {
  default = "ghcr.io/alexhraber/kait"
}

variable "VERSION" {
  default = "dev"
}

variable "AGENT_VERSION" {
  default = "3.123.1"
}

group "default" {
  targets = ["cpu-slim"]
}

target "common" {
  context = "."
  dockerfile = "Dockerfile"
  args = {
    BUILDKITE_AGENT_VERSION = AGENT_VERSION
    KAIT_VERSION = VERSION
  }
}

# The authoritative profile and hardware definitions live in
# cmd/kait/capability-contract.json. These targets are the release-facing
# Docker Bake projection of that contract; CI checks the same model through
# `kait matrix` before building.

target "cpu-base" {
  inherits = ["common"]
  platforms = ["linux/amd64", "linux/arm64"]
  args = { BASE_IMAGE = "ubuntu:24.04", KAIT_HARDWARE = "cpu", KAIT_PYTHON = "python3" }
}

target "nvidia-base" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  args = { BASE_IMAGE = "nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04", KAIT_HARDWARE = "nvidia", KAIT_PYTHON = "python3" }
}

target "amd-base" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  args = { BASE_IMAGE = "rocm/dev-ubuntu-24.04:6.2.4-complete", KAIT_HARDWARE = "amd", KAIT_PYTHON = "python3" }
}

target "intel-base" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  args = { BASE_IMAGE = "intel/oneapi-basekit:2025.0.1-0-devel-ubuntu22.04", KAIT_HARDWARE = "intel", KAIT_PYTHON = "python3.11" }
}

target "cpu-slim" {
  inherits = ["cpu-base"]
  tags = ["${REGISTRY}:${VERSION}-cpu-slim", "${REGISTRY}:cpu-slim", "${REGISTRY}:${VERSION}-cpu", "${REGISTRY}:cpu"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "slim", KAIT_CAPABILITIES = "data-science" }
}
target "cpu" { inherits = ["cpu-slim"] }
target "cpu-full" {
  inherits = ["cpu-base"]
  tags = ["${REGISTRY}:${VERSION}-cpu-full", "${REGISTRY}:cpu-full"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "full", KAIT_CAPABILITIES = "data-science,training,orchestration,serving" }
}
target "cpu-data-science" {
  inherits = ["cpu-base"]
  tags = ["${REGISTRY}:${VERSION}-cpu-data-science", "${REGISTRY}:cpu-data-science"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "data-science", KAIT_CAPABILITIES = "data-science" }
}
target "cpu-training" {
  inherits = ["cpu-base"]
  tags = ["${REGISTRY}:${VERSION}-cpu-training", "${REGISTRY}:cpu-training"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "training", KAIT_CAPABILITIES = "data-science,training" }
}
target "cpu-orchestration" {
  inherits = ["cpu-base"]
  tags = ["${REGISTRY}:${VERSION}-cpu-orchestration", "${REGISTRY}:cpu-orchestration"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "orchestration", KAIT_CAPABILITIES = "orchestration" }
}
target "cpu-serving" {
  inherits = ["cpu-base"]
  tags = ["${REGISTRY}:${VERSION}-cpu-serving", "${REGISTRY}:cpu-serving"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "serving", KAIT_CAPABILITIES = "serving" }
}

# Accelerator targets are explicit opt-ins. They use the same six profiles and
# are only scheduled by workflows when matching hosts are deliberately enabled.
target "nvidia-slim" {
  inherits = ["nvidia-base"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-slim", "${REGISTRY}:nvidia-slim", "${REGISTRY}:${VERSION}-nvidia", "${REGISTRY}:nvidia"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "slim", KAIT_CAPABILITIES = "data-science" }
}
target "nvidia" { inherits = ["nvidia-slim"] }
target "nvidia-full" {
  inherits = ["nvidia-base"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-full", "${REGISTRY}:nvidia-full"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "full", KAIT_CAPABILITIES = "data-science,training,orchestration,serving" }
}
target "nvidia-data-science" {
  inherits = ["nvidia-base"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-data-science", "${REGISTRY}:nvidia-data-science"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "data-science", KAIT_CAPABILITIES = "data-science" }
}
target "nvidia-training" {
  inherits = ["nvidia-base"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-training", "${REGISTRY}:nvidia-training"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "training", KAIT_CAPABILITIES = "data-science,training" }
}
target "nvidia-orchestration" {
  inherits = ["nvidia-base"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-orchestration", "${REGISTRY}:nvidia-orchestration"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "orchestration", KAIT_CAPABILITIES = "orchestration" }
}
target "nvidia-serving" {
  inherits = ["nvidia-base"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-serving", "${REGISTRY}:nvidia-serving"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "serving", KAIT_CAPABILITIES = "serving" }
}

target "amd-slim" {
  inherits = ["amd-base"]
  tags = ["${REGISTRY}:${VERSION}-amd-slim", "${REGISTRY}:amd-slim", "${REGISTRY}:${VERSION}-amd", "${REGISTRY}:amd"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "slim", KAIT_CAPABILITIES = "data-science" }
}
target "amd" { inherits = ["amd-slim"] }
target "amd-full" {
  inherits = ["amd-base"]
  tags = ["${REGISTRY}:${VERSION}-amd-full", "${REGISTRY}:amd-full"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "full", KAIT_CAPABILITIES = "data-science,training,orchestration,serving" }
}
target "amd-data-science" {
  inherits = ["amd-base"]
  tags = ["${REGISTRY}:${VERSION}-amd-data-science", "${REGISTRY}:amd-data-science"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "data-science", KAIT_CAPABILITIES = "data-science" }
}
target "amd-training" {
  inherits = ["amd-base"]
  tags = ["${REGISTRY}:${VERSION}-amd-training", "${REGISTRY}:amd-training"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "training", KAIT_CAPABILITIES = "data-science,training" }
}
target "amd-orchestration" {
  inherits = ["amd-base"]
  tags = ["${REGISTRY}:${VERSION}-amd-orchestration", "${REGISTRY}:amd-orchestration"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "orchestration", KAIT_CAPABILITIES = "orchestration" }
}
target "amd-serving" {
  inherits = ["amd-base"]
  tags = ["${REGISTRY}:${VERSION}-amd-serving", "${REGISTRY}:amd-serving"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "serving", KAIT_CAPABILITIES = "serving" }
}

target "intel-slim" {
  inherits = ["intel-base"]
  tags = ["${REGISTRY}:${VERSION}-intel-slim", "${REGISTRY}:intel-slim", "${REGISTRY}:${VERSION}-intel", "${REGISTRY}:intel"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "slim", KAIT_CAPABILITIES = "data-science" }
}
target "intel" { inherits = ["intel-slim"] }
target "intel-full" {
  inherits = ["intel-base"]
  tags = ["${REGISTRY}:${VERSION}-intel-full", "${REGISTRY}:intel-full"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "full", KAIT_CAPABILITIES = "data-science,training,orchestration,serving" }
}
target "intel-data-science" {
  inherits = ["intel-base"]
  tags = ["${REGISTRY}:${VERSION}-intel-data-science", "${REGISTRY}:intel-data-science"]
  args = { KAIT_VARIANT = "slim", KAIT_PROFILE = "data-science", KAIT_CAPABILITIES = "data-science" }
}
target "intel-training" {
  inherits = ["intel-base"]
  tags = ["${REGISTRY}:${VERSION}-intel-training", "${REGISTRY}:intel-training"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "training", KAIT_CAPABILITIES = "data-science,training" }
}
target "intel-orchestration" {
  inherits = ["intel-base"]
  tags = ["${REGISTRY}:${VERSION}-intel-orchestration", "${REGISTRY}:intel-orchestration"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "orchestration", KAIT_CAPABILITIES = "orchestration" }
}
target "intel-serving" {
  inherits = ["intel-base"]
  tags = ["${REGISTRY}:${VERSION}-intel-serving", "${REGISTRY}:intel-serving"]
  args = { KAIT_VARIANT = "full", KAIT_PROFILE = "serving", KAIT_CAPABILITIES = "serving" }
}
