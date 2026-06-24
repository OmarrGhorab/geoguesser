# File Storage

## Concepts

File storage covers multipart uploads, validation, streaming uploads, local development storage, S3-compatible storage, and signed URLs.

## Architecture Decisions

- Stream files; do not load large uploads fully into memory.
- Validate size, content type, and extension.
- Use local storage only for development.
- Use S3-compatible storage in production.
- Use signed URLs for direct upload/download when appropriate.

## Trade-offs

Proxying uploads through the API centralizes validation but consumes API bandwidth. Signed URLs reduce load but require careful policy and callback validation.

## Anti-patterns

- Trusting client-provided file names.
- Storing uploads in the container filesystem in production.
- No size limits.
- Public buckets by default.
- Serving unvalidated user content inline.

## Common Mistakes

- Path traversal via filenames.
- Missing content sniffing.
- No virus/malware workflow for risky domains.
- No cleanup for failed uploads.
- No object lifecycle rules.

## Production Examples

Use `multipart.Reader` for streaming and store metadata in PostgreSQL after object upload succeeds.

## Go Code Samples

```go
func sanitizeFilename(name string) string {
	base := filepath.Base(name)
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
}
```

## Performance Considerations

Stream uploads, use signed URLs for large files, and avoid buffering entire payloads.

## Security Considerations

Validate file type, enforce size limits, sanitize names, use private buckets, and generate short-lived signed URLs.

## Scalability Considerations

Object storage scales better than local disk. Store metadata separately and design lifecycle/retention policies.

