"use client";

import Image from "next/image";
import Link from "next/link";
import { useMemo, useState } from "react";
import { ArrowLeft, ChevronDown, Medal, RotateCcw, Trophy } from "lucide-react";
import type { Route } from "next";
import { MISSION_ASSETS } from "@/features/mission/assets";
import type { DailyGameResults } from "@/features/mission/schemas";
import type { DailyResultsCopy } from "@/features/mission/types";
import type { AppLocale } from "@/lib/i18n/routing";
import { DailyResultsMap } from "./daily-results-map";

type DailyResultsProps = Readonly<{
  locale: AppLocale;
  results: DailyGameResults;
  googleMapsApiKey: string;
  copy: DailyResultsCopy;
}>;

type DailyResultsPanelProps = DailyResultsProps &
  Readonly<{
    id?: string;
    showBackToMission?: boolean;
    initialTab?: "results" | "map";
  }>;

function formatDistance(locale: AppLocale, distanceMeters: number) {
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(distanceMeters / 1_000)} km`;
}

export function DailyResultsPanel({
  locale,
  results,
  googleMapsApiKey,
  copy,
  id,
  showBackToMission = false,
  initialTab = "results",
}: DailyResultsPanelProps) {
  const [activeTab, setActiveTab] = useState<"results" | "map">(initialTab);
  const rounds = useMemo(
    () => [...results.rounds].sort((a, b) => a.round_number - b.round_number),
    [results.rounds],
  );
  const guesses = rounds.flatMap((round) => round.guesses.slice(0, 1));
  const totalScore = results.game.total_score;
  const totalDistance = guesses.reduce(
    (sum, guess) => sum + guess.distance_meters,
    0,
  );
  const scorePercent = Math.min(100, (totalScore / 25_000) * 100);
  const missionHref = `/${locale}/daily-mission` as Route;
  const player =
    results.players.find((entry) => entry.role === "player") ??
    results.players[0];

  return (
    <section
      id={id}
      tabIndex={-1}
      aria-labelledby={id ? `${id}-title` : undefined}
      className="w-full scroll-mt-4 rounded-[1.35rem] border border-violet-400/35 bg-[#0b0827] p-3 shadow-[inset_0_0_45px_rgba(99,37,184,.11),0_24px_70px_rgba(0,0,0,.4)] outline-none sm:p-4"
    >
      <div className="flex items-center justify-between gap-4 px-1 pb-3">
        <h2
          id={id ? `${id}-title` : undefined}
          className="text-base font-black uppercase italic"
        >
          {copy.title}
        </h2>
        <div className="flex items-center gap-2 text-end">
          <span className="grid size-8 place-items-center rounded-full border-2 border-violet-400 bg-[#231750] text-xs font-black">
            {(player?.display_name ?? copy.you).slice(0, 1).toUpperCase()}
          </span>
          <div className="hidden sm:block">
            <p className="max-w-44 truncate text-xs font-black">
              {player?.display_name ?? copy.you}
            </p>
            <p className="text-[.62rem] text-violet-300">
              {totalScore.toLocaleString(locale)} pts
            </p>
          </div>
        </div>
      </div>

      <div
        role="tablist"
        aria-label={copy.overview}
        className="mx-auto grid max-w-md grid-cols-2 rounded-lg bg-[#17103b] p-1"
      >
        {(["results", "map"] as const).map((tab) => {
          const selected = activeTab === tab;
          const label = tab === "results" ? copy.resultsTab : copy.mapTab;
          return (
            <button
              key={tab}
              type="button"
              role="tab"
              aria-selected={selected}
              onClick={() => setActiveTab(tab)}
              className={`rounded-md px-5 py-2 text-xs font-black uppercase italic transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-200 ${
                selected
                  ? "bg-[linear-gradient(180deg,#B04BFF_0%,#8B33E6_45%,#6325B8_100%)] text-white shadow-[0_0_22px_rgba(155,74,255,.45)]"
                  : "text-white/55 hover:text-white"
              }`}
            >
              {label}
            </button>
          );
        })}
      </div>

      {activeTab === "results" ? (
        <div role="tabpanel" className="mt-3 space-y-4">
          <section className="grid min-h-[12.5rem] items-center gap-4 rounded-2xl border border-violet-400/20 bg-[#100b2e] p-4 sm:grid-cols-[9rem_minmax(0,1fr)_14rem] sm:p-5">
            <Image
              src={MISSION_ASSETS.summaryTrophy}
              alt=""
              width={160}
              height={160}
              className="mx-auto size-28 object-contain sm:size-36"
            />
            <div className="min-w-0 text-center sm:text-start">
              <p className="text-[.65rem] font-black tracking-wider text-violet-300 uppercase">
                {copy.finalScore}
              </p>
              <p
                data-testid="daily-mission-total-score"
                className="mt-1 bg-[linear-gradient(110deg,#ffb640,#ff685f_48%,#df53ff)] bg-clip-text text-6xl leading-none font-black tracking-[-.06em] text-transparent"
              >
                {totalScore.toLocaleString(locale)}
              </p>
              <p className="mt-1 text-xs font-black italic">{copy.pointsOf}</p>
              <div className="mt-4 h-3 overflow-hidden rounded-full border border-violet-400/30 bg-[#21154d]">
                <div
                  className="h-full rounded-full bg-[linear-gradient(90deg,#ffb43c,#ba3cff_62%,#6e35ff)]"
                  style={{ width: `${scorePercent}%` }}
                />
              </div>
              <div className="mt-1 flex justify-between text-[.58rem] font-bold text-white/45">
                <span>0 pts</span>
                <span>25,000 pts</span>
              </div>
            </div>
            <div className="flex min-h-28 items-center gap-3 rounded-xl border border-violet-400/20 bg-[#17103b] p-4">
              <Medal
                className="size-12 shrink-0 text-[#c97842]"
                aria-hidden="true"
              />
              <p className="text-xs leading-5 text-white/75">
                {copy.statusTitle}
                <br />
                <strong className="text-[#ff9a4c]">{copy.statusBody}</strong>
              </p>
            </div>
          </section>

          <div className="grid grid-cols-5 overflow-hidden rounded-lg bg-[#100b2e] p-1 text-center text-[.66rem] font-black uppercase italic">
            {[
              copy.myGame,
              copy.friends,
              copy.clubs,
              copy.country,
              copy.all,
            ].map((label, index) => (
              <span
                key={label}
                className={`rounded-md px-1 py-2 ${index === 0 ? "bg-[linear-gradient(180deg,#B04BFF,#7027cf)] text-white shadow-[0_0_16px_rgba(155,74,255,.35)]" : "text-white/55"}`}
              >
                {label}
              </span>
            ))}
          </div>

          <section className="overflow-hidden rounded-xl border border-violet-400/25 bg-[#100b2e]">
            <div className="grid grid-cols-[minmax(0,1.5fr)_.7fr_.7fr_2rem] items-center gap-2 border-b border-violet-400/15 px-4 py-2 text-[.56rem] font-black tracking-wide text-violet-200 uppercase">
              <span>{copy.player}</span>
              <span>{copy.roundsCompleted}</span>
              <span>{copy.score}</span>
              <span />
            </div>
            <div className="grid grid-cols-[minmax(0,1.5fr)_.7fr_.7fr_2rem] items-center gap-2 px-4 py-3">
              <div className="flex min-w-0 items-center gap-2">
                <span className="grid size-8 shrink-0 place-items-center rounded-full border-2 border-violet-400 bg-[#251956] text-xs font-black">
                  {(player?.display_name ?? copy.you).slice(0, 1).toUpperCase()}
                </span>
                <span className="truncate text-xs font-black">{copy.you}</span>
              </div>
              <span className="text-xs font-bold">{rounds.length}</span>
              <span className="text-sm font-black">
                {totalScore.toLocaleString(locale)}
              </span>
              <ChevronDown className="size-4 text-violet-300" />
            </div>
            <ol className="grid grid-cols-2 gap-px border-t border-violet-400/15 bg-violet-400/10 p-px sm:grid-cols-6">
              {rounds.map((round) => {
                const guess = round.guesses[0];
                return (
                  <li key={round.round_id} className="bg-[#0d092b] p-3">
                    <p className="flex items-center gap-1 text-[.62rem] font-black italic">
                      <span>
                        {copy.round} {round.round_number}
                      </span>
                      {(guess?.score ?? 0) >= 4000 ? (
                        <Trophy className="size-3 fill-[#ffc341] text-[#ffc341]" />
                      ) : null}
                    </p>
                    <p className="mt-2 text-sm font-black">
                      {(guess?.score ?? 0).toLocaleString(locale)} pts
                    </p>
                    <p className="mt-1 text-[.6rem] text-white/50">
                      {formatDistance(locale, guess?.distance_meters ?? 0)}
                    </p>
                  </li>
                );
              })}
              <li className="bg-[#0d092b] p-3">
                <p className="text-[.62rem] font-black italic">{copy.total}</p>
                <p className="mt-2 text-sm font-black">
                  {totalScore.toLocaleString(locale)} pts
                </p>
                <p className="mt-1 text-[.6rem] text-white/50">
                  {formatDistance(locale, totalDistance)}
                </p>
              </li>
            </ol>
          </section>

          <section className="rounded-xl border border-violet-400/20 bg-[#100b2e] p-5 text-center">
            <p className="text-sm font-black italic">{copy.curious}</p>
            <p className="mt-1 text-xs text-white/45">
              {copy.comparisonUnavailable}
            </p>
            <span className="mt-3 inline-flex items-center gap-2 rounded-lg border border-violet-400/20 px-4 py-2 text-[.65rem] font-bold text-white/45">
              <RotateCcw className="size-3" />
              {copy.replay}
            </span>
          </section>
        </div>
      ) : (
        <section
          role="tabpanel"
          className="relative mt-3 min-h-[38rem] overflow-hidden rounded-2xl border border-violet-400/25 bg-[#100b2e] sm:min-h-[42rem]"
        >
          <DailyResultsMap
            rounds={rounds}
            locale={locale}
            totalScore={totalScore}
            googleMapsApiKey={googleMapsApiKey}
            roundLabel={copy.round}
            scoreLabel={copy.score}
            distanceLabel={copy.distance}
            overviewLabel={copy.overview}
            breakdownLabel={copy.breakdown}
            nextLabel={copy.next}
            finalScoreLabel={copy.finalScore}
            pointsOfLabel={copy.pointsOf}
            yourGuessLabel={copy.yourGuess}
            correctPositionLabel={copy.correctPosition}
            unavailableLabel={copy.mapUnavailable}
            ariaLabel={copy.aria.resultMap}
            showZoomControls
            className="absolute inset-0"
          />
        </section>
      )}

      {showBackToMission ? (
        <div className="mt-5 flex justify-center">
          <Link
            href={missionHref}
            className="rounded-full bg-[linear-gradient(180deg,#B04BFF,#6325B8)] px-9 py-3 text-sm font-black uppercase shadow-[0_0_24px_rgba(155,74,255,.45)]"
          >
            {copy.backToMission}
          </Link>
        </div>
      ) : null}
    </section>
  );
}

export function DailyResultsScreen(props: DailyResultsProps) {
  const missionHref = `/${props.locale}/daily-mission` as Route;
  return (
    <main className="min-h-dvh bg-[#07051d] px-3 py-5 text-white sm:px-5">
      <div className="mx-auto max-w-[60rem]">
        <Link
          href={missionHref}
          aria-label={props.copy.backToMission}
          className="mb-4 grid size-9 place-items-center rounded-full border border-violet-400/25 bg-[#130d36]"
        >
          <ArrowLeft className="size-4 rtl:rotate-180" />
        </Link>
        <DailyResultsPanel {...props} initialTab="map" showBackToMission />
      </div>
    </main>
  );
}
