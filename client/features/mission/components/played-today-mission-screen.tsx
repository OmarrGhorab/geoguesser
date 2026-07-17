"use client";

import Image from "next/image";
import Link from "next/link";
import {
  ArrowLeft,
  BarChart3,
  Gamepad2,
  Globe2,
  Route as RouteIcon,
  Trophy,
  Users,
} from "lucide-react";
import type { Route } from "next";
import { startDailyMissionAction } from "@/features/mission/actions";
import { MISSION_ASSETS } from "@/features/mission/assets";
import type { DailyGameResults } from "@/features/mission/schemas";
import type {
  DailyMissionCopy,
  DailyResultsCopy,
} from "@/features/mission/types";
import type { AppLocale } from "@/lib/i18n/routing";
import { DailyResultsPanel } from "./daily-results-screen";
import { MissionDayPicker } from "./mission-day-picker";
import { MissionRouteMap } from "./mission-route-map";

type PlayedTodayMissionScreenProps = Readonly<{
  locale: AppLocale;
  copy: DailyMissionCopy;
  resultsCopy: DailyResultsCopy;
  results: DailyGameResults;
  selectedDate: string;
  todayDate: string;
  googleMapsApiKey: string;
  dailyGamesPlayed: number;
  dailyGamesTotal: number;
  isToday: boolean;
}>;

