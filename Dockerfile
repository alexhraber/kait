# syntax=docker/dockerfile:1.7
ARG BASE_IMAGE=ubuntu:24.04

FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS kait-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY cmd/kait ./cmd/kait
RUN test -n "${TARGETOS}" -a -n "${TARGETARCH}" \
    && CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath -ldflags='-s -w' -o /out/kait ./cmd/kait

FROM ${BASE_IMAGE} AS runtime
ARG BUILDKITE_AGENT_VERSION=3.123.1
ARG KAIT_HARDWARE=cpu
ARG KAIT_VARIANT=
ARG KAIT_PROFILE=
ARG KAIT_CAPABILITIES=
ARG KAIT_PYTHON=python3
ARG KAIT_EXTRA_PYTHON_PACKAGES=""
ARG TARGETARCH

LABEL io.kait.identity="/etc/kait/identity.json" \
      io.kait.hardware="${KAIT_HARDWARE}" \
      io.kait.variant="${KAIT_VARIANT}" \
      io.kait.profile="${KAIT_PROFILE}" \
      io.kait.capabilities="${KAIT_CAPABILITIES}"

ENV DEBIAN_FRONTEND=noninteractive \
    BUILDKITE_AGENT_HOME=/buildkite \
    BUILDKITE_AGENT_CONFIG=/buildkite/buildkite-agent.cfg \
    BUILDKITE_AGENT_HOOKS_PATH=/buildkite/hooks \
    BUILDKITE_BUILD_PATH=/buildkite/builds \
    KAIT_HARDWARE=${KAIT_HARDWARE} \
    KAIT_VARIANT=${KAIT_VARIANT} \
    KAIT_PROFILE=${KAIT_PROFILE} \
    KAIT_O11Y=none \
    PATH=/opt/kait/venv/bin:/buildkite/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    PYTHONUNBUFFERED=1

USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       bash ca-certificates curl git jq python3 python3-venv build-essential procps tini \
    && rm -rf /var/lib/apt/lists/*

RUN if [ "${KAIT_PYTHON}" = "python3.11" ]; then \
      apt-get update \
      && apt-get install -y --no-install-recommends python3.11 python3.11-venv \
      && rm -rf /var/lib/apt/lists/*; \
    fi

RUN mkdir -p /buildkite/bin /buildkite/builds /buildkite/hooks /etc/kait /opt/kait /tmp/buildkite-agent \
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

COPY --from=kait-build /out/kait /usr/local/bin/kait
COPY requirements /opt/kait/requirements
RUN set -eux; \
    "${KAIT_PYTHON}" -m venv /opt/kait/venv; \
    /opt/kait/venv/bin/pip install --no-cache-dir --upgrade pip; \
    contract_args="--hardware ${KAIT_HARDWARE}"; \
    if [ -n "${KAIT_PROFILE}" ]; then contract_args="${contract_args} --profile ${KAIT_PROFILE}"; fi; \
    if [ -n "${KAIT_VARIANT}" ]; then contract_args="${contract_args} --variant ${KAIT_VARIANT}"; fi; \
    if [ -n "${KAIT_CAPABILITIES}" ]; then contract_args="${contract_args} --capabilities ${KAIT_CAPABILITIES}"; fi; \
    /usr/local/bin/kait contract ${contract_args} > /tmp/kait-contract.json; \
    jq -e '.schema == 2 and (.hardware | length > 0) and (.profile | length > 0) and (.capabilities | length > 0)' /tmp/kait-contract.json >/dev/null; \
    jq -r '.requirements[]' /tmp/kait-contract.json | while IFS= read -r requirements_file; do \
      test -f "/opt/kait/requirements/${requirements_file}" || { echo "missing requirements manifest: ${requirements_file}" >&2; exit 1; }; \
      /opt/kait/venv/bin/pip install --no-cache-dir -r "/opt/kait/requirements/${requirements_file}"; \
    done; \
    if [ -n "${KAIT_EXTRA_PYTHON_PACKAGES}" ]; then \
      /opt/kait/venv/bin/pip install --no-cache-dir ${KAIT_EXTRA_PYTHON_PACKAGES}; \
    fi; \
    cp /tmp/kait-contract.json /etc/kait/identity.json

RUN chmod +x /usr/local/bin/kait \
    && touch /buildkite/buildkite-agent.cfg

WORKDIR /buildkite/builds
EXPOSE 9090
VOLUME ["/buildkite/builds"]
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/kait"]
