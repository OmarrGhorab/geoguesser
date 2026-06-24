# shadcn/ui

## Concepts

shadcn/ui is the design system foundation. It provides copyable components built on Tailwind CSS and Radix UI primitives. Treat generated components as local base primitives, then extend through composition, variants, and wrappers.

Why this exists: teams need accessible, consistent UI that remains owned by the application without hiding behavior behind a black-box component package.

## Best Practices

- Install and update with `pnpm dlx shadcn@latest`.
- Keep generated components in `components/ui`.
- Compose feature components around UI primitives.
- Prefer Radix behavior for dialogs, menus, popovers, selects, tabs, and tooltips.
- Use `class-variance-authority`, `clsx`, and `tailwind-merge` patterns already established by shadcn.
- Use lucide icons in icon buttons when available.
- Keep accessibility props intact when wrapping primitives.

## Anti-Patterns

- Editing generated components for one feature-specific case.
- Forking Radix behavior manually.
- Replacing semantic button/link behavior with `div`.
- Creating nested cards and decorative wrappers around every section.
- Adding a second component library without a clear reason.
- Ignoring shadcn updates for React or Tailwind compatibility.

## Common Mistakes

- Removing `Slot` or `asChild` behavior and breaking composition.
- Forgetting accessible labels for icon-only buttons.
- Styling disabled states without setting `disabled` or `aria-disabled`.
- Breaking focus rings for visual polish.
- Creating one-off colors outside Tailwind tokens.
- Modifying generated code instead of adding a feature wrapper.

## Production Examples

```tsx
// features/projects/components/delete-project-dialog.tsx
'use client'

import { Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

export function DeleteProjectDialog() {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button size="icon" variant="destructive" aria-label="Delete project">
          <Trash2 className="size-4" aria-hidden="true" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete project?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction>Delete</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
```

## Folder Organization

```text
components/ui/
  button.tsx
  dialog.tsx
  form.tsx
features/projects/components/
  delete-project-dialog.tsx
  project-card.tsx
```

Do not put product-specific components in `components/ui`.

## TypeScript Examples

```tsx
import { Button } from '@/components/ui/button'

type ToolbarButtonProps = {
  label: string
  icon: React.ComponentType<{ className?: string; 'aria-hidden'?: boolean }>
  onClick: () => void
}

export function ToolbarButton({ label, icon: Icon, onClick }: ToolbarButtonProps) {
  return (
    <Button type="button" size="icon" variant="ghost" aria-label={label} onClick={onClick}>
      <Icon className="size-4" aria-hidden="true" />
    </Button>
  )
}
```

## Performance Considerations

- Import only the components needed.
- Keep heavy interactive primitives in client islands.
- Avoid wrapping entire pages in client-only UI shells.
- Use `size-*` utilities for icons to reduce repeated classes.
- Watch bundle impact from complex primitives and chart components.

## Security Considerations

- Do not render untrusted HTML inside UI components.
- Keep destructive actions protected by server authorization, not only confirmation dialogs.
- Avoid leaking sensitive data in modal content that remains mounted.
- Ensure file inputs and upload controls validate server-side.
- Do not trust hidden form fields from shadcn forms.

## Accessibility Considerations

- Preserve Radix labels, descriptions, focus management, and keyboard behavior.
- Use `aria-label` for icon-only buttons.
- Keep visible focus indicators.
- Ensure dialogs have title and description.
- Prefer semantic shadcn primitives over custom interactive divs.

