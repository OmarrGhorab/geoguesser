/**
 * Resolve the browser-facing app origin for redirects.
 *
 * Behind ngrok (or any reverse proxy), Next may see an internal URL like
 * `http://localhost:3000` while the public origin is HTTPS on the tunnel host.
 * Prefer NEXT_PUBLIC_APP_URL, then the request origin. Forwarded headers are
 * intentionally ignored because they are untrusted unless the proxy boundary
 * is configured and enforced outside the application.
 */
export function getPublicAppOrigin(request: Request): string {
  const configured = process.env.NEXT_PUBLIC_APP_URL?.trim();
  if (configured) {
    const url = new URL(configured);
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      throw new Error("NEXT_PUBLIC_APP_URL must use http or https");
    }
    return url.origin;
  }

  return new URL(request.url).origin;
}

export function publicAppUrl(request: Request, path: string): URL {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return new URL(normalized, `${getPublicAppOrigin(request)}/`);
}
