import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { AuthenticatedShell } from "@/features/home/components/authenticated-shell";
import { getAuthenticatedChromeCopy } from "@/features/home/chrome-copy";
import { getAuthenticatedHome } from "@/features/home/data";
import { MissionsBoard } from "@/features/missions/components/missions-board";
import { getMissions } from "@/features/missions/data";
import type { AppLocale } from "@/lib/i18n/routing";

const missionKeys = [
  "daily_completion",
  "score_threshold",
  "round_accuracy",
  "shared_participation",
  "streak_milestone",
] as const;

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "MissionBoard" });
  return { title: t("metaTitle"), description: t("metaDescription") };
}

export default async function MissionsPage({ params }: Props) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);
  const t = await getTranslations("MissionBoard");

  let missions;
  let home;
  try {
    [missions, home] = await Promise.all([
      getMissions(),
      getAuthenticatedHome(),
    ]);
  } catch {
    return (
      <main className="grid min-h-dvh place-items-center bg-[#08071c] px-6 text-center text-white">
        <div className="max-w-md space-y-3">
          <h1 className="text-2xl font-bold">{t("errorTitle")}</h1>
          <p className="text-sm text-white/65">{t("errorBody")}</p>
          <a
            href={`/${appLocale}/missions`}
            className="inline-flex rounded-full bg-[#7654e8] px-6 py-2.5 text-sm font-black uppercase"
          >
            {t("retry")}
          </a>
        </div>
      </main>
    );
  }

  const chrome = await getAuthenticatedChromeCopy(appLocale, home);
  const items = Object.fromEntries(
    missionKeys.map((key) => [
      key,
      {
        title: t(`items.${key}.title`),
        description: t(`items.${key}.description`),
      },
    ]),
  );
  const progressById = Object.fromEntries(
    missions.map((mission) => [
      mission.id,
      t("progress", {
        current: mission.current_value,
        target: mission.target_value,
      }),
    ]),
  );
  const rewardById = Object.fromEntries(
    missions.map((mission) => [
      mission.id,
      t("reward", { xp: mission.reward_xp }),
    ]),
  );

  return (
    <main className="h-dvh max-h-dvh overflow-hidden">
      <AuthenticatedShell
        locale={appLocale}
        copy={chrome}
        activeSidebarItem="missions"
        activeTopNavId="challenges"
        showPremiumPromo={false}
      >
        <MissionsBoard
          locale={appLocale}
          missions={missions}
          progressById={progressById}
          rewardById={rewardById}
          copy={{
            title: t("title"),
            subtitle: t("subtitle"),
            daily: t("daily"),
            weekly: t("weekly"),
            claim: t("claim"),
            claiming: t("claiming"),
            claimed: t("claimed"),
            claimSuccess: t("claimSuccess"),
            claimError: t("claimError"),
            complete: t("complete"),
            empty: t("empty"),
            items,
          }}
        />
      </AuthenticatedShell>
    </main>
  );
}
