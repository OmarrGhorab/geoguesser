import { useTranslations } from "next-intl";
import type { GameHistoryItemDTO, PageDTO } from "./types";

type GameHistoryProps = {
  games: GameHistoryItemDTO[];
  page: PageDTO;
  /** Fully-formed, locale-prefixed path (e.g. "/en/profile") to append the next cursor to. */
  basePath: string;
};

export function GameHistory({ games, page, basePath }: GameHistoryProps) {
  const t = useTranslations("Profile.history");

  return (
    <section aria-labelledby="game-history-heading" className="flex flex-col gap-4">
      <h2 id="game-history-heading" className="text-lg font-semibold text-slate-950">
        {t("title")}
      </h2>

      {games.length === 0 ? (
        <p className="text-sm text-slate-600">{t("empty")}</p>
      ) : (
        <ul className="grid gap-3">
          {games.map((game) => (
            <li key={game.id} className="rounded-lg border border-slate-200 p-4">
              <div className="flex items-center justify-between gap-3">
                <span className="text-sm font-semibold text-slate-950">{t(`status.${game.status}`)}</span>
                <span className="text-sm text-slate-600">{t("score", { score: game.total_score })}</span>
              </div>
              <dl className="mt-2 grid grid-cols-2 gap-2 text-sm text-slate-600 sm:grid-cols-4">
                <div>
                  <dt className="text-xs uppercase text-slate-500">{t("mode")}</dt>
                  <dd>{game.mode}</dd>
                </div>
                <div>
                  <dt className="text-xs uppercase text-slate-500">{t("rounds")}</dt>
                  <dd>
                    {game.current_round_number != null ? t("roundProgress", { current: game.current_round_number, total: game.round_count }) : game.round_count}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs uppercase text-slate-500">{t("createdAt")}</dt>
                  <dd>{new Date(game.created_at).toLocaleDateString()}</dd>
                </div>
              </dl>
            </li>
          ))}
        </ul>
      )}

      {page.next_cursor ? (
        <a
          href={`${basePath}?cursor=${encodeURIComponent(page.next_cursor)}`}
          className="inline-flex w-fit items-center justify-center rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
        >
          {t("loadMore")}
        </a>
      ) : null}
    </section>
  );
}
