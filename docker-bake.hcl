variable "REGISTRY" {
  default = "ghcr.io/alexhraber/kaite"
}

variable "VERSION" {
  default = "dev"
}

variable "AGENT_VERSION" {
  default = "3.123.1"
}

group "default" {
  targets = ["cpu"]
}

target "common" {
  context = "."
  dockerfile = "Dockerfile"
  args = {
    BUILDKITE_AGENT_VERSION = AGENT_VERSION
  }
}

target "cpu" {
  inherits = ["common"]
  platforms = ["linux/amd64", "linux/arm64"]
  tags = ["${REGISTRY}:cpu-${VERSION}", "${REGISTRY}:cpu"]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAITE_HARDWARE = "cpu"
  }
}

target "apple" {
  inherits = ["common"]
  platforms = ["linux/arm64"]
  tags = ["${REGISTRY}:apple-${VERSION}", "${REGISTRY}:apple"]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAITE_HARDWARE = "apple"
  }
}

target "nvidia" {
  # Inactive in automatic CI and release workflows; invoke explicitly when a
  # matching host is intentionally enabled.
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}:nvidia-${VERSION}", "${REGISTRY}:nvidia"]
  args = {
    BASE_IMAGE = "nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04"
    KAITE_HARDWARE = "nvidia"
  }
}

target "amd" {
  # Inactive in automatic CI and release workflows; invoke explicitly when a
  # matching host is intentionally enabled.
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}:amd-${VERSION}", "${REGISTRY}:amd"]
  args = {
    BASE_IMAGE = "rocm/dev-ubuntu-24.04:6.2.4-complete"
    KAITE_HARDWARE = "amd"
  }
}

target "intel" {
  # Inactive in automatic CI and release workflows; invoke explicitly when a
  # matching host is intentionally enabled.
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}:intel-${VERSION}", "${REGISTRY}:intel"]
  args = {
    BASE_IMAGE = "intel/oneapi-basekit:2025.0.1-0-devel-ubuntu22.04"
    KAITE_HARDWARE = "intel"
    KAITE_PYTHON = "python3.11"
  }
}
