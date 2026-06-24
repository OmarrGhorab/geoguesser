# GitHub Actions

## Concepts

Use GitHub Actions for linting, testing, building, Docker image publishing, migrations, deployments, secrets, Go module caching, Dependabot, CodeQL, Trivy scanning, artifact caching, release workflow, and semantic versioning.

## Architecture Decisions

- Separate CI, image, security, and release workflows.
- Cache Go modules and build cache.
- Run golangci-lint.
- Run tests with race detector where practical.
- Scan images with Trivy.
- Use CodeQL for security analysis.
- Use Dependabot for dependencies.

## Trade-offs

More gates slow CI but prevent expensive production failures. Use parallel jobs and caching to keep feedback fast.

## Anti-patterns

- Deploying on untested commits.
- Secrets in workflow files.
- Skipping lint because tests pass.
- Publishing unscanned images.
- Running migrations automatically without environment controls.

## Common Mistakes

- Cache keys too broad or stale.
- No permissions hardening.
- Pull request workflows with secret exposure.
- No artifact retention policy.
- No release tags.

## Production Examples

```yaml
name: ci
on: [pull_request, push]
jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
          cache: true
      - run: go test -race ./...
      - uses: golangci/golangci-lint-action@v6
```

## Go Code Samples

```go
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
```

## Performance Considerations

Use Go module cache, build cache, Docker layer cache, and parallel jobs. Keep integration tests focused.

## Security Considerations

Use least-privilege workflow permissions, OIDC for cloud auth, CodeQL, Trivy, Dependabot, and protected environments.

## Scalability Considerations

Split workflows by responsibility so teams can diagnose failures quickly as the codebase grows.

