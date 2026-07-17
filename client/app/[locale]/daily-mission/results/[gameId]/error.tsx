"use client";

import Link from "next/link";
import type { Route } from "next";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { RotateCcw } from "lucide-react";

export default function DailyResultsError({ reset }: { reset(): void }) {
  const t = useTranslations("DailyMission.game");
  const { locale } = useParams<{ locale: string }>();

  return (
    <main className="grid min-h-dvh place-items-center bg-[radial-gradient(circle_at_center,#25205d,#08051d)] px-5 text-center text-white">
      <section className="w-full max-w-md rounded-3xl border border-white/10 bg-[#100b2e]/85 p-7 shadow-2xl backdrop-blur">
        <h1 className="text-2xl font-black">{t("errors.resultUnavailable")}</h1>
        <p className="mt-2 text-sm leading-6 text-white/60">
          {t("errors.resultUnavailableDescription")}
        </p>
        <div className="mt-6 flex flex-wrap justify-center gap-3">
          <button
            type="button"
            onClick={reset}
            className="inline-flex items-center gap-2 rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-3 text-sm font-black uppercase focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300"
          >
            <RotateCcw className="size-4" />
            {t("errors.retry")}
          </button>
          <Link
            href={`/${locale}/daily-mission` as Route}
            className="rounded-full border border-white/15 bg-white/5 px-6 py-3 text-sm font-black uppercase transition hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300"
          >
            {t("backToMission")}
          </Link>
        </div>
      </section>
    </main>
  );
}
