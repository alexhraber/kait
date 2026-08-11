# Security Policy

## Supported versions

| Version | Supported |
| --- | --- |
| `v0.2.x` (latest release) | Yes |
| Older tags | Best effort |

Use immutable versioned image tags in production (`vX.Y.Z-cpu-slim`, not only
floating aliases).

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security-sensitive reports.

Email the maintainer listed in the GitHub profile for
[alexhraber/kait](https://github.com/alexhraber/kait), or use GitHub’s
**Private vulnerability reporting** on this repository if enabled.

Include:

- Affected image tag or commit SHA
- Environment (Docker / Kubernetes / bare metal)
- Reproduction steps and impact assessment

You should receive an acknowledgement within a few business days.

## Operational notes

- Never bake `BUILDKITE_AGENT_TOKEN` or vendor API keys into images or commits.
- Prefer `BUILDKITE_AGENT_TOKEN_FILE` with a mounted secret.
- Keep GHCR package visibility intentional: public packages for open pulls, or
  private packages with explicit pull credentials for private fleets.
- Release images are built with provenance and SBOM attestations when published
  by `release-images.yml`.
