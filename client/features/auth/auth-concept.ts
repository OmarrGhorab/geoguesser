/**
 * Auth form-panel concept tokens (right side only).
 * Pure data — safe for unit tests without React/DOM.
 */
export const AUTH_CONCEPT = {
  logoSrc: "/logo.png",
  logoAlt: "WorldGuesser",
  /** Electric-blue journey CTA from the auth-2 reference. */
  primaryButtonClass:
    "w-full rounded-full border border-[#7280ff] bg-[#5865F2] py-3.5 text-base font-semibold text-auth-accent-foreground shadow-[0_10px_30px_rgba(88,101,242,0.28)] transition-[filter,box-shadow,transform] hover:brightness-110 hover:shadow-[0_12px_34px_rgba(88,101,242,0.38)] active:scale-[0.98] disabled:pointer-events-none disabled:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-auth-accent/70 focus-visible:ring-offset-2 focus-visible:ring-offset-auth-bg",
  accentTextClass: "font-black not-italic text-auth-accent",
  linkClass:
    "font-semibold text-auth-link transition-colors hover:text-auth-link/85 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-auth-link/50 rounded-sm",
  mutedLinkClass:
    "text-[12px] font-medium text-auth-link transition-colors hover:text-auth-link/85 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-auth-link/50 rounded-sm",
  fieldClass:
    "w-full rounded-xl border border-auth-field-border bg-auth-surface/55 px-5 py-4 text-sm text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.02)] transition-colors placeholder:text-neutral-400 focus:border-auth-field-border-focus focus:bg-auth-surface/75 focus:ring-1 focus:ring-auth-accent/35 focus:outline-none aria-invalid:border-destructive aria-invalid:ring-1 aria-invalid:ring-destructive/40",
  passwordFieldClass:
    "w-full rounded-xl border border-auth-field-border bg-auth-surface/55 py-4 ps-5 pe-12 text-sm text-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.02)] transition-colors placeholder:text-neutral-400 focus:border-auth-field-border-focus focus:bg-auth-surface/75 focus:ring-1 focus:ring-auth-accent/35 focus:outline-none aria-invalid:border-destructive aria-invalid:ring-1 aria-invalid:ring-destructive/40",
  socialButtonClass:
    "flex items-center justify-center gap-3 rounded-full border py-3.5 text-[13px] font-semibold transition-[filter,transform,box-shadow] active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-auth-accent/60",
} as const;