function formatDistance(locale: AppLocale, distanceMeters: number) {
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(distanceMeters / 1_000)} km`;
}

export function PlayedTodayMissionScreen({
  locale,
  copy,
  resultsCopy,
  results,
  selectedDate,
  todayDate,
  googleMapsApiKey,
  dailyGamesPlayed,
  dailyGamesTotal,
  isToday,
}: PlayedTodayMissionScreenProps) {
  const rounds = [...results.rounds].sort(
    (left, right) => left.round_number - right.round_number,
  );
  const guesses = rounds.flatMap((round) => round.guesses.slice(0, 1));
  const bestScore = guesses.reduce(
    (best, guess) => Math.max(best, guess.score),
    0,
  );
  const totalDistance = guesses.reduce(
    (total, guess) => total + guess.distance_meters,
    0,
  );
  const homeHref = `/${locale}` as Route;
  const score = results.game.total_score.toLocaleString(locale);
  const canPlayAnother = isToday && dailyGamesPlayed < dailyGamesTotal;
  const startAction = startDailyMissionAction.bind(null, locale);

  const benefits = [
    [Globe2, copy.played.mapsBenefit, copy.played.mapsBenefitBody],
    [Gamepad2, copy.played.modesBenefit, copy.played.modesBenefitBody],
    [Users, copy.played.friendsBenefit, copy.played.friendsBenefitBody],
    [BarChart3, copy.played.progressBenefit, copy.played.progressBenefitBody],
  ] as const;

  return (
    <main
      data-testid="played-today-mission-screen"
      aria-label={copy.played.completedTitle}
      className="min-h-dvh overflow-x-hidden bg-[#07051d] bg-[radial-gradient(circle_at_50%_0%,rgba(77,33,153,.22),transparent_34%)] text-white"
    >
      <section
        data-testid="played-today-overview-screen"
        className="mx-auto flex min-h-dvh w-full max-w-[76rem] flex-col px-3 pb-6 sm:px-5 lg:px-7"
      >
        <header className="relative flex h-20 shrink-0 items-center justify-center">
          <Link
            href={homeHref}
            aria-label={copy.aria.back}
            className="absolute start-0 grid size-10 place-items-center rounded-full border border-violet-400/25 bg-[#130d36] transition hover:bg-[#241657] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300"
          >
            <ArrowLeft className="size-4 rtl:rotate-180" />
          </Link>
          <MissionDayPicker
            selectedDate={selectedDate}
            todayDate={todayDate}
            copy={{
              today: copy.today,
              prevDay: copy.prevDay,
              nextDay: copy.nextDay,
              aria: copy.aria,
              weekdays: copy.weekdays,
              months: copy.months,
            }}
          />
        </header>

        <section className="relative h-[20rem] shrink-0 overflow-hidden rounded-[1.35rem] border border-violet-400/45 bg-[#0c0a2c] shadow-[inset_0_0_35px_rgba(113,48,255,.18),0_18px_55px_rgba(0,0,0,.35)] sm:h-[23rem] lg:h-[25rem]">
          <MissionRouteMap
            rounds={rounds}
            roundLabel={copy.played.round}
            ariaLabel={copy.played.resultMap}
          />
          <div className="absolute inset-x-0 bottom-0 flex flex-wrap items-end justify-between gap-3 bg-gradient-to-t from-[#090720] via-[#090720]/82 to-transparent px-5 pt-16 pb-5 sm:px-7">
            <div>
              <p className="text-[.62rem] font-black tracking-[.18em] text-violet-200 uppercase">
                {copy.played.completedTitle}
              </p>
              <p className="mt-1 text-xl font-black italic sm:text-2xl">
                {copy.played.mapSummary}
              </p>
            </div>
            <a
              href="#daily-results-detail"
              aria-controls="daily-results-detail"
              className="rounded-full bg-[linear-gradient(180deg,#B04BFF_0%,#8B33E6_45%,#6325B8_100%)] px-7 py-2.5 text-xs font-black tracking-wide uppercase shadow-[0_0_24px_rgba(155,74,255,0.45),inset_0_1px_0_rgba(255,255,255,0.18)] transition hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            >
              {copy.viewResults}
            </a>
          </div>
        </section>

        <section className="mt-5 grid gap-4 lg:grid-cols-[minmax(0,1.65fr)_minmax(19rem,.8fr)]">
          <div className="grid gap-4">
            <article className="relative min-h-[17rem] overflow-hidden rounded-2xl border border-violet-400/30 bg-[#100b2e]">
              <Image
                src={MISSION_ASSETS.afterMission}
                alt=""
                fill
                sizes="(max-width: 1024px) 100vw, 760px"
                className="object-cover object-[61%_50%]"
              />
              <div className="absolute inset-0 bg-gradient-to-r from-[#100b2e] via-[#100b2e]/84 to-transparent rtl:bg-gradient-to-l" />
              <div className="relative flex h-full max-w-[25rem] flex-col justify-center p-6 sm:p-8">
                <h2 className="text-[2rem] leading-[1.02] font-black text-violet-100 italic sm:text-[2.35rem]">
                  {copy.played.premiumTitle}
                </h2>
                <ul className="mt-5 space-y-2 text-xs font-bold text-white/82 sm:text-sm">
                  {copy.played.premiumBenefits.map((benefit) => (
                    <li key={benefit} className="flex items-center gap-2">
                      <span className="text-fuchsia-400" aria-hidden="true">
                        ✓
                      </span>
                      {benefit}
                    </li>
                  ))}
                </ul>
              </div>
            </article>

            <article className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-violet-400/25 bg-[#100b2e] px-5 py-4">
              <div className="flex items-center gap-3">
                <span className="grid size-11 place-items-center rounded-xl bg-violet-500/20 text-violet-200">
                  <RouteIcon className="size-6" aria-hidden="true" />
                </span>
                <div>
                  <p className="font-black italic">
                    {copy.played.completeTitle}
                  </p>
                  <p className="text-xs text-white/55">
                    {copy.played.completeBody}
                  </p>
                </div>
              </div>
              <a
                href="#daily-results-detail"
                className="rounded-full border border-violet-300/30 bg-violet-500/15 px-6 py-2 text-xs font-black uppercase transition hover:bg-violet-500/25 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
              >
                {copy.viewResults}
              </a>
            </article>
          </div>

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-1">
            <article className="rounded-2xl border border-violet-400/30 bg-[#100b2e] p-5">
              <div className="flex items-center gap-4">
                <Trophy className="size-12 shrink-0 fill-[#ffb23f] text-[#ffb23f]" />
                <div>
                  <p className="text-5xl leading-none font-black text-white">
                    {score}
                  </p>
                  <p className="mt-1 text-xs text-white/60">
                    {copy.played.score}
                  </p>
                </div>
              </div>
              <dl className="mt-5 grid grid-cols-3 gap-2">
                <div className="rounded-xl bg-white/5 p-3">
                  <dt className="text-[.58rem] text-white/45 uppercase">
                    {copy.played.bestRound}
                  </dt>
                  <dd className="mt-1 text-sm font-black text-amber-300">
                    {bestScore.toLocaleString(locale)}
                  </dd>
                </div>
                <div className="rounded-xl bg-white/5 p-3">
                  <dt className="text-[.58rem] text-white/45 uppercase">
                    {copy.played.rounds}
                  </dt>
                  <dd className="mt-1 text-sm font-black">
                    {rounds.length.toLocaleString(locale)}
                  </dd>
                </div>
                <div className="rounded-xl bg-white/5 p-3">
                  <dt className="text-[.58rem] text-white/45 uppercase">
                    {copy.played.totalDistance}
                  </dt>
                  <dd className="mt-1 text-sm font-black">
                    {formatDistance(locale, totalDistance)}
                  </dd>
                </div>
              </dl>
            </article>

            <article className="relative min-h-[12rem] overflow-hidden rounded-2xl border border-violet-400/30 bg-[#100b2e]">
              <Image
                src={MISSION_ASSETS.prizeWorld}
                alt=""
                fill
                sizes="340px"
                className="object-cover object-[70%_50%] opacity-85"
              />
              <div className="absolute inset-0 bg-gradient-to-r from-[#100b2e] via-[#100b2e]/88 to-transparent rtl:bg-gradient-to-l" />
              <div className="relative flex h-full max-w-[72%] flex-col justify-center p-5">
                <p className="text-xl font-black italic">
                  {copy.played.completeTitle}
                </p>
                <p className="mt-1 text-[.68rem] leading-4 text-white/60">
                  {copy.gamesProgress}
                </p>
                {canPlayAnother ? (
                  <form action={startAction} className="mt-4">
                    <button
                      type="submit"
                      className="rounded-full bg-[linear-gradient(180deg,#B04BFF,#6325B8)] px-6 py-2.5 text-xs font-black uppercase shadow-[0_0_20px_rgba(155,74,255,.35)] transition hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
                    >
                      {copy.play}
                    </button>
                  </form>
                ) : (
                  <a
                    href="#daily-results-detail"
                    className="mt-4 w-fit rounded-full bg-[linear-gradient(180deg,#B04BFF,#6325B8)] px-6 py-2.5 text-xs font-black uppercase shadow-[0_0_20px_rgba(155,74,255,.35)]"
                  >
                    {copy.viewResults}
                  </a>
                )}
              </div>
            </article>
          </div>
        </section>

        <section className="mt-5 grid grid-cols-2 overflow-hidden rounded-2xl border border-violet-400/25 bg-[#100b2e] sm:grid-cols-4">
          {benefits.map(([Icon, title, body]) => (
            <div
              key={title}
              className="flex items-center gap-3 border-white/8 p-4 not-last:border-e"
            >
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-violet-500/20 text-violet-300">
                <Icon className="size-5" aria-hidden="true" />
              </span>
              <div>
                <p className="text-sm font-black">{title}</p>
                <p className="text-[.65rem] text-white/50">{body}</p>
              </div>
            </div>
          ))}
        </section>
      </section>

      <section
        data-testid="played-today-results-screen"
        className="mx-auto flex min-h-dvh w-full max-w-[76rem] items-center px-3 py-10 sm:px-5 lg:px-7"
      >
        <DailyResultsPanel
          id="daily-results-detail"
          locale={locale}
          results={results}
          googleMapsApiKey={googleMapsApiKey}
          copy={resultsCopy}
        />
      </section>
    </main>
  );
}
