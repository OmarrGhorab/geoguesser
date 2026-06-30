import { useTranslations } from "next-intl";
import type { ChallengeMetadataResponse } from "./types";
import { ChallengeCountdown } from "./countdown";

type DailyPanelProps = {
  data: ChallengeMetadataResponse;
  startAction: () => Promise<void>;
};

export function DailyPanel({ data, startAction }: DailyPanelProps) {
  const t = useTranslations("Challenges");
  const { challenge, attempt_state: attempt } = data;

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-6 px-4 py-8">
      <header className="border-b border-slate-200 pb-5">
        <p className="text-sm font-semibold uppercase tracking-wide text-emerald-700">{t("daily.eyebrow")}</p>
        <h1 className="mt-2 text-3xl font-bold text-slate-950">{t("daily.title")}</h1>
        <p className="mt-2 max-w-2xl text-slate-700">{t("daily.description")}</p>
      </header>

      <section className="grid gap-4 md:grid-cols-3">
        <Info label={t("fields.seed")} value={challenge.seed} />
        <Info label={t("fields.map")} value={challenge.map.id} />
        <Info label={t("fields.status")} value={challenge.status} />
        <Info label={t("fields.rounds")} value={String(challenge.settings.round_count)} />
        <Info label={t("fields.timer")} value={challenge.settings.timer_seconds ? `${challenge.settings.timer_seconds}s` : t("fields.noTimer")} />
        <Info label={t("fields.movement")} value={challenge.settings.movement_rules} />
      </section>

      {data.countdown ? <ChallengeCountdown resetEndsAt={data.countdown.reset_ends_at} label={t("daily.countdown")} /> : null}

      <section className="grid gap-4 md:grid-cols-2">
        <div className="rounded-lg border border-slate-200 p-4">
          <h2 className="text-lg font-semibold text-slate-950">{t("streak.title")}</h2>
          <p className="mt-2 text-sm text-slate-700">
            {t("streak.summary", { current: data.streak.current_count, best: data.streak.best_count })}
          </p>
          <p className="mt-1 text-sm text-slate-600">{t(`streak.protection.${data.streak.protection_state}`)}</p>
        </div>
        <div className="rounded-lg border border-slate-200 p-4">
          <h2 className="text-lg font-semibold text-slate-950">{t("missions.title")}</h2>
          <ul className="mt-2 space-y-2 text-sm text-slate-700">
            {data.missions_summary.map((mission) => (
              <li key={mission.code}>
                {mission.code}: {mission.current_value}/{mission.target_value}
              </li>
            ))}
          </ul>
        </div>
      </section>

      <form action={startAction}>
        <button
          className="inline-flex w-fit items-center justify-center rounded-md bg-slate-950 px-4 py-2 text-sm font-semibold text-white focus:outline-none focus:ring-2 focus:ring-emerald-600 disabled:opacity-50"
          type="submit"
        >
          {attempt ? t("daily.resume") : t("daily.start")}
        </button>
      </form>
    </main>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-200 p-4">
      <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-2 break-all text-sm font-medium text-slate-950">{value}</dd>
    </div>
  );
}
