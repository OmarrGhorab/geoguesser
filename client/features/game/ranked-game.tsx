"use client";

import { useCallback, useEffect, useState, useTransition } from "react";
import { useTranslations } from "next-intl";
import {
  loadRankedResultsAction,
  reloadRankedRoundAction,
  submitRankedGuessAction,
} from "./ranked-actions";
import type {
  RankedGameDTO,
  RankedGameResultsResponse,
  RankedGuessResultResponse,
  RankedRoundDTO,
} from "./ranked-types";

type RankedGameProps = {
  gameId: string;
  initialGame: RankedGameDTO;
  initialRound: RankedRoundDTO | null;
};

export function RankedGame({ gameId, initialGame, initialRound }: RankedGameProps) {
  const t = useTranslations("RankedGame");
  const [game, setGame] = useState(initialGame);
  const [round, setRound] = useState(initialRound);
  const [results, setResults] = useState<RankedGameResultsResponse | null>(null);
  const [lastGuess, setLastGuess] = useState<RankedGuessResultResponse | null>(null);
  const [latitude, setLatitude] = useState("0");
  const [longitude, setLongitude] = useState("0");
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const [now, setNow] = useState(0);

  const clockReady = now > 0;
  const effectiveStartsInMs =
    round?.starts_at && clockReady ? new Date(round.starts_at).getTime() - now : 0;
  const canGuess =
    clockReady &&
    game.status === "active" &&
    round != null &&
    round.status === "active" &&
    effectiveStartsInMs <= 0 &&
    lastGuess == null;

  useEffect(() => {
    const boot = window.setTimeout(() => setNow(Date.now()), 0);
    const id = window.setInterval(() => setNow(Date.now()), 250);
    return () => {
      window.clearTimeout(boot);
      window.clearInterval(id);
    };
  }, [round?.starts_at]);

  useEffect(() => {
    if (game.status === "completed" && !results) {
      startTransition(async () => {
        const action = await loadRankedResultsAction(gameId);
        if (action.ok && action.kind === "results") {
          setResults(action.results);
          setError(null);
        }
      });
    }
  }, [game.status, gameId, results]);

  const onReload = useCallback(() => {
    startTransition(async () => {
      const action = await reloadRankedRoundAction(gameId);
      if (!action.ok) {
        setError(action.message);
        return;
      }
      setError(null);
      if (action.kind === "results") {
        setResults(action.results);
        setGame(action.results.game);
        setRound(null);
        return;
      }
      if (action.kind === "reload") {
        setGame(action.game);
        setRound(action.round);
        setLastGuess(null);
      }
    });
  }, [gameId]);

  const onSubmit = () => {
    if (!round || !canGuess) return;
    const lat = Number(latitude);
    const lng = Number(longitude);
    startTransition(async () => {
      const action = await submitRankedGuessAction(gameId, round.id, {
        latitude: lat,
        longitude: lng,
      });
      if (!action.ok) {
        setError(action.message);
        return;
      }
      if (action.kind !== "guess") {
        return;
      }
      setLastGuess(action.result);
      setError(null);
      // Refresh for multiplayer round advancement.
      const reload = await reloadRankedRoundAction(gameId);
      if (!reload.ok) {
        return;
      }
      if (reload.kind === "reload") {
        setGame(reload.game);
        setRound(reload.round);
        if (reload.round && reload.round.id !== round.id) {
          setLastGuess(null);
        }
        return;
      }
      if (reload.kind === "results") {
        setResults(reload.results);
        setGame(reload.results.game);
        setRound(null);
      }
    });
  };

  if (results || game.status === "completed") {
    const players = results?.players ?? [];
    return (
      <section className="mx-auto flex w-full max-w-3xl flex-col gap-4" aria-labelledby="ranked-results-heading">
        <h1 id="ranked-results-heading" className="text-2xl font-semibold tracking-tight">
          {t("results.title")}
        </h1>
        <p className="text-sm text-zinc-600 dark:text-zinc-300" role="status" aria-live="polite">
          {t("results.message")}
        </p>
        <ul className="divide-y divide-zinc-200 rounded-lg border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
          {players.map((player) => (
            <li key={player.id} className="flex items-center justify-between px-4 py-3 text-sm">
              <span className="font-medium">{player.display_name}</span>
              <span aria-label={t("results.playerScore", { name: player.display_name, score: player.total_score })}>
                {player.total_score}
              </span>
            </li>
          ))}
        </ul>
        {error ? (
          <p className="text-sm text-red-600" role="alert">
            {error}
          </p>
        ) : null}
        <button
          type="button"
          className="self-start rounded-md border border-zinc-300 px-4 py-2 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:border-zinc-700"
          onClick={onReload}
          disabled={isPending}
          aria-label={t("actions.reload")}
        >
          {t("actions.reload")}
        </button>
      </section>
    );
  }

  const countdownSeconds = Math.max(0, Math.ceil(effectiveStartsInMs / 1000));

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-4" aria-labelledby="ranked-game-heading">
      <header className="flex flex-col gap-1">
        <h1 id="ranked-game-heading" className="text-2xl font-semibold tracking-tight">
          {t("title")}
        </h1>
        <p className="text-sm text-zinc-600 dark:text-zinc-300">
          {t("roundProgress", {
            current: round?.round_number ?? game.current_round_number ?? 1,
            total: game.round_count,
          })}
        </p>
      </header>

      {effectiveStartsInMs > 0 ? (
        <p className="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-100" role="status" aria-live="polite">
          {t("countdown", { seconds: countdownSeconds })}
        </p>
      ) : (
        <p className="text-sm text-zinc-600 dark:text-zinc-300" role="status" aria-live="polite">
          {t("roundReady")}
        </p>
      )}

      {round?.media?.url ? (
        // Media is safe public URL only; no hidden coordinates are rendered pre-guess.
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={round.media.url}
          alt={t("mediaAlt", { round: round.round_number })}
          className="max-h-80 w-full rounded-lg object-cover ring-1 ring-zinc-200 dark:ring-zinc-800"
        />
      ) : (
        <div className="flex h-48 items-center justify-center rounded-lg border border-dashed border-zinc-300 text-sm text-zinc-500 dark:border-zinc-700">
          {t("mediaPending")}
        </div>
      )}

      <form
        className="grid gap-3 rounded-lg border border-zinc-200 p-4 dark:border-zinc-800 sm:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit();
        }}
      >
        <label className="flex flex-col gap-1 text-sm">
          <span>{t("guess.latitude")}</span>
          <input
            type="number"
            name="latitude"
            step="any"
            min={-90}
            max={90}
            value={latitude}
            onChange={(e) => setLatitude(e.target.value)}
            disabled={!canGuess || isPending}
            className="rounded-md border border-zinc-300 px-3 py-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-950"
            aria-label={t("guess.latitude")}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span>{t("guess.longitude")}</span>
          <input
            type="number"
            name="longitude"
            step="any"
            min={-180}
            max={180}
            value={longitude}
            onChange={(e) => setLongitude(e.target.value)}
            disabled={!canGuess || isPending}
            className="rounded-md border border-zinc-300 px-3 py-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-950"
            aria-label={t("guess.longitude")}
          />
        </label>
        <div className="flex flex-wrap gap-2 sm:col-span-2">
          <button
            type="submit"
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
            disabled={!canGuess || isPending}
            aria-label={t("actions.submitGuess")}
            aria-busy={isPending || undefined}
          >
            {isPending ? t("actions.submitting") : t("actions.submitGuess")}
          </button>
          <button
            type="button"
            className="rounded-md border border-zinc-300 px-4 py-2 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:border-zinc-700"
            onClick={onReload}
            disabled={isPending}
            aria-label={t("actions.reload")}
          >
            {t("actions.reload")}
          </button>
        </div>
      </form>

      {lastGuess ? (
        <div className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-100" role="status" aria-live="polite">
          {t("guess.submitted", {
            score: lastGuess.guess.score,
            distance: lastGuess.guess.distance_meters,
          })}
        </div>
      ) : null}

      {error ? (
        <p className="text-sm text-red-600" role="alert">
          {error}
        </p>
      ) : null}
    </section>
  );
}
