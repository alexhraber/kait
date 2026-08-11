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
  }
}

target "cpu-slim" {
  inherits = ["common"]
  platforms = ["linux/amd64", "linux/arm64"]
  tags = [
    "${REGISTRY}:${VERSION}-cpu-slim",
    "${REGISTRY}:cpu-slim",
    "${REGISTRY}:${VERSION}-cpu",
    "${REGISTRY}:cpu",
    "${REGISTRY}:${VERSION}-cpu-data-science",
    "${REGISTRY}:cpu-data-science",
  ]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAIT_HARDWARE = "cpu"
    KAIT_VARIANT = "slim"
    KAIT_CAPABILITIES = "data-science"
  }
}

target "cpu" {
  inherits = ["cpu-slim"]
}

target "cpu-full" {
  inherits = ["common"]
  platforms = ["linux/amd64", "linux/arm64"]
  tags = [
    "${REGISTRY}:${VERSION}-cpu-full",
    "${REGISTRY}:cpu-full",
    "${REGISTRY}:${VERSION}-cpu-training",
    "${REGISTRY}:cpu-training",
    "${REGISTRY}:${VERSION}-cpu-orchestration",
    "${REGISTRY}:cpu-orchestration",
    "${REGISTRY}:${VERSION}-cpu-serving",
    "${REGISTRY}:cpu-serving",
  ]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAIT_HARDWARE = "cpu"
    KAIT_VARIANT = "full"
    KAIT_CAPABILITIES = "data-science,training,orchestration,serving"
  }
}

target "apple-slim" {
  inherits = ["common"]
  platforms = ["linux/arm64"]
  tags = [
    "${REGISTRY}:${VERSION}-apple-slim",
    "${REGISTRY}:apple-slim",
    "${REGISTRY}:${VERSION}-apple",
    "${REGISTRY}:apple",
    "${REGISTRY}:${VERSION}-apple-data-science",
    "${REGISTRY}:apple-data-science",
  ]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAIT_HARDWARE = "apple"
    KAIT_VARIANT = "slim"
    KAIT_CAPABILITIES = "data-science"
  }
}

target "apple" {
  inherits = ["apple-slim"]
}

target "apple-full" {
  inherits = ["common"]
  platforms = ["linux/arm64"]
  tags = [
    "${REGISTRY}:${VERSION}-apple-full",
    "${REGISTRY}:apple-full",
    "${REGISTRY}:${VERSION}-apple-training",
    "${REGISTRY}:apple-training",
    "${REGISTRY}:${VERSION}-apple-orchestration",
    "${REGISTRY}:apple-orchestration",
    "${REGISTRY}:${VERSION}-apple-serving",
    "${REGISTRY}:apple-serving",
  ]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAIT_HARDWARE = "apple"
    KAIT_VARIANT = "full"
    KAIT_CAPABILITIES = "data-science,training,orchestration,serving"
  }
}

target "nvidia-slim" {
  # Inactive in automatic CI and release workflows; invoke explicitly when a
  # matching host is intentionally enabled.
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = [
    "${REGISTRY}:${VERSION}-nvidia-slim",
    "${REGISTRY}:nvidia-slim",
    "${REGISTRY}:${VERSION}-nvidia",
    "${REGISTRY}:nvidia",
    "${REGISTRY}:${VERSION}-nvidia-data-science",
    "${REGISTRY}:nvidia-data-science",
  ]
  args = {
    BASE_IMAGE = "nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04"
    KAIT_HARDWARE = "nvidia"
    KAIT_VARIANT = "slim"
    KAIT_CAPABILITIES = "data-science"
  }
}

target "nvidia" {
  inherits = ["nvidia-slim"]
}

target "nvidia-full" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = [
    "${REGISTRY}:${VERSION}-nvidia-full",
    "${REGISTRY}:nvidia-full",
    "${REGISTRY}:${VERSION}-nvidia-training",
    "${REGISTRY}:nvidia-training",
    "${REGISTRY}:${VERSION}-nvidia-orchestration",
    "${REGISTRY}:nvidia-orchestration",
    "${REGISTRY}:${VERSION}-nvidia-serving",
    "${REGISTRY}:nvidia-serving",
  ]
  args = {
    BASE_IMAGE = "nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04"
    KAIT_HARDWARE = "nvidia"
    KAIT_VARIANT = "full"
    KAIT_CAPABILITIES = "data-science,training,orchestration,serving"
  }
}

target "amd-slim" {
  # Inactive in automatic CI and release workflows; invoke explicitly when a
  # matching host is intentionally enabled.
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = [
    "${REGISTRY}:${VERSION}-amd-slim",
    "${REGISTRY}:amd-slim",
    "${REGISTRY}:${VERSION}-amd",
    "${REGISTRY}:amd",
    "${REGISTRY}:${VERSION}-amd-data-science",
    "${REGISTRY}:amd-data-science",
  ]
  args = {
    BASE_IMAGE = "rocm/dev-ubuntu-24.04:6.2.4-complete"
    KAIT_HARDWARE = "amd"
    KAIT_VARIANT = "slim"
    KAIT_CAPABILITIES = "data-science"
  }
}

target "amd" {
  inherits = ["amd-slim"]
}

target "amd-full" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = [
    "${REGISTRY}:${VERSION}-amd-full",
    "${REGISTRY}:amd-full",
    "${REGISTRY}:${VERSION}-amd-training",
    "${REGISTRY}:amd-training",
    "${REGISTRY}:${VERSION}-amd-orchestration",
    "${REGISTRY}:amd-orchestration",
    "${REGISTRY}:${VERSION}-amd-serving",
    "${REGISTRY}:amd-serving",
  ]
  args = {
    BASE_IMAGE = "rocm/dev-ubuntu-24.04:6.2.4-complete"
    KAIT_HARDWARE = "amd"
    KAIT_VARIANT = "full"
    KAIT_CAPABILITIES = "data-science,training,orchestration,serving"
  }
}

target "intel-slim" {
  # Inactive in automatic CI and release workflows; invoke explicitly when a
  # matching host is intentionally enabled.
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = [
    "${REGISTRY}:${VERSION}-intel-slim",
    "${REGISTRY}:intel-slim",
    "${REGISTRY}:${VERSION}-intel",
    "${REGISTRY}:intel",
    "${REGISTRY}:${VERSION}-intel-data-science",
    "${REGISTRY}:intel-data-science",
  ]
  args = {
    BASE_IMAGE = "intel/oneapi-basekit:2025.0.1-0-devel-ubuntu22.04"
    KAIT_HARDWARE = "intel"
    KAIT_VARIANT = "slim"
    KAIT_CAPABILITIES = "data-science"
    KAIT_PYTHON = "python3.11"
  }
}

target "intel" {
  inherits = ["intel-slim"]
}

target "intel-full" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = [
    "${REGISTRY}:${VERSION}-intel-full",
    "${REGISTRY}:intel-full",
    "${REGISTRY}:${VERSION}-intel-training",
    "${REGISTRY}:intel-training",
    "${REGISTRY}:${VERSION}-intel-orchestration",
    "${REGISTRY}:intel-orchestration",
    "${REGISTRY}:${VERSION}-intel-serving",
    "${REGISTRY}:intel-serving",
  ]
  args = {
    BASE_IMAGE = "intel/oneapi-basekit:2025.0.1-0-devel-ubuntu22.04"
    KAIT_HARDWARE = "intel"
    KAIT_VARIANT = "full"
    KAIT_CAPABILITIES = "data-science,training,orchestration,serving"
    KAIT_PYTHON = "python3.11"
  }
}
