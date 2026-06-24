# Tailwind Review

## Review Scope

Review Tailwind CSS v4 utility-first styling, design tokens, CSS variables, responsive utilities, spacing, typography, arbitrary values, inline styles, and CSS file usage.

## Blockers

- CSS Modules unless specifically requested.
- Inline styles for static design values.
- `!important` usage without a strong integration reason.
- Hardcoded colors that bypass design tokens.
- Responsive or RTL-breaking layout choices.

## What To Check

- Tailwind v4 tokens are defined with CSS variables and `@theme` where appropriate.
- Utilities are readable and not excessively duplicated.
- Arbitrary values are rare and justified.
- Spacing and typography follow system scales.
- Focus, disabled, hover, reduced-motion, dark, and responsive states are covered.
- Layout uses stable dimensions to prevent shift.
- RTL-sensitive styles avoid physical left/right where logical alternatives are needed.

## Severity Guidance

- `High`: styling breaks layout, accessibility, or RTL across key viewports.
- `Medium`: token bypass, arbitrary value sprawl, maintainability risk.
- `Low`: class cleanup or consistency.

## Example Finding

```text
Medium - features/dashboard/components/stat-card.tsx:18
The component hardcodes `text-[#4f46e5]` instead of using a design token.
Why it matters: This bypasses theme control, dark mode, and brand consistency.
Recommended fix: map the color to a Tailwind v4 token or existing semantic utility.
```

