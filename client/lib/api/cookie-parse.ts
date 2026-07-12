type SameSite = "lax" | "strict" | "none";

export type ParsedCookie = {
  name: string;
  value: string;
  path?: string;
  domain?: string;
  httpOnly?: boolean;
  secure?: boolean;
  sameSite?: SameSite;
  maxAge?: number;
  expires?: Date;
};

function parseSameSite(value: string | undefined): SameSite | undefined {
  if (!value) return undefined;
  const normalized = value.toLowerCase();
  if (
    normalized === "lax" ||
    normalized === "strict" ||
    normalized === "none"
  ) {
    return normalized;
  }
  return undefined;
}

/** Parse a single Set-Cookie header value into cookie options. */
export function parseSetCookieHeader(header: string): ParsedCookie | null {
  const parts = header.split(";").map((part) => part.trim());
  const [nameValue, ...attributes] = parts;
  if (!nameValue) return null;

  const eq = nameValue.indexOf("=");
  if (eq <= 0) return null;

  const name = nameValue.slice(0, eq).trim();
  const value = nameValue.slice(eq + 1).trim();
  if (!name) return null;

  const parsed: ParsedCookie = { name, value };

  for (const attribute of attributes) {
    const [rawKey, ...rawRest] = attribute.split("=");
    const key = rawKey?.trim().toLowerCase();
    const attrValue = rawRest.join("=").trim();
    if (!key) continue;

    switch (key) {
      case "path":
        parsed.path = attrValue || "/";
        break;
      case "domain":
        parsed.domain = attrValue || undefined;
        break;
      case "httponly":
        parsed.httpOnly = true;
        break;
      case "secure":
        parsed.secure = true;
        break;
      case "samesite":
        parsed.sameSite = parseSameSite(attrValue);
        break;
      case "max-age": {
        const maxAge = Number(attrValue);
        if (Number.isFinite(maxAge)) parsed.maxAge = maxAge;
        break;
      }
      case "expires": {
        const expires = new Date(attrValue);
        if (!Number.isNaN(expires.getTime())) parsed.expires = expires;
        break;
      }
      default:
        break;
    }
  }

  return parsed;
}
