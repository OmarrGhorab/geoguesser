# GitHub Actions Review

## Verify

- Formatting.
- Linting.
- Testing.
- Coverage.
- Docker build.
- Caching.
- Security scanning.
- Dependency caching.
- Dependabot.
- CodeQL.
- Trivy.
- Release workflow.
- Semantic versioning.
- Cache optimization.
- Least-privilege permissions.

## Reject

- Deploy without tests.
- Secrets exposed to pull requests.
- Missing lint.
- Missing image scanning.
- Broad workflow permissions.

## Common Findings

High: workflow publishes image before tests complete. Impact: broken or vulnerable artifacts can be released. Recommendation: make image publish depend on lint, tests, CodeQL, and Trivy scan.

