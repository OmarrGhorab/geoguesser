# Email

## Concepts

Email covers SMTP, templates, background sending, retries, unsubscribe flows, and deliverability. Do not block user-facing requests on slow email providers.

## Architecture Decisions

- Send email from background jobs.
- Use typed templates.
- Keep provider behind an interface.
- Retry transient failures with backoff.
- Log message IDs and delivery outcomes.

## Trade-offs

SMTP is portable but provider APIs may offer better observability and templates. Keep the interface provider-neutral.

## Anti-patterns

- Sending email synchronously in handlers.
- Building templates with string concatenation.
- Logging full email bodies with PII.
- No retry or dead-letter path.
- No unsubscribe handling for marketing email.

## Common Mistakes

- Missing text alternative.
- No template preview tests.
- No idempotency for repeated sends.
- Not separating transactional and marketing mail.
- No bounce handling.

## Production Examples

User signup commits transaction, enqueues `send_welcome_email`, and the worker renders template and sends via SMTP/provider.

## Go Code Samples

```go
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}
```

## Performance Considerations

Use worker pools and provider timeouts. Avoid rendering expensive templates repeatedly when batching.

## Security Considerations

Do not put secrets in templates. Validate recipient addresses and protect reset links with short TTLs.

## Scalability Considerations

Queue email, track status, and design provider failover for critical transactional mail.

