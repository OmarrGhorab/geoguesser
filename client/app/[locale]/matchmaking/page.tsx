import { getMatchmakingStatus } from "@/lib/api/matchmaking";
import { ApiError } from "@/lib/api/errors";
import { MatchmakingPanel } from "@/features/matchmaking/matchmaking-panel";
import { MatchmakingStatePanel } from "@/features/matchmaking/matchmaking-states";
import { getTranslations } from "next-intl/server";
import type { MatchmakingStatusResponse } from "@/features/matchmaking/types";

type MatchmakingPageProps = {
  params: Promise<{ locale: string }>;
};

export default async function MatchmakingPage({ params }: MatchmakingPageProps) {
  const { locale } = await params;
  const t = await getTranslations("Matchmaking");

  let initialStatus: MatchmakingStatusResponse;
  try {
    initialStatus = await getMatchmakingStatus();
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return (
        <main className="px-4 py-8">
          <MatchmakingStatePanel title={t("unauthorized.title")} message={t("unauthorized.message")} role="alert" />
        </main>
      );
    }
    if (error instanceof ApiError && error.status === 503) {
      initialStatus = { status: "temporarily_unavailable", queue: null, match: null };
    } else {
      return (
        <main className="px-4 py-8">
          <MatchmakingStatePanel title={t("errors.unexpectedTitle")} message={t("errors.unexpected")} role="alert" />
        </main>
      );
    }
  }

  return (
    <main className="px-4 py-8">
      <MatchmakingPanel locale={locale} initialStatus={initialStatus} />
    </main>
  );
}
