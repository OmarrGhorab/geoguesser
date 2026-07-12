import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { getDirection } from "@/lib/i18n/direction";
import { routing, type AppLocale } from "@/lib/i18n/routing";
import { siteUrl } from "@/lib/site";
import "../globals.css";

type LocaleLayoutProps = Readonly<{
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}>;

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export async function generateMetadata({
  params,
}: Pick<LocaleLayoutProps, "params">): Promise<Metadata> {
  const { locale } = await params;

  if (!routing.locales.includes(locale as AppLocale)) {
    notFound();
  }

  const appLocale = locale as AppLocale;
  const t = await getTranslations({ locale: appLocale, namespace: "Metadata" });
  const canonicalPath = `/${appLocale}`;

  return {
    metadataBase: siteUrl,
    title: t("title"),
    description: t("description"),
    alternates: {
      canonical: canonicalPath,
      languages: Object.fromEntries(
        routing.locales.map((supportedLocale) => [
          supportedLocale,
          `/${supportedLocale}`,
        ]),
      ),
    },
    openGraph: {
      type: "website",
      url: canonicalPath,
      siteName: t("title"),
      title: t("title"),
      description: t("description"),
      locale: appLocale === "ar" ? "ar_AR" : "en_US",
    },
  };
}

export default async function LocaleLayout({
  children,
  params,
}: LocaleLayoutProps) {
  const { locale } = await params;

  if (!routing.locales.includes(locale as AppLocale)) {
    notFound();
  }

  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  return (
    <html lang={appLocale} dir={getDirection(appLocale)}>
      <body>{children}</body>
    </html>
  );
}
