const fallbackSiteUrl = "http://localhost:3000";

export const siteUrl = new URL(
  process.env.NEXT_PUBLIC_APP_URL ?? fallbackSiteUrl,
);
