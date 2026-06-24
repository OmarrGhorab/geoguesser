# Docker Review

## Verify

- Dockerfile.
- Multi-stage builds.
- BuildKit.
- Non-root user.
- Image size.
- `.dockerignore`.
- Distroless images.
- Health checks.
- Image scanning.
- Compose.
- Environment variables.
- Secrets.

## Reject

- Runtime image with build tools.
- Running as root.
- Missing `.dockerignore`.
- Secrets baked into image.
- Unpinned or unscanned base images.

## Common Findings

High: final image uses `golang` base and runs as root. Impact: large attack surface and elevated container privileges. Recommendation: build multi-stage and run distroless/non-root final image.

