# Docker

## Concepts

Use Docker for reproducible builds and deployment. Production images must be multi-stage, small, non-root, health-checkable, and scan-friendly.

## Architecture Decisions

- Use BuildKit.
- Use `.dockerignore`.
- Build static Go binaries where possible.
- Use distroless or scratch-like small runtime images.
- Run as non-root.
- Add health checks outside the binary where platform supports them.

## Trade-offs

Distroless images reduce attack surface but make shell debugging unavailable. Keep a separate debug image or use Kubernetes ephemeral debug containers.

## Anti-patterns

- Shipping source code in production image.
- Running as root.
- Using `latest` tags.
- Installing build tools in runtime image.
- Missing `.dockerignore`.

## Common Mistakes

- CGO mismatch with distroless.
- Not copying CA certificates.
- No image labels.
- No vulnerability scanning.
- Large module cache in final layer.

## Production Examples

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
ENTRYPOINT ["/api"]
```

## Go Code Samples

```go
srv := &http.Server{
	Addr:              cfg.HTTPAddr,
	Handler:           router,
	ReadHeaderTimeout: 5 * time.Second,
}
```

## Performance Considerations

Use layer caching for modules and builds. Keep final images small to speed pulls and rollouts.

## Security Considerations

Use non-root, image scanning, pinned bases, minimal images, and no secrets baked into layers.

## Scalability Considerations

Small, immutable images roll out faster and reduce node disk pressure.

