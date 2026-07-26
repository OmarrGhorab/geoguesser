"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { Route } from "next";
import {
  createPracticeGameAction,
  createQuickPlayAction,
  createSoloGameAction,
} from "@/features/play/actions";
import { gamePlayHref } from "@/features/play/routes";
import type { PlayModeQuery, PlayableMap } from "@/features/play/schemas";
import type { AppLocale } from "@/lib/i18n/routing";

export type PlaySetupCopy = Readonly<{
  title: string;
  subtitle: string;
  modes: {
    solo: string;
    quick_play: string;
    practice: string;
  };
  descriptions: {
    solo: string;
    quick_play: string;
    practice: string;
  };
  mapLabel: string;
  mapPlaceholder: string;
  noMaps: string;
  roundsLabel: string;
  timerLabel: string;
  timerOff: string;
  start: string;
  starting: string;
  quickPlayDefaults: string;
  practiceRules: string;
  back: string;
  errors: {
    validation: string;
    unavailable: string;
    rateLimited: string;
    unauthenticated: string;
    notEnoughLocations: string;
    generic: string;
  };
}>;

const ROUND_OPTIONS = [1, 3, 5, 10] as const;

type PlaySetupScreenProps = Readonly<{
  locale: AppLocale;
  mode: PlayModeQuery;
  maps: PlayableMap[];
  copy: PlaySetupCopy;
  homeHref: Route | string;
}>;

const TIMER_OPTIONS = [
  { value: 30, label: "30s" },
  { value: 60, label: "60s" },
  { value: 180, label: "3m" },
  { value: 300, label: "5m" },
  { value: null, labelKey: "timerOff" as const },
] as const;

function errorMessage(code: string, copy: PlaySetupCopy): string {
  switch (code) {
    case "validation_failed":
      return copy.errors.validation;
    case "rate_limited":
      return copy.errors.rateLimited;
    case "unauthenticated":
      return copy.errors.unauthenticated;
    case "unavailable":
      return copy.errors.unavailable;
    case "not_enough_locations":
      return copy.errors.notEnoughLocations;
    default:
      return copy.errors.generic;
  }
}

