# Kait on Kubernetes

[`kait-agent.yaml`](kait-agent.yaml) is a one-shot Buildkite agent Job: the
pod registers, claims one matching job, then disconnects so the Job can finish.

## Quick start

```bash
kubectl create secret generic buildkite-agent \
  --from-literal=token="$BUILDKITE_AGENT_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f deploy/kubernetes/kait-agent.yaml
```

Defaults in the template:

- image `ghcr.io/alexhraber/kait:cpu-data-science` (`data-science` profile)
- queue `ai`, `BUILDKITE_AGENT_DISCONNECT_AFTER_JOB=true`
- Prometheus scrape annotations on port `9090`

For a continuously available pool, use a Deployment or
[Buildkite Agent Stack for Kubernetes](https://buildkite.com/docs/agent/kubernetes).
Do not set `BUILDKITE_KUBERNETES_EXEC` on this plain Job; that flag is for the
Agent Stack only.

## Targeting

Change `image`, `KAIT_HARDWARE`, `KAIT_VARIANT`, `KAIT_PROFILE`, capability
tags, and resources for the host class. Active paths are CPU (`linux/amd64` + `arm64`)
and Apple (`linux/arm64`). Accelerator images need the matching device plugin
and stay manual opt-ins outside automatic CI/release. The image's baked
identity must agree with the values and tags in the manifest.

Pipeline side:

```yaml
steps:
  - label: ":robot: CPU evaluation"
    command: "kait smoke"
    agents:
      queue: ai
      kait.hardware: cpu
      kait.capability.data-science: "true"
```

## Observability

| `KAIT_O11Y` | Behavior |
| --- | --- |
| `prometheus` | `/metrics` + logs on stdout/stderr |
| `datadog` | DogStatsD counters (`KAIT_DD_*` / `DD_*`); agent owns credentials |
| `splunk` | Same metrics + standard `OTEL_*` for a collector sidecar/DaemonSet |

Keep vendor API keys out of the image and this manifest. Use Secrets or the
node-level collector.
