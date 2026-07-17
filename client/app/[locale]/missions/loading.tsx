import { getTranslations } from "next-intl/server";

export default async function MissionsLoading() {
  const t = await getTranslations("MissionBoard");
  return (
    <main
      className="grid min-h-dvh place-items-center bg-[#08071c] text-sm font-bold text-white/70"
      aria-busy="true"
    >
      {t("loading")}
    </main>
  );
}
