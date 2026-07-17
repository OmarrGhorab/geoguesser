# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Build Goose separately so production Compose deployments can run schema
# changes before starting the API, without putting a compiler in the API image.
FROM golang:1.25-alpine AS migrate-build

RUN GOBIN=/out go install github.com/pressly/goose/v3/cmd/goose@v3.27.2

FROM alpine:3.21 AS migrate

RUN apk add --no-cache ca-certificates

COPY --from=migrate-build /out/goose /usr/local/bin/goose
COPY migrations /migrations

ENTRYPOINT ["goose"]

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/api /app/api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
