import { NextIntlClientProvider } from "next-intl";
import { getMessages, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { routing, type AppLocale } from "@/lib/i18n/routing";

type DailyResultsLayoutProps = Readonly<{
  children: React.ReactNode;
  params: Promise<{ locale: string; gameId: string }>;
}>;

/**
 * Client-only shells (loading/error) call useTranslations("DailyMission.game").
 * Root layout intentionally passes messages={{}} for a minimal client catalog,
 * so this segment re-provides only the keys those boundaries need.
 */
export default async function DailyResultsLayout({
  children,
  params,
}: DailyResultsLayoutProps) {
  const { locale } = await params;

  if (!routing.locales.includes(locale as AppLocale)) {
    notFound();
  }

  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const allMessages = await getMessages();
  const dailyMission = (
    allMessages as {
      DailyMission?: { game?: Record<string, unknown> };
    }
  ).DailyMission;

  return (
    <NextIntlClientProvider
      locale={appLocale}
      messages={{
        DailyMission: {
          game: dailyMission?.game ?? {},
        },
      }}
    >
      {children}
    </NextIntlClientProvider>
  );
}
