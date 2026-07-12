import type { MetadataRoute } from "next";
import { routing } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";

export default function sitemap(): MetadataRoute.Sitemap {
  const languages = Object.fromEntries(
    routing.locales.map((locale) => [
      locale,
      new URL(`/${locale}`, siteUrl).toString(),
    ]),
  );

  return routing.locales.map((locale) => ({
    url: new URL(`/${locale}`, siteUrl).toString(),
    alternates: { languages },
  }));
}
