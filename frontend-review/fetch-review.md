# fetch Review

## Review Scope

Verify native `fetch()`, server-side fetching, request options, error handling, AbortController, credentials, cache configuration, tags, revalidation, and duplicate request prevention.

## Blockers

- Axios.
- React Query for server-renderable data.
- Fetching initial data in `useEffect` when a Server Component can fetch it.
- Calling internal Route Handlers from Server Components.
- Trusting `response.json()` without validating untrusted external data.

## What To Check

- `fetch()` checks `response.ok`.
- Errors are handled intentionally and do not leak secrets.
- Request credentials and headers are explicit when needed.
- Abort signals are used for client-side user-triggered long requests.
- Stable server data is cached with `use cache`, `cacheLife`, and `cacheTag`.
- Mutations revalidate or update matching tags.
- Parallel fetches are started before awaiting when independent.
- Fetch helpers remain small and do not recreate Axios-like abstractions.

## Severity Guidance

- `High`: banned HTTP client, client fetch causing major SSR loss, missing auth credentials, secret leak.
- `Medium`: missing error handling, duplicate requests, missing response validation.
- `Low`: helper naming or minor option cleanup.

## Example Finding

```text
High - features/search/components/search-results.tsx:22
The component fetches initial search results in useEffect even though searchParams are available to the page.
Why it matters: Users see an empty hydrated shell, the request is duplicated on navigation, and the result cannot use Server Component streaming or caching.
Recommended fix: parse searchParams in the page, fetch in an async Server Component, and keep only interactive filters on the client.
```

