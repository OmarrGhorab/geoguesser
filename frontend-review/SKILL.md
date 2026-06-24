---
name: frontend-review
description: Strict production frontend code review skill for Next.js 16+ App Router applications using TypeScript, Server Components, Server Actions, Route Handlers, native fetch, Zustand, next-intl, Tailwind CSS v4, shadcn/ui, Radix UI, React Hook Form, Zod, Motion, and pnpm. Use only for reviewing frontend code, pull requests, diffs, architecture, performance, accessibility, security, SEO, testing, localization, state management, folder structure, and maintainability. Do not use for generating new features. Enforce standards and reject Axios, React Query, Redux Toolkit, React Router, Pages Router, JavaScript, and CSS Modules unless specifically requested.
---

# Frontend Review

## Mission

Review frontend code like a Staff Engineer approving a production pull request. Be strict. Assume the code will be deployed, maintained by multiple engineers, localized, audited, and scaled.

This skill reviews code only. Do not generate new features unless the user explicitly asks for fixes after the review.

## Stack Contract

Assume:

- Next.js 16+
- App Router
- TypeScript
- Server Components
- Server Actions
- Route Handlers
- Native `fetch()`
- Zustand
- next-intl
- Tailwind CSS v4
- shadcn/ui
- Radix UI
- React Hook Form
- Zod
- Motion
- pnpm

Reject or flag:

- Axios
- React Query
- Redux Toolkit
- React Router
- Pages Router
- JavaScript
- CSS Modules unless specifically requested

## Review Behavior

- Review architecture, not just syntax.
- Never simply say "looks good."
- Prefer targeted fixes over rewrites.
- Explain trade-offs.
- Reference official Next.js 16+, Vercel, TypeScript, Tailwind CSS v4, shadcn/ui, next-intl, React Hook Form, and Zod best practices when relevant.
- Treat Server Actions and Route Handlers as public trust boundaries.
- Treat every `"use client"` directive as a bundle, hydration, and boundary decision.
- Treat every user-facing string as a localization requirement.
- Treat every form, dialog, menu, and state transition as an accessibility surface.

## Required Issue Format

Every issue must include:

- Severity: `Critical`, `High`, `Medium`, `Low`, or `Suggestion`.
- Location: file and line when available.
- Explanation: what is wrong.
- Why it matters: production impact.
- Recommended fix: targeted remediation.
- Example solution: include when it clarifies the fix.

## Review Process

1. Read `review-process.md`.
2. Inspect the diff and surrounding code.
3. Load only the category files relevant to the changed code.
4. Identify blockers before suggestions.
5. Score the review with `scoring.md`.
6. Emit the report using `report-template.md`.
7. Include an action plan in priority order.

## Category Files

- `architecture-review.md`: feature organization, coupling, cohesion, server/client boundaries.
- `nextjs-review.md`: App Router, Server Components, Server Actions, streaming, caching, metadata.
- `typescript-review.md`: strict typing, `any`, assertions, generics, discriminated unions.
- `fetch-review.md`: native fetch, request options, errors, credentials, cache tags.
- `zustand-review.md`: UI-only global state, no server state, no tokens.
- `forms-review.md`: React Hook Form, Zod, accessible states, submissions.
- `validation-review.md`: shared schemas, inference, server validation, duplication.
- `localization-review.md`: next-intl, no hardcoded strings, RTL, metadata.
- `shadcn-review.md`: composition, variants, CVA, Radix accessibility.
- `tailwind-review.md`: Tailwind v4 tokens, CSS variables, utilities, responsive styling.
- `accessibility-review.md`: WCAG AA, semantics, keyboard, focus, screen readers.
- `seo-review.md`: Metadata API, canonical URLs, OG, structured data, robots, sitemap.
- `performance-review.md`: client JS, streaming, rerenders, images, fonts.
- `security-review.md`: XSS, CSRF, cookies, env, Server Actions, sanitization.
- `testing-review.md`: Vitest, Testing Library, Playwright, coverage, assertions.
- `folder-review.md`: feature-based structure, naming, boundaries.
- `code-quality-review.md`: readability, complexity, duplication, dead code.
- `anti-patterns.md`: automatic detection list and severity guidance.
- `scoring.md`: 0-100 category scoring model.
- `report-template.md`: required report format.
- `checklists.md`: quick gates for PR review.

## Default Severity Rules

- `Critical`: security breach, data leak, auth bypass, production outage, broken core flow, inaccessible critical flow.
- `High`: architecture violation, wrong server/client boundary, severe performance regression, missing authz on mutation, broken localization strategy.
- `Medium`: maintainability problem, duplicated validation, weak typing, missing loading/error states, avoidable client JS.
- `Low`: localized polish, naming, small duplication, minor layout or DX issue.
- `Suggestion`: optional improvement with clear upside and low urgency.

## Report Requirement

Always return this structure:

```markdown
# Overall Assessment

One paragraph summarizing code quality.

---

## Score

Overall: XX/100

Architecture:
Next.js:
Performance:
Accessibility:
Security:
Maintainability:
SEO:
Testing:
Localization:
Developer Experience:

---

## Critical Issues

...

---

## High Priority Issues

...

---

## Medium Priority Issues

...

---

## Low Priority Issues

...

---

## Suggestions

...

---

## Positive Findings

...

---

## Action Plan

...
```

If there are no findings in a severity section, write `None found.` Do not omit sections.

