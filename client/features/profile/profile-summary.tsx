import { useTranslations } from "next-intl";
import type { ProfileDTO, StatsDTO } from "./types";
import { ProfileForm } from "./profile-form";

type ProfileSummaryProps = {
  profile: ProfileDTO;
  stats: StatsDTO;
};

export function ProfileSummary({ profile, stats }: ProfileSummaryProps) {
  const t = useTranslations("Profile");

  return (
    <main className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-4 py-8">
      <header className="border-b border-slate-200 pb-5">
        <h1 className="text-3xl font-bold text-slate-950">{t("title")}</h1>
        <p className="mt-2 text-sm text-slate-600">{profile.email}</p>
      </header>

      <section aria-labelledby="profile-stats-heading" className="grid gap-4 sm:grid-cols-2 md:grid-cols-4">
        <h2 id="profile-stats-heading" className="sr-only">
          {t("stats.title")}
        </h2>
        <StatCard label={t("stats.gamesPlayed")} value={String(stats.games_played)} />
        <StatCard label={t("stats.totalScore")} value={String(stats.total_score)} />
        <StatCard label={t("stats.averageScore")} value={stats.average_score.toFixed(1)} />
        <StatCard label={t("stats.bestScore")} value={String(stats.best_score)} />
      </section>

      <section aria-labelledby="profile-edit-heading">
        <h2 id="profile-edit-heading" className="text-lg font-semibold text-slate-950">
          {t("editHeading")}
        </h2>
        <div className="mt-4">
          <ProfileForm profile={profile} />
        </div>
      </section>
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
