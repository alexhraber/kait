# Kaite on Kubernetes

kaite-agent.yaml is a self-hosted Buildkite agent Job. Create the
buildkite-agent Secret with the cluster agent token, choose the hardware
image, then set KAITE_O11Y to none, prometheus, datadog, or splunk. CPU and
Apple arm64 are the active image paths. NVIDIA, AMD, and Intel remain explicit
operator opt-ins and their automatic CI/release runner labels are inactive.
Use the `*-slim` image for the compact runtime or the matching `*-full` image
when the broader AI/ML package set is needed; for example, `cpu-slim` and
`apple-full`.

Create the secret without putting the token in a manifest, then submit the
Job:

~~~bash
kubectl create secret generic buildkite-agent \
  --from-literal=token="$BUILDKITE_AGENT_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/kubernetes/kaite-agent.yaml
~~~

The template uses `BUILDKITE_AGENT_QUEUE=ai` and
`BUILDKITE_AGENT_DISCONNECT_AFTER_JOB=true`, so one pod claims one matching
job and exits. Use a Deployment or Buildkite Agent Stack when a continuously
available pool is required. Do not set `BUILDKITE_KUBERNETES_EXEC` on this
plain Job; that option belongs to Agent Stack for Kubernetes.

The image is the Buildkite agent and the workload environment together. Jobs
are dispatched by Buildkite into the selected pod, so pipeline steps can target
the immutable hardware contract with agent tags:

~~~yaml
steps:
  - label: ":robot: CPU evaluation"
    command: "kaite smoke"
    agents:
      queue: ai
      kaite.hardware: cpu
~~~

For target hardware and package footprint, change the image,
`KAITE_HARDWARE`, `KAITE_VARIANT`, agent tags, and resource limits in the Job.
The `cpu-slim` and `cpu-full` images support linux/amd64 and linux/arm64; use
the matching `apple-slim` or `apple-full` image for an Apple Silicon host's
Ubuntu arm64 CPU workload. NVIDIA nodes require the NVIDIA device plugin; AMD nodes require
the ROCm device plugin and /dev/kfd and /dev/dri; Intel nodes require the
oneAPI runtime and /dev/dri. Those accelerator images are inactive in the
repository's automatic CI and release paths until matching hosts are enabled.

Observability is collector-friendly:

- prometheus exposes /metrics and keeps logs on stdout/stderr for the
  cluster's log collector.
- datadog emits runtime counters to DogStatsD using
  KAITE_DD_AGENT_HOST and KAITE_DD_DOGSTATSD_PORT; the Datadog Agent owns
  API credentials and container-log collection.
- splunk exposes the same metrics endpoint and honors standard
  OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_SERVICE_NAME values for a Splunk
  OpenTelemetry Collector sidecar or DaemonSet; the collector owns the access
  token and log export.

Do not put vendor API keys in the image or commit them to this manifest. Use
Kubernetes Secrets or the vendor's node-level collector.
