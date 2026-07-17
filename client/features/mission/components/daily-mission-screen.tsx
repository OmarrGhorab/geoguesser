import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import { ArrowLeft } from "lucide-react";
import type { AppLocale } from "@/lib/i18n/routing";
import type { DailyMissionCopy } from "@/features/mission/types";
import type { DailyMissionData } from "@/features/mission/schemas";
import { startDailyMissionAction } from "@/features/mission/actions";
import { MISSION_ASSETS } from "@/features/mission/assets";
import { MissionDayPicker } from "./mission-day-picker";
import { MissionFeatureCards } from "./mission-feature-cards";

type DailyMissionScreenProps = Readonly<{
  locale: AppLocale;
  copy: DailyMissionCopy;
  mission: DailyMissionData;
  selectedDate: string;
}>;

/**
 * Daily mission landing screen (mission-play.png).
 * Does **not** render the design’s top-left “A” badge.
 */
export function DailyMissionScreen({
  locale,
  copy,
  mission,
  selectedDate,
}: DailyMissionScreenProps) {
  const homeHref = `/${locale}` as Route;
  const todayDate = mission.challenge.challenge_date ?? selectedDate;
  const isToday = selectedDate === todayDate;
  const attempt = mission.attempt_state;
  const ctaLabel =
    attempt?.status === "completed"
      ? copy.viewResults
      : attempt?.game_id
        ? copy.resume
        : copy.play;
  const startAction = startDailyMissionAction.bind(null, locale);

  const cards = [
    {
      key: "locations" as const,
      src: MISSION_ASSETS.map,
      label: copy.cards.locations,
    },
    {
      key: "compare" as const,
      src: MISSION_ASSETS.trophy,
      label: copy.cards.compare,
    },
    {
      key: "daily" as const,
      src: MISSION_ASSETS.fiveK,
      label: copy.cards.daily,
    },
  ];

  return (
    <main
      data-testid="daily-mission-screen"
      aria-label={copy.aria.screen}
      className="relative flex h-dvh max-h-dvh flex-col overflow-hidden text-white"
    >
      {/* Full-bleed mission background */}
      <div className="absolute inset-0 z-0">
        <Image
          src={MISSION_ASSETS.background}
          alt=""
          fill
          priority
          sizes="100vw"
          className="object-cover object-center"
        />
        <span className="absolute inset-0 bg-[#05041A]/35" aria-hidden="true" />
      </div>

      {/* Top: back (no A badge) + TODAY + calendar (highest stacking) */}
      <header className="relative z-50 flex items-start justify-between px-5 pt-5 sm:px-8 sm:pt-6">
        <Link
          href={homeHref}
          aria-label={copy.aria.back}
          className="grid size-11 place-items-center rounded-2xl border border-white/10 bg-[#1A1040]/75 text-white shadow-[0_8px_24px_rgba(0,0,0,0.35)] backdrop-blur-md transition hover:bg-white/10"
        >
          <ArrowLeft className="size-5" strokeWidth={2.25} />
          <span className="sr-only">{copy.back}</span>
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

        {/* Spacer balances the back button so TODAY stays centered */}
        <span className="size-11 shrink-0" aria-hidden="true" />
      </header>

      {/* Center feature cards (animated) */}
      <div className="relative z-10 flex flex-1 items-center justify-center px-4">
        <MissionFeatureCards cards={cards} />
      </div>

      {/* Bottom bar: larger players strip + PLAY */}
      <footer className="relative z-10 px-4 pb-6 sm:px-8 sm:pb-8">
        <div className="mx-auto flex w-full max-w-5xl flex-wrap items-center justify-between gap-3 rounded-full border border-white/10 bg-[#1A1040]/80 px-4 py-3.5 shadow-[0_14px_40px_rgba(20,5,60,0.55)] backdrop-blur-md sm:gap-4 sm:px-6 sm:py-4">
          <div className="flex min-w-0 items-center gap-2">
            {/*
              Three compact faces from the sheet.
              Use CSS background (full-res PNG) — Next/Image was over-downscaling
              tiny sizes and made the avatars look soft/blurry.
            */}
            <div
              data-testid="mission-players-strip"
              className="flex shrink-0 -space-x-2.5"
            >
              {(["0% 50%", "50% 50%", "100% 50%"] as const).map((pos) => (
                <span
                  key={pos}
                  className="size-9 shrink-0 rounded-full bg-no-repeat ring-2 ring-[#1A1040] sm:size-10"
                  style={{
                    backgroundImage: `url(${MISSION_ASSETS.players})`,
                    // 3 faces across → zoom each circle to one face
                    backgroundSize: "300% 100%",
                    backgroundPosition: pos,
                  }}
                  aria-hidden="true"
                />
              ))}
            </div>
            <p
              data-testid="mission-played-today"
              className="truncate text-sm font-semibold text-white sm:text-base"
            >
              {copy.playedToday}
            </p>
            <span className="shrink-0 rounded-full bg-white/10 px-3 py-1 text-xs font-black text-violet-100">
              {copy.gamesProgress}
            </span>
          </div>

          <div className="flex flex-col items-center gap-1.5">
            <form action={startAction}>
              <button
                type="submit"
                disabled={!isToday}
                data-testid="mission-play-cta"
                className="inline-flex min-w-[8.5rem] items-center justify-center rounded-full bg-[#6B4EFF] [background-image:linear-gradient(180deg,#8B72FF_0%,#6B4EFF_45%,#5538D4_100%)] px-10 py-3 text-sm font-black tracking-[0.12em] text-white uppercase shadow-[0_10px_28px_rgba(90,50,220,0.55),inset_0_1px_0_rgba(255,255,255,0.15)] transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-45 sm:min-w-[10rem] sm:py-3.5 sm:text-base"
              >
                {ctaLabel}
              </button>
            </form>
            {!isToday ? (
              <p className="text-xs font-semibold text-white/65">
                {copy.todayOnly}
              </p>
            ) : null}
          </div>
        </div>
      </footer>
    </main>
  );
}
