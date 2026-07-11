import { RankedGame } from "@/features/game/ranked-game";
import type { RankedGameDTO, RankedRoundDTO } from "@/features/game/ranked-types";
import { getCurrentRound, getGame } from "@/lib/api/games";
import { ApiError } from "@/lib/api/errors";
import { getTranslations } from "next-intl/server";

type RankedGamePageProps = {
  params: Promise<{ locale: string; gameId: string }>;
};

type LoadResult =
  | { ok: true; game: RankedGameDTO; round: RankedRoundDTO | null }
  | { ok: false; kind: "unauthorized" | "not_found" | "error" };

async function loadRankedGame(gameId: string): Promise<LoadResult> {
  try {
    const game = await getGame(gameId);
    let round: RankedRoundDTO | null = null;
    if (game.status === "active" || game.status === "pending") {
      try {
        round = await getCurrentRound(gameId);
      } catch (error) {
        if (!(error instanceof ApiError && (error.status === 404 || error.status === 409))) {
          throw error;
        }
      }
    }
    return { ok: true, game, round };
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return { ok: false, kind: "unauthorized" };
    }
    if (error instanceof ApiError && error.status === 404) {
      return { ok: false, kind: "not_found" };
    }
    return { ok: false, kind: "error" };
  }
}

export default async function RankedGamePage({ params }: RankedGamePageProps) {
  const { gameId } = await params;
  const t = await getTranslations("RankedGame");
  const loaded = await loadRankedGame(gameId);

  if (!loaded.ok) {
    if (loaded.kind === "unauthorized") {
      return (
        <main className="mx-auto w-full max-w-3xl px-4 py-8">
          <h1 className="text-xl font-semibold">{t("unauthorized.title")}</h1>
          <p className="mt-2 text-sm text-zinc-600">{t("unauthorized.message")}</p>
        </main>
      );
    }
    if (loaded.kind === "not_found") {
      return (
        <main className="mx-auto w-full max-w-3xl px-4 py-8">
          <h1 className="text-xl font-semibold">{t("notFound.title")}</h1>
          <p className="mt-2 text-sm text-zinc-600">{t("notFound.message")}</p>
        </main>
      );
    }
    return (
      <main className="mx-auto w-full max-w-3xl px-4 py-8">
        <h1 className="text-xl font-semibold">{t("error.title")}</h1>
        <p className="mt-2 text-sm text-zinc-600">{t("error.message")}</p>
      </main>
    );
  }

  return (
    <main className="px-4 py-8">
      <RankedGame gameId={gameId} initialGame={loaded.game} initialRound={loaded.round} />
    </main>
  );
}
