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
ARG KAIT_VARIANT=slim
ARG KAIT_CAPABILITIES=
ARG KAIT_PYTHON=python3
ARG KAIT_EXTRA_PYTHON_PACKAGES=""
ARG TARGETARCH

LABEL io.kait.identity="/etc/kait/identity.json" \
      io.kait.hardware="${KAIT_HARDWARE}" \
      io.kait.variant="${KAIT_VARIANT}" \
      io.kait.capabilities="${KAIT_CAPABILITIES}"

ENV DEBIAN_FRONTEND=noninteractive \
    BUILDKITE_AGENT_HOME=/buildkite \
    BUILDKITE_AGENT_CONFIG=/buildkite/buildkite-agent.cfg \
    BUILDKITE_AGENT_HOOKS_PATH=/buildkite/hooks \
    BUILDKITE_BUILD_PATH=/buildkite/builds \
    KAIT_HARDWARE=${KAIT_HARDWARE} \
    KAIT_VARIANT=${KAIT_VARIANT} \
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

COPY requirements /opt/kait/requirements
RUN set -eux; \
    "${KAIT_PYTHON}" -m venv /opt/kait/venv; \
    /opt/kait/venv/bin/pip install --no-cache-dir --upgrade pip; \
    case "${KAIT_VARIANT}" in slim|full) ;; *) echo "unsupported KAIT_VARIANT: ${KAIT_VARIANT}" >&2; exit 1 ;; esac; \
    effective_capabilities="${KAIT_CAPABILITIES}"; \
    if [ -z "${effective_capabilities}" ]; then \
      case "${KAIT_VARIANT}" in \
        slim) effective_capabilities="data-science" ;; \
        full) effective_capabilities="data-science,training,orchestration,serving" ;; \
      esac; \
    fi; \
    case ",${effective_capabilities}," in *,data-science,*) ;; *) echo "KAIT_CAPABILITIES must include data-science" >&2; exit 1 ;; esac; \
    requirements_files="slim.txt ${KAIT_HARDWARE}.txt"; \
    identity_capabilities=""; \
    old_ifs="${IFS}"; IFS=','; \
    for capability in ${effective_capabilities}; do \
      case "${capability}" in \
        data-science) ;; \
        training) requirements_files="${requirements_files} base.txt training.txt" ;; \
        orchestration) requirements_files="${requirements_files} orchestration.txt" ;; \
        serving) requirements_files="${requirements_files} serving.txt" ;; \
        *) echo "unsupported KAIT_CAPABILITIES entry: ${capability}" >&2; exit 1 ;; \
      esac; \
      case ",${identity_capabilities}," in *,"${capability}",*) echo "duplicate KAIT_CAPABILITIES entry: ${capability}" >&2; exit 1 ;; esac; \
      if [ -n "${identity_capabilities}" ]; then identity_capabilities="${identity_capabilities},"; fi; \
      identity_capabilities="${identity_capabilities}\"${capability}\""; \
    done; \
    IFS="${old_ifs}"; \
    for requirements_file in ${requirements_files}; do \
      test -f "/opt/kait/requirements/${requirements_file}" || { echo "missing requirements manifest: ${requirements_file}" >&2; exit 1; }; \
      /opt/kait/venv/bin/pip install --no-cache-dir -r "/opt/kait/requirements/${requirements_file}"; \
    done; \
    if [ -n "${KAIT_EXTRA_PYTHON_PACKAGES}" ]; then \
      /opt/kait/venv/bin/pip install --no-cache-dir ${KAIT_EXTRA_PYTHON_PACKAGES}; \
    fi; \
    printf '{"schema":1,"hardware":"%s","variant":"%s","capabilities":[%s]}\n' \
      "${KAIT_HARDWARE}" "${KAIT_VARIANT}" "${identity_capabilities}" > /etc/kait/identity.json

COPY --from=kait-build /out/kait /usr/local/bin/kait
RUN chmod +x /usr/local/bin/kait \
    && touch /buildkite/buildkite-agent.cfg

WORKDIR /buildkite/builds
EXPOSE 9090
VOLUME ["/buildkite/builds"]
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/kait"]
