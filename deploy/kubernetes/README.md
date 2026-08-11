# Kaite on Kubernetes

[`kaite-agent.yaml`](kaite-agent.yaml) is a one-shot Buildkite agent Job: the
pod registers, claims one matching job, then disconnects so the Job can finish.

## Quick start

```bash
kubectl create secret generic buildkite-agent \
  --from-literal=token="$BUILDKITE_AGENT_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f deploy/kubernetes/kaite-agent.yaml
```

Defaults in the template:

- image `ghcr.io/alexhraber/kaite:cpu-slim`
- queue `ai`, `BUILDKITE_AGENT_DISCONNECT_AFTER_JOB=true`
- Prometheus scrape annotations on port `9090`

For a continuously available pool, use a Deployment or
[Buildkite Agent Stack for Kubernetes](https://buildkite.com/docs/agent/kubernetes).
Do not set `BUILDKITE_KUBERNETES_EXEC` on this plain Job; that flag is for the
Agent Stack only.

## Targeting

Change `image`, `KAITE_HARDWARE`, `KAITE_VARIANT`, agent tags, and resources
for the host class. Active paths are CPU (`linux/amd64` + `arm64`) and Apple
(`linux/arm64`). Accelerator images need the matching device plugin and stay
manual opt-ins outside automatic CI/release.

Pipeline side:

```yaml
steps:
  - label: ":robot: CPU evaluation"
    command: "kaite smoke"
    agents:
      queue: ai
      kaite.hardware: cpu
      kaite.variant: slim
```

## Observability

| `KAITE_O11Y` | Behavior |
| --- | --- |
| `prometheus` | `/metrics` + logs on stdout/stderr |
| `datadog` | DogStatsD counters (`KAITE_DD_*` / `DD_*`); agent owns credentials |
| `splunk` | Same metrics + standard `OTEL_*` for a collector sidecar/DaemonSet |

Keep vendor API keys out of the image and this manifest. Use Secrets or the
node-level collector.
