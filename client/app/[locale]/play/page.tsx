import { getTranslations, setRequestLocale } from "next-intl/server";
import { PlaySetupScreen } from "@/features/play/components/play-setup-screen";
import { getPlaySetupMaps } from "@/features/play/data";
import { playModeQuerySchema } from "@/features/play/schemas";
import type { AppLocale } from "@/lib/i18n/routing";

type PlayPageProps = Readonly<{
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ mode?: string }>;
}>;

export default async function PlayPage({ params, searchParams }: PlayPageProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const query = await searchParams;
  const parsedMode = playModeQuerySchema.safeParse(query.mode ?? "solo");
  const mode = parsedMode.success ? parsedMode.data : "solo";

  const [mapsResult, t] = await Promise.all([
    getPlaySetupMaps().catch(() => ({
      maps: [] as Awaited<ReturnType<typeof getPlaySetupMaps>>["maps"],
    })),
    getTranslations("Play"),
  ]);

  return (
    <PlaySetupScreen
      locale={appLocale}
      mode={mode}
      maps={mapsResult.maps}
      homeHref={`/${appLocale}`}
      copy={{
        title: t("setup.title"),
        subtitle: t("setup.subtitle"),
        modes: {
          solo: t("modes.solo"),
          quick_play: t("modes.quick_play"),
          practice: t("modes.practice"),
        },
        descriptions: {
          solo: t("descriptions.solo"),
          quick_play: t("descriptions.quick_play"),
          practice: t("descriptions.practice"),
        },
        mapLabel: t("setup.mapLabel"),
        mapPlaceholder: t("setup.mapPlaceholder"),
        noMaps: t("setup.noMaps"),
        roundsLabel: t("setup.roundsLabel"),
        timerLabel: t("setup.timerLabel"),
        timerOff: t("setup.timerOff"),
        start: t("setup.start"),
        starting: t("setup.starting"),
        quickPlayDefaults: t("setup.quickPlayDefaults"),
        practiceRules: t("setup.practiceRules"),
        back: t("setup.back"),
        errors: {
          validation: t("errors.validation"),
          unavailable: t("errors.unavailable"),
          rateLimited: t("errors.rateLimited"),
          unauthenticated: t("errors.unauthenticated"),
          notEnoughLocations: t("errors.notEnoughLocations"),
          generic: t("errors.generic"),
        },
      }}
    />
  );
}
