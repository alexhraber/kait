# Security

## Threat Model
The primary trust boundary is the executor that supplies a Buildkite cluster
token, vendor collector settings, and host device access to a privileged AI
workload. Kait must not expand that trust boundary by baking credentials into
an image or forwarding them to logs.

| Threat | Surface | Mitigation | Verification |
|---|---|---|---|
| Token disclosure | Environment, args, logs | Prefer token file; never log token | Go config tests and log review |
| Image tampering | Base images and downloads | Pin agent version and verify SHA256 | Dockerfile review and scanner |
| Host privilege | Docker socket and device mounts | Document mounts; use least privilege | Deployment review |
| Collector credential leak | Datadog/Splunk paths | Credentials stay in node agent/collector | Manifest review |
| Resource exhaustion | AI job and agent process | Queue/resource limits and health probes | Reference job validation |

## Authentication
- Identity source: Buildkite cluster agent token supplied only at runtime.
- Token lifetime and rotation are controlled by Buildkite and the operator's
  secret manager.
- A mounted token file is preferred over a shell-visible token value.

## Authorization
Buildkite cluster queue membership authorizes job dispatch. Kait does not
introduce an application role system; host, container, and device privileges
are controlled by Docker or Kubernetes configuration.

## Data Classification
| Data class | Examples | Storage rules | Access rules |
|---|---|---|---|
| Public | Image metadata and documentation | Registry and repository | Unrestricted |
| Internal | Agent tags and runtime metrics | Buildkite and collectors | Operator team |
| Sensitive | Tokens, API keys, private datasets | Secret manager or mounted volume | Least privilege |

## Sensitive Data Handling
- Encryption at rest is delegated to host, Kubernetes secret, Buildkite, and
  vendor systems.
- Buildkite Agent API and OTLP endpoints must use TLS.
- Kait never logs token values; Buildkite redaction remains authoritative for
  job logs.
- Retention is delegated to Buildkite and the selected collector.

## Supply Chain
- Agent archives are downloaded at a pinned version and checked against the
  published SHA256SUMS file.
- Base images and Python packages are versioned in the Bake file or requirements
  file and should be scanned on tagged builds.
- Registry signing and provenance attestations are required before production
  promotion.
- GHCR package visibility is an operator decision independent of repository
  visibility; public anonymous pulls require an explicit Public package setting.
- Root `SECURITY.md` documents vulnerability reporting and supported versions.

## Secrets Matrix
| Secret | Source | Rotation | Consumer |
|---|---|---|---|
| Buildkite agent token | Docker/Kubernetes secret | Buildkite policy | Kait |
| Datadog API key | Datadog Agent Secret | Vendor policy | Datadog Agent |
| Splunk access token | OTel Collector Secret | Vendor policy | Splunk Collector |

## Security Proof
- No credential is present in the Dockerfile, image tags, examples, or logs.
- Runtime configuration fails closed when an agent token is absent.
- Vendor integrations are collector-friendly and do not require Kait to own
  external API credentials.

<!-- decapod:capability-overlay:public-api:start -->

## Public API Security Overlay

### Authentication Requirements
- All public endpoints MUST validate authentication tokens
- Token validation MUST include expiry, revocation, and scope checks
- Anonymous access MUST be explicitly documented and justified

### Input Validation
- All request bodies MUST be validated against schemas
- Reject requests with unknown fields (strict schema validation)
- Size limits MUST be enforced on all request bodies

### Rate Limiting
- Limits and enforcement boundaries MUST be selected for this deployment
- Clustered enforcement behavior MUST be documented when applicable
- Client-visible throttling behavior MUST be part of the contract when applicable
<!-- decapod:capability-overlay:public-api:end -->

<!-- decapod:codebase-attestation:start -->

## Codebase Attestation

- Repository signal fingerprint: `09785926cd7f035e50e28ff508548568636bef945ba1806b74e7a1d1c19aa712`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (7 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
