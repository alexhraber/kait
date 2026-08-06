# syntax=docker/dockerfile:1.7
ARG BASE_IMAGE=ubuntu:24.04

FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS kaite-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY cmd/kaite ./cmd/kaite
RUN test -n "${TARGETOS}" -a -n "${TARGETARCH}" \
    && CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath -ldflags='-s -w' -o /out/kaite ./cmd/kaite

FROM ${BASE_IMAGE} AS runtime
ARG BUILDKITE_AGENT_VERSION=3.123.1
ARG KAITE_HARDWARE=cpu
ARG KAITE_PYTHON=python3
ARG KAITE_EXTRA_PYTHON_PACKAGES=""
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive \
    BUILDKITE_AGENT_HOME=/buildkite \
    BUILDKITE_AGENT_CONFIG=/buildkite/buildkite-agent.cfg \
    BUILDKITE_AGENT_HOOKS_PATH=/buildkite/hooks \
    BUILDKITE_BUILD_PATH=/buildkite/builds \
    KAITE_HARDWARE=${KAITE_HARDWARE} \
    KAITE_O11Y=none \
    PATH=/opt/kaite/venv/bin:/buildkite/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    PYTHONUNBUFFERED=1

USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       bash ca-certificates curl git jq python3 python3-venv build-essential procps tini \
    && rm -rf /var/lib/apt/lists/*

RUN if [ "${KAITE_PYTHON}" = "python3.11" ]; then \
      apt-get update \
      && apt-get install -y --no-install-recommends python3.11 python3.11-venv \
      && rm -rf /var/lib/apt/lists/*; \
    fi

RUN mkdir -p /buildkite/bin /buildkite/builds /buildkite/hooks /opt/kaite /tmp/buildkite-agent \
    && agent_arch="$(dpkg --print-architecture)" \
    && case "${agent_arch}" in amd64|arm64) ;; *) echo "unsupported Debian architecture: ${agent_arch}" >&2; exit 1 ;; esac \
    && archive="buildkite-agent-linux-${agent_arch}-${BUILDKITE_AGENT_VERSION}.tar.gz" \
    && curl -fsSL "https://github.com/buildkite/agent/releases/download/v${BUILDKITE_AGENT_VERSION}/${archive}" -o /tmp/buildkite-agent/agent.tgz \
    && curl -fsSL "https://github.com/buildkite/agent/releases/download/v${BUILDKITE_AGENT_VERSION}/buildkite-agent-${BUILDKITE_AGENT_VERSION}.SHA256SUMS" -o /tmp/buildkite-agent/SHA256SUMS \
    && expected="$(awk -v file="${archive}" '$2 == file || $2 == "*" file {print $1; exit}' /tmp/buildkite-agent/SHA256SUMS)" \
    && test -n "${expected}" \
    && echo "${expected}  /tmp/buildkite-agent/agent.tgz" | sha256sum -c - \
    && mkdir -p /tmp/buildkite-agent/extracted \
    && tar -xzf /tmp/buildkite-agent/agent.tgz -C /tmp/buildkite-agent/extracted \
    && install -m 0755 /tmp/buildkite-agent/extracted/buildkite-agent /buildkite/bin/buildkite-agent \
    && install -m 0755 /tmp/buildkite-agent/extracted/bootstrap.sh /buildkite/bootstrap.sh \
    && install -m 0644 /tmp/buildkite-agent/extracted/buildkite-agent.cfg /buildkite/buildkite-agent.cfg \
    && rm -rf /tmp/buildkite-agent

COPY requirements /opt/kaite/requirements
RUN "${KAITE_PYTHON}" -m venv /opt/kaite/venv \
    && /opt/kaite/venv/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/kaite/venv/bin/pip install --no-cache-dir -r /opt/kaite/requirements/base.txt \
    && if [ -f "/opt/kaite/requirements/${KAITE_HARDWARE}.txt" ]; then /opt/kaite/venv/bin/pip install --no-cache-dir -r "/opt/kaite/requirements/${KAITE_HARDWARE}.txt"; fi \
    && if [ -n "${KAITE_EXTRA_PYTHON_PACKAGES}" ]; then /opt/kaite/venv/bin/pip install --no-cache-dir ${KAITE_EXTRA_PYTHON_PACKAGES}; fi

COPY --from=kaite-build /out/kaite /usr/local/bin/kaite
RUN chmod +x /usr/local/bin/kaite \
    && touch /buildkite/buildkite-agent.cfg

WORKDIR /buildkite/builds
EXPOSE 9090
VOLUME ["/buildkite/builds"]
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/kaite"]
