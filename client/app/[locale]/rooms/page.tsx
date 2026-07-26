import { getTranslations, setRequestLocale } from "next-intl/server";
import { RoomEntryScreen } from "@/features/rooms/components/room-entry-screen";
import { getPlaySetupMaps } from "@/features/play/data";
import type { AppLocale } from "@/lib/i18n/routing";

type RoomsPageProps = Readonly<{
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ mode?: string }>;
}>;

export default async function RoomsPage({ params }: RoomsPageProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const [mapsResult, t] = await Promise.all([
    getPlaySetupMaps().catch(() => ({ maps: [] as Awaited<ReturnType<typeof getPlaySetupMaps>>["maps"] })),
    getTranslations("Rooms"),
  ]);

  return (
    <RoomEntryScreen
      locale={appLocale}
      maps={mapsResult.maps}
      homeHref={`/${appLocale}`}
      copy={{
        title: t("title"),
        subtitle: t("subtitle"),
        createTitle: t("createTitle"),
        joinTitle: t("joinTitle"),
        mapLabel: t("mapLabel"),
        codeLabel: t("codeLabel"),
        codePlaceholder: t("codePlaceholder"),
        create: t("create"),
        creating: t("creating"),
        join: t("join"),
        joining: t("joining"),
        back: t("back"),
        noMaps: t("noMaps"),
        errors: {
          validation: t("errors.validation"),
          unavailable: t("errors.unavailable"),
          notFound: t("errors.notFound"),
          hostActionRequired: t("errors.hostActionRequired"),
          generic: t("errors.generic"),
        },
      }}
    />
  );
}
