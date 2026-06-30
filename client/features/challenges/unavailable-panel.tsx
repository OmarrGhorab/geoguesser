import { useTranslations } from "next-intl";
import { Link } from "@/lib/i18n/navigation";

export function ChallengeUnavailablePanel() {
  const t = useTranslations("Challenges.daily");

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-950 px-4 text-white">
      <section className="w-full max-w-xl rounded-lg border border-white/10 bg-white/[0.06] p-6 shadow-2xl">
        <p className="text-sm font-semibold uppercase tracking-wide text-emerald-300">{t("eyebrow")}</p>
        <h1 className="mt-3 text-2xl font-bold">{t("unavailableTitle")}</h1>
        <p className="mt-3 text-sm leading-6 text-zinc-300">{t("unavailableMessage")}</p>
        <Link
          href="/"
          className="mt-5 inline-flex rounded-md bg-white px-4 py-2 text-sm font-semibold text-zinc-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-300"
        >
          {t("backToPlay")}
        </Link>
      </section>
    </main>
  );
}
