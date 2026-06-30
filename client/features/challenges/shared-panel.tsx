import { useTranslations } from "next-intl";
import type { ChallengeMetadataResponse } from "./types";

type SharedPanelProps = {
  data: ChallengeMetadataResponse;
  startAction: () => Promise<void>;
};

export function SharedPanel({ data, startAction }: SharedPanelProps) {
  const t = useTranslations("Challenges");
  const { challenge, attempt_state: attempt } = data;

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-4xl flex-col gap-6 px-4 py-8">
      <header className="border-b border-slate-200 pb-5">
        <p className="text-sm font-semibold uppercase tracking-wide text-cyan-700">{t("shared.eyebrow")}</p>
        <h1 className="mt-2 text-3xl font-bold text-slate-950">{t("shared.title")}</h1>
        <p className="mt-2 max-w-2xl text-slate-700">{t("shared.description")}</p>
      </header>

      <section className="grid gap-4 md:grid-cols-2">
        <Info label={t("fields.seed")} value={challenge.seed} />
        <Info label={t("fields.shareCode")} value={challenge.share_code ?? challenge.id} />
        <Info label={t("fields.rounds")} value={String(challenge.settings.round_count)} />
        <Info label={t("fields.status")} value={challenge.status} />
      </section>

      <form action={startAction}>
        <button className="rounded-md bg-slate-950 px-4 py-2 text-sm font-semibold text-white focus:outline-none focus:ring-2 focus:ring-cyan-600" type="submit">
          {attempt ? t("shared.resume") : t("shared.start")}
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
