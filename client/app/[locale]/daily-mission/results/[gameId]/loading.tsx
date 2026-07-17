"use client";

import { useTranslations } from "next-intl";

export default function DailyResultsLoading() {
  const t = useTranslations("DailyMission.game");

  return (
    <main
      aria-busy="true"
      aria-label={t("loadingResults")}
      className="relative grid min-h-dvh place-items-center overflow-hidden bg-[#090720] text-white"
    >
      <div className="absolute inset-0 animate-pulse bg-[radial-gradient(circle_at_center,#26365e_0%,#141536_45%,#090720_100%)] motion-reduce:animate-none" />
      <div className="relative flex flex-col items-center gap-4">
        <span className="size-12 animate-spin rounded-full border-4 border-violet-300/20 border-t-violet-300 motion-reduce:animate-none" />
        <p className="text-sm font-black tracking-wider text-white/75 uppercase">
          {t("loadingResults")}
        </p>
      </div>
    </main>
  );
}
