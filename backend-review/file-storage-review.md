# File Storage Review

## Verify

- Upload validation.
- Size limits.
- MIME validation.
- Path traversal protection.
- Signed URLs.
- Streaming.
- Local vs S3-compatible storage separation.
- Private object ACLs.

## Reject

- Trusting client filenames.
- Loading large files fully into memory.
- No content-type validation.
- Public uploads by default.
- Writing uploads to container disk in production.

## Common Findings

Critical: upload path uses raw filename in `filepath.Join`. Impact: path traversal can overwrite arbitrary files. Recommendation: sanitize basename, generate server-side object keys, and store outside executable path.

