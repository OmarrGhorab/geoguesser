import type { Metadata } from "next";
import { NextIntlClientProvider } from "next-intl";
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

  const title = t("title");
  const description = t("description");
  const brandIcon = {
    url: "/logo-3.png",
    type: "image/png",
  } as const;

  return {
    metadataBase: siteUrl,
    title: {
      default: title,
      template: `%s · ${title}`,
    },
    description,
    applicationName: title,
    authors: [{ name: title }],
    creator: title,
    publisher: title,
    keywords: [
      "WorldGuess",
      "geography game",
      "guess the location",
      "street view quiz",
      "multiplayer geography",
      "world explorer",
    ],
    icons: {
      icon: brandIcon,
      shortcut: brandIcon,
      apple: brandIcon,
    },
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
      siteName: title,
      title,
      description,
      locale: appLocale === "ar" ? "ar_AR" : "en_US",
      images: [
        {
          url: "/logo-3.png",
          alt: title,
        },
      ],
    },
    twitter: {
      card: "summary",
      title,
      description,
      images: ["/logo-3.png"],
    },
    robots: {
      index: true,
      follow: true,
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

  // Minimal client provider for next-intl navigation; no full message catalog.
  return (
    <html lang={appLocale} dir={getDirection(appLocale)}>
      <body>
        <NextIntlClientProvider locale={appLocale} messages={{}}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
