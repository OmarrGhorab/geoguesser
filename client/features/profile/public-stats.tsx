import { useTranslations } from "next-intl";
import type { PublicProfileDTO, StatsDTO } from "./types";

type PublicStatsProps = {
  profile: PublicProfileDTO;
  stats: StatsDTO;
};

export function PublicStats({ profile, stats }: PublicStatsProps) {
  const t = useTranslations("Profile");

  return (
    <main className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 py-8">
      <header className="border-b border-slate-200 pb-5">
        <h1 className="text-3xl font-bold text-slate-950">{profile.display_name}</h1>
        {profile.country_code ? <p className="mt-2 text-sm text-slate-600">{profile.country_code}</p> : null}
      </header>

      <section aria-labelledby="public-stats-heading" className="grid gap-4 sm:grid-cols-2 md:grid-cols-4">
        <h2 id="public-stats-heading" className="sr-only">
          {t("stats.title")}
        </h2>
        <StatCard label={t("stats.gamesPlayed")} value={String(stats.games_played)} />
        <StatCard label={t("stats.totalScore")} value={String(stats.total_score)} />
        <StatCard label={t("stats.averageScore")} value={stats.average_score.toFixed(1)} />
        <StatCard label={t("stats.bestScore")} value={String(stats.best_score)} />
      </section>

      {stats.games_played === 0 ? <p className="text-sm text-slate-600">{t("stats.emptyState")}</p> : null}
    </main>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-slate-200 p-4">
      <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className="mt-2 text-lg font-semibold text-slate-950">{value}</dd>
    </div>
  );
}
