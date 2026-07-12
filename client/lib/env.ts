import "server-only";

function required(name: string, value: string | undefined): string {
  if (!value?.trim()) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value.trim().replace(/\/$/, "");
}

/** Server-only backend API base, e.g. http://localhost:8080/api/v1 */
export function getBackendApiUrl(): string {
  return required(
    "BACKEND_API_URL",
    process.env.BACKEND_API_URL ?? "http://localhost:8080/api/v1",
  );
}
