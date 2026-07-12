import { NextIntlClientProvider } from "next-intl";
import { getMessages, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { getDirection } from "@/lib/i18n/direction";
import { routing, type AppLocale } from "@/lib/i18n/routing";

type LocaleLayoutProps = Readonly<{
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}>;

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export default async function LocaleLayout({ children, params }: LocaleLayoutProps) {
  const { locale } = await params;

  if (!routing.locales.includes(locale as AppLocale)) {
    notFound();
  }

  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  return (
    <NextIntlClientProvider messages={await getMessages()}>
      <div lang={appLocale} dir={getDirection(appLocale)}>
        {children}
      </div>
    </NextIntlClientProvider>
  );
}
