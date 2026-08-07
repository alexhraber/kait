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
  ]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAITE_HARDWARE = "cpu"
    KAITE_VARIANT = "slim"
  }
}

target "cpu" {
  inherits = ["cpu-slim"]
}

target "cpu-full" {
  inherits = ["common"]
  platforms = ["linux/amd64", "linux/arm64"]
  tags = ["${REGISTRY}:${VERSION}-cpu-full", "${REGISTRY}:cpu-full"]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAITE_HARDWARE = "cpu"
    KAITE_VARIANT = "full"
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
  ]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAITE_HARDWARE = "apple"
    KAITE_VARIANT = "slim"
  }
}

target "apple" {
  inherits = ["apple-slim"]
}

target "apple-full" {
  inherits = ["common"]
  platforms = ["linux/arm64"]
  tags = ["${REGISTRY}:${VERSION}-apple-full", "${REGISTRY}:apple-full"]
  args = {
    BASE_IMAGE = "ubuntu:24.04"
    KAITE_HARDWARE = "apple"
    KAITE_VARIANT = "full"
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
  ]
  args = {
    BASE_IMAGE = "nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04"
    KAITE_HARDWARE = "nvidia"
    KAITE_VARIANT = "slim"
  }
}

target "nvidia" {
  inherits = ["nvidia-slim"]
}

target "nvidia-full" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}:${VERSION}-nvidia-full", "${REGISTRY}:nvidia-full"]
  args = {
    BASE_IMAGE = "nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04"
    KAITE_HARDWARE = "nvidia"
    KAITE_VARIANT = "full"
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
  ]
  args = {
    BASE_IMAGE = "rocm/dev-ubuntu-24.04:6.2.4-complete"
    KAITE_HARDWARE = "amd"
    KAITE_VARIANT = "slim"
  }
}

target "amd" {
  inherits = ["amd-slim"]
}

target "amd-full" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}:${VERSION}-amd-full", "${REGISTRY}:amd-full"]
  args = {
    BASE_IMAGE = "rocm/dev-ubuntu-24.04:6.2.4-complete"
    KAITE_HARDWARE = "amd"
    KAITE_VARIANT = "full"
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
  ]
  args = {
    BASE_IMAGE = "intel/oneapi-basekit:2025.0.1-0-devel-ubuntu22.04"
    KAITE_HARDWARE = "intel"
    KAITE_VARIANT = "slim"
    KAITE_PYTHON = "python3.11"
  }
}

target "intel" {
  inherits = ["intel-slim"]
}

target "intel-full" {
  inherits = ["common"]
  platforms = ["linux/amd64"]
  tags = ["${REGISTRY}:${VERSION}-intel-full", "${REGISTRY}:intel-full"]
  args = {
    BASE_IMAGE = "intel/oneapi-basekit:2025.0.1-0-devel-ubuntu22.04"
    KAITE_HARDWARE = "intel"
    KAITE_VARIANT = "full"
    KAITE_PYTHON = "python3.11"
  }
}
