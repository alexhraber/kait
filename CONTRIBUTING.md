# Contributing

Thanks for looking at Kaite. Small, focused changes land fastest.

## Setup

```bash
# Go 1.23+
make test
make vet
make build
```

Docker Buildx is required for image work:

```bash
make build-plan
make build-cpu    # local cpu-slim bake
```

## Layout

| Path | Role |
| --- | --- |
| `cmd/kaite/` | Go supervisor (config, process lifecycle, metrics, doctor/smoke) |
| `docs/architecture.md` | System layout, image matrix, deploy and failure model |
| `Dockerfile` + `docker-bake.hcl` | Image matrix |
| `requirements/` | Python layer manifests |
| `deploy/` | Docker launcher and Kubernetes Job template |
| `examples/` | Buildkite pipeline snippets |
| `.github/workflows/` | Image CI, release promotion |

Keep the supervisor dependency-free (stdlib only) unless there is a strong reason
to pull something in.

## Pull requests

- Prefer one concern per PR.
- Match existing style; run `make fmt` and `make test` before opening.
- If you change runtime behavior, env contracts, or image tags, update
  `README.md` (and `requirements/README.md` or deploy docs when relevant).
- Bump the `version` constant in `cmd/kaite/version.go` only when shipping a
  release that should surface a new supervisor identity; Release Please owns
  the package changelog and git tags.

## Releases

1. Merge ordinary PRs to `main` → Release Please maintains a release PR.
2. Merge the release PR → creates the semver tag and GitHub release.
3. Release Please dispatches `release-images.yml` against that tag (GitHub does
   not re-trigger `on: push` tags created with `GITHUB_TOKEN`).
4. Image CI builds active CPU/Apple slim+full tags to GHCR with provenance/SBOM.

Repository settings:

- Enable “Allow GitHub Actions to create and approve pull requests”.
- The release workflow must keep `contents`, `issues`, `pull-requests`, and
  **`actions: write`** so it can `workflow_dispatch` the image job. Without
  `actions: write`, the tag/release still appears but GHCR stays empty.

To republish an existing tag (e.g. after a failed dispatch):

```bash
gh workflow run release-images.yml --ref v0.2.0 \
  -f version=v0.2.0 -f publish_stable=true -f enable_accelerators=false
```

Accelerator images (NVIDIA/AMD/Intel) stay manual opt-in until matching runners
exist. Do not re-enable them on the default CI path without host capacity.

## Code of conduct expectations

Be kind, be specific in review comments, and assume good intent. Security-sensitive
findings should go to the maintainer privately when possible.