export function PlaySetupScreen({
  locale,
  mode,
  maps,
  copy,
  homeHref,
}: PlaySetupScreenProps) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [mapId, setMapId] = useState(maps[0]?.id ?? "");
  const [roundCount, setRoundCount] = useState<number>(5);
  const [timerSeconds, setTimerSeconds] = useState<number | null>(180);
  const [error, setError] = useState<string | null>(null);

  const activeMaps = maps.filter((map) => map.status === "active");
  const selectableMaps = activeMaps.length > 0 ? activeMaps : maps;

  const start = () => {
    setError(null);
    startTransition(async () => {
      const idempotencyKey = `play-${mode}-${crypto.randomUUID()}`;
      let result;
      if (mode === "quick_play") {
        result = await createQuickPlayAction({ idempotencyKey }, locale);
      } else if (mode === "practice") {
        if (!mapId) {
          setError(copy.errors.validation);
          return;
        }
        result = await createPracticeGameAction(
          { mapId, idempotencyKey },
          locale,
        );
      } else {
        if (!mapId) {
          setError(copy.errors.validation);
          return;
        }
        result = await createSoloGameAction(
          {
            mapId,
            roundCount,
            timerSeconds,
            idempotencyKey,
          },
          locale,
        );
      }

      if (!result.ok) {
        setError(errorMessage(result.code, copy));
        return;
      }
      router.push(gamePlayHref(locale, result.gameId));
    });
  };

  return (
    <main className="min-h-dvh bg-[#08051d] px-4 py-8 text-white sm:px-8">
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
        <div className="flex items-center justify-between gap-4">
          <Link
            href={homeHref as Route}
            className="text-sm font-semibold text-violet-200 underline-offset-4 hover:underline"
          >
            {copy.back}
          </Link>
        </div>

        <header className="space-y-2">
          <p className="text-xs font-black tracking-[0.25em] text-violet-300 uppercase">
            {copy.modes[mode]}
          </p>
          <h1 className="text-3xl font-black tracking-tight sm:text-4xl">
            {copy.title}
          </h1>
          <p className="text-sm text-white/70 sm:text-base">
            {copy.descriptions[mode]}
          </p>
        </header>

        {mode === "quick_play" ? (
          <section className="rounded-2xl border border-white/10 bg-white/5 p-5">
            <p className="text-sm leading-relaxed text-white/80">
              {copy.quickPlayDefaults}
            </p>
          </section>
        ) : null}

        {mode === "practice" ? (
          <section className="rounded-2xl border border-white/10 bg-white/5 p-5">
            <p className="text-sm leading-relaxed text-white/80">
              {copy.practiceRules}
            </p>
          </section>
        ) : null}

        {mode !== "quick_play" ? (
          <section className="space-y-3 rounded-2xl border border-white/10 bg-white/5 p-5">
            <label className="block space-y-2">
              <span className="text-xs font-black tracking-wide text-white/70 uppercase">
                {copy.mapLabel}
              </span>
              {selectableMaps.length === 0 ? (
                <p className="text-sm text-amber-200">{copy.noMaps}</p>
              ) : (
                <select
                  value={mapId}
                  onChange={(event) => setMapId(event.target.value)}
                  className="w-full rounded-xl border border-white/15 bg-[#120b2e] px-3 py-3 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-violet-400"
                >
                  <option value="" disabled>
                    {copy.mapPlaceholder}
                  </option>
                  {selectableMaps.map((map) => (
                    <option key={map.id} value={map.id}>
                      {map.name}
                    </option>
                  ))}
                </select>
              )}
            </label>

            {mode === "solo" ? (
              <>
                <fieldset className="space-y-2">
                  <legend className="text-xs font-black tracking-wide text-white/70 uppercase">
                    {copy.roundsLabel}
                  </legend>
                  <div className="flex flex-wrap gap-2">
                    {ROUND_OPTIONS.map((count) => (
                      <button
                        key={count}
                        type="button"
                        onClick={() => setRoundCount(count)}
                        className={`rounded-full px-4 py-2 text-sm font-black ${
                          roundCount === count
                            ? "bg-violet-500 text-white"
                            : "bg-white/10 text-white/80 hover:bg-white/15"
                        }`}
                      >
                        {count}
                      </button>
                    ))}
                  </div>
                </fieldset>

                <fieldset className="space-y-2">
                  <legend className="text-xs font-black tracking-wide text-white/70 uppercase">
                    {copy.timerLabel}
                  </legend>
                  <div className="flex flex-wrap gap-2">
                    {TIMER_OPTIONS.map((option) => {
                      const selected = timerSeconds === option.value;
                      const label =
                        option.value === null
                          ? copy.timerOff
                          : option.label;
                      return (
                        <button
                          key={String(option.value)}
                          type="button"
                          onClick={() => setTimerSeconds(option.value)}
                          className={`rounded-full px-4 py-2 text-sm font-black ${
                            selected
                              ? "bg-violet-500 text-white"
                              : "bg-white/10 text-white/80 hover:bg-white/15"
                          }`}
                        >
                          {label}
                        </button>
                      );
                    })}
                  </div>
                </fieldset>
              </>
            ) : null}
          </section>
        ) : null}

        {error ? (
          <p role="alert" className="text-sm font-semibold text-red-300">
            {error}
          </p>
        ) : null}

        <button
          type="button"
          onClick={start}
          disabled={
            isPending ||
            (mode !== "quick_play" && selectableMaps.length === 0)
          }
          className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-8 py-3.5 text-sm font-black tracking-wide uppercase shadow-lg shadow-violet-950/40 disabled:cursor-not-allowed disabled:opacity-45"
        >
          {isPending ? copy.starting : copy.start}
        </button>
      </div>
    </main>
  );
}
