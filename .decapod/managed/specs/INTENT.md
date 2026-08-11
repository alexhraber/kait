# Intent

<!-- decapod:declared-capabilities:start -->

## Declared Capability Surfaces

- `authentication`
- `background-processing`
- `event-driven`
- `external-integrations`
- `infrastructure-management`
- `persistent-state`
- `public-api`
- `secrets-handling`

<!-- decapod:declared-capabilities:end -->

## Product Outcome
Kaite ships self-hosted Buildkite agents with a portable AI/ML runtime: a
pinned agent, slim/full Python toolchains, hardware image contracts, and
collector-friendly metrics. Operators pick an image tag for the host class and
run ordinary Buildkite jobs inside that environment.

## What This Project Is
A small Go supervisor (`cmd/kaite`) packaged into hardware-specific Linux
images. Buildkite remains the orchestrator; Kaite owns process supervision,
input validation, health/metrics, and diagnostic subcommands (`doctor`,
`smoke`, `hardware`). Active delivery targets are CPU and Apple Silicon
(linux/arm64). NVIDIA, AMD, and Intel bake targets stay available for
deliberate host testing but are inactive in automatic CI and release until
matching runners exist.

Key operating facts:
- **Primary languages**: Go, Dockerfile, shell, YAML
- **Surfaces**: Go supervisor, Docker Bake matrix, Docker launcher, Kubernetes
  Job template, Python requirement layers, Release Please + GHCR image CI
- **Docs**: root README for operators; `docs/architecture.md` for system layout

## Product View
```mermaid
flowchart LR
  U[AI Platform Operator] --> P[Kaite image and launcher]
  P --> O[User-visible Outcome]
  P --> G[Proof Gates]
  G --> E[Evidence Artifacts]
```

## Inferred Baseline
- Repository: kaite
- Product type: self-hosted AI runtime image
- Primary languages: Go
- Detected surfaces: runtime, image matrix, deployment templates, CI

## Scope
| Area | In Scope | Proof Surface |
|---|---|---|
| Core workflow | Start a Buildkite agent in the selected hardware image | Go tests, container smoke tests, deployment examples |
| Data contracts | Pass Buildkite, hardware, and observability inputs explicitly | INTERFACES.md and manifest review |
| Delivery quality | Build and validate the image matrix without embedding secrets | VALIDATION.md and CI bake graph |

## Non-Goals (Falsifiable)
| Non-goal | How to falsify |
|---|---|
| Feature creep beyond the primary outcome | Any PR adds capability not tied to outcome criteria |
| Shipping without evidence | Missing validation artifacts for promoted changes |
| Ambiguous ownership boundaries | Missing owner/system-of-record in interfaces |

## Constraints
- Technical: runtime, dependency, and topology boundaries are explicit.
- Operational: deployment, rollback, and incident ownership are defined.
- Security/compliance: sensitive data handling and authz are mandatory.

## Acceptance Criteria (must be objectively testable)
- [ ] CPU, Apple Silicon arm64, NVIDIA, AMD, and Intel image targets are versioned and reproducible from the Bake definition.
- [ ] Each hardware target exposes canonical `<tag>-<hardware>-slim` and `<tag>-<hardware>-full` image tags, with slim compatibility aliases preserved for active CPU and Apple images.
- [ ] Every published target declares its supported Linux platform; GPU images remain limited to platforms supported by their vendor runtime.
- [ ] Image CI builds and runs `kaite smoke` for every active target on a matching host class before the workflow can pass.
- [ ] NVIDIA, AMD, and Intel jobs remain inactive on ordinary pushes and tags, with explicit manual opt-in required when matching hosts are available.
- [ ] The default entrypoint validates the Buildkite token, forwards standard agent options, and exits with the child status.
- [ ] Every image contains the pinned Buildkite agent; `kaite smoke` verifies the compact slim contract and imports the advertised full AI/ML package layer.
- [ ] KAITE_O11Y selects none, Prometheus, Datadog, or Splunk/OTel behavior without changing workload commands.
- [ ] Docker and Kubernetes launch paths pass environment inputs and accelerator resources without committing secrets.
- [ ] Go tests, bake-plan validation, CI, documentation, and Decapod validation are present.

## Epistemic Custody Fields

### Active Assumptions
- [x] Buildkite cluster tokens are supplied at runtime through an environment variable or mounted file.
- [x] Vendor collectors/agents own Datadog and Splunk credentials; Kaite emits compatible runtime signals.
- [ ] Exact accelerator base-image availability is verified by image CI and may require operator overrides.
- [x] Ubuntu/glibc is the common image contract; Apple Silicon is Linux arm64 CPU execution, not Apple Metal inside a container.

### Confidence & Risk Level
- **Confidence**: Medium (Rationale: runtime contract and local tests are concrete; accelerator builds depend on upstream base images.)
- **Risk**: Medium (Impact of wrong assumptions: a hardware image or collector integration may require a target-specific override.)

### Measured vs Inferred Facts
| Fact | Source (Provenance) | Type (Measured/Inferred) |
|---|---|---|
| Buildkite start accepts token, tags, config, and Kubernetes exec options | Buildkite agent CLI contract | Measured from official docs |
| Five hardware image targets select base images, platforms, and agent tags | docker-bake.hcl | Inferred from repository source |
| O11y modes keep credentials outside the image | runtime and deployment docs | Inferred from repository source |

### Unresolved Contradictions
- [ ] List any evidence that conflicts with current assumptions or intent.

### Deferred Questions
- [ ] Should future releases publish framework-specific images separately from the base hardware images?
- [ ] Which collector deployment is canonical for each operator's Kubernetes platform?

### Stop Conditions
- [ ] Explicit conditions under which the agent should stop and ask for help.

### Proof Required Before Completion
- [ ] GOCACHE=/tmp/kaite-gocache go test ./... passes.
- [ ] docker buildx bake --print renders all five targets.
- [ ] GitHub Actions routes active CPU and arm64 work to native hosts; accelerator work is routed to explicit host labels only after the inactive manual path is enabled, rather than relying on emulation for device proof.
- [ ] decapod validate passes after generated artifacts and living specs are refreshed.

## Tradeoffs Register
| Decision | Benefit | Cost | Review Trigger |
|---|---|---|---|
| Simplicity vs extensibility | Faster iteration | Potential rework | Feature set expands |
| Strict gates vs dev speed | Higher confidence | More upfront discipline | Lead time regressions |

## First Implementation Slice
- [x] Start Kaite as a Buildkite agent with runtime-selected hardware and o11y.
- [x] Define token, tag, metrics, collector, Docker, and Kubernetes input contracts.
- [x] Postpone framework-specific training images, checkpoint stores, and vendor collector deployment automation.

## Open Questions (with decision deadlines)
| Question | Owner | Deadline | Decision |
|---|---|---|---|
| Which interfaces are versioned at launch? | TBD | YYYY-MM-DD | |
| Which non-functional target is hardest to hit? | TBD | YYYY-MM-DD | |

<!-- decapod:codebase-attestation:start -->

## Codebase Attestation

- Repository signal fingerprint: `6c4805884ae976e92ae4a4aa78a88d1ef2b5d83bc3a16bd9d53bdd45c03b6a87`
- Significant implementation surfaces: `.github/` (4 files), `Dockerfile/` (1 files), `Makefile/` (1 files), `README.md/` (1 files), `deploy/` (4 files), `go.mod/` (1 files), `requirements/` (1 files)
- Refreshed from the current codebase by `decapod specs.refresh`
<!-- decapod:codebase-attestation:end -->
