/** Resolve a field-error localization key to a message, with fallback. */
export function resolveErrorMessage(
  key: string | undefined,
  t: (key: string) => string,
  fallbackKey = "generic",
): string {
  if (!key) return t(fallbackKey);
  try {
    return t(key);
  } catch {
    return t(fallbackKey);
  }
}

/** Pick the first field error key from a field errors map. */
export function firstFieldError(
  fieldErrors: Partial<Record<string, string[]>> | undefined,
  field: string,
): string | undefined {
  return fieldErrors?.[field]?.[0];
}
