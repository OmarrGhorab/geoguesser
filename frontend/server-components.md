# Server Components

## Concepts

Server Components are the default in App Router. They render on the server, can fetch data directly, can access secrets safely, can stream through Suspense, and do not add JavaScript to the client bundle. Client Components are opt-in islands for interactivity.

Why this exists: Server Components reduce client JavaScript, improve FCP/LCP, simplify data fetching, and keep privileged logic off the browser.

## Best Practices

- Keep pages, layouts, and data-heavy UI as Server Components.
- Move interactivity into small Client Components.
- Pass serializable props from server to client.
- Pass Server Components as `children` to Client Component shells when useful.
- Mark server-only modules with `import 'server-only'`.
- Use async Server Components for data reads.
- Fetch close to where data is used; rely on request memoization and cache utilities.

## Anti-Patterns

- Adding `"use client"` because a child needs interactivity.
- Passing class instances, functions, database records with methods, or non-serializable objects to Client Components.
- Importing `fs`, database clients, secrets, or server-only modules into client graphs.
- Moving data fetching to `useEffect`.
- Creating provider wrappers around the entire document.

## Common Mistakes

- Forgetting that a Client Component's imports join the client bundle.
- Assuming React context works in Server Components.
- Using browser APIs in Server Components.
- Wrapping every component in Suspense even when no async or runtime work exists.
- Not considering DTO shape before passing data to client islands.

## Production Examples

```tsx
// Server Component
import { getInvoices } from '@/features/invoices/data'
import { InvoiceFilters } from '@/features/invoices/components/invoice-filters'

export default async function InvoicesPage() {
  const invoices = await getInvoices()

  return (
    <main>
      <h1>Invoices</h1>
      <InvoiceFilters />
      <ul>
        {invoices.map((invoice) => (
          <li key={invoice.id}>{invoice.customerName}</li>
        ))}
      </ul>
    </main>
  )
}
```

```tsx
// Client Component island
'use client'

import { useState } from 'react'

export function InvoiceFilters() {
  const [open, setOpen] = useState(false)
  return (
    <section>
      <button type="button" onClick={() => setOpen((value) => !value)}>
        Filters
      </button>
      {open ? <form>{/* interactive controls */}</form> : null}
    </section>
  )
}
```

## Folder Organization

```text
features/invoices/
  data.ts              # server-only reads
  components/
    invoice-table.tsx  # Server Component
    invoice-filters.tsx # Client Component
```

Name client files by behavior, not by suffix alone. The `"use client"` directive is the real boundary.

## TypeScript Examples

```ts
// features/invoices/types.ts
export type InvoiceListItem = {
  id: string
  customerName: string
  amountLabel: string
  status: 'draft' | 'sent' | 'paid'
}
```

```tsx
type Props = {
  invoice: InvoiceListItem
}

export function InvoiceRow({ invoice }: Props) {
  return (
    <tr>
      <td>{invoice.customerName}</td>
      <td>{invoice.amountLabel}</td>
      <td>{invoice.status}</td>
    </tr>
  )
}
```

## Performance Considerations

- Small client islands keep hydration and bundle size low.
- Server Components can stream, so isolate slow reads behind Suspense.
- Avoid importing heavy client libraries above server/client boundaries.
- Use cached server reads for stable data shared across routes.
- Prefer server formatting for dates, numbers, and labels where locale is known.

## Security Considerations

- Keep credentials and privileged calls in Server Components or server-only modules.
- Return DTOs instead of raw models.
- Do authorization before reading sensitive records.
- Do not rely on hiding Client Component controls to protect data.
- Avoid serializing sensitive fields into props.

## Accessibility Considerations

- Server-rendered HTML should be meaningful before hydration.
- Client islands must preserve labels, focus, and keyboard behavior.
- Streaming fallback content should be understandable to assistive tech.
- Interactive controls need semantic elements, not clickable `div`s.
- Server-render localized text should include correct language and direction.

