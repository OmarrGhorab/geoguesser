import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import {
  ArrowLeft,
  ArrowRight,
  CalendarX2,
  Compass,
  MapPinOff,
} from "lucide-react";
import { MISSION_ASSETS } from "@/features/mission/assets";
import type { DailyMissionCopy } from "@/features/mission/types";
import type { AppLocale } from "@/lib/i18n/routing";
import { MissionDayPicker } from "./mission-day-picker";

type PastDailyMissionEmptyScreenProps = Readonly<{
  locale: AppLocale;
  copy: DailyMissionCopy;
  selectedDate: string;
  todayDate: string;
  hasStarted: boolean;
}>;

export function PastDailyMissionEmptyScreen({
  locale,
  copy,
  selectedDate,
  todayDate,
  hasStarted,
}: PastDailyMissionEmptyScreenProps) {
  const homeHref = `/${locale}` as Route;
  const todayHref = `/${locale}/daily-mission` as Route;
  const status = hasStarted
    ? copy.past.statusIncomplete
    : copy.past.statusNoRounds;
  const description = hasStarted ? copy.past.incomplete : copy.past.noRounds;

  return (
    <main
      data-testid="past-daily-mission-empty-screen"
      aria-labelledby="past-daily-mission-title"
      className="relative flex h-dvh max-h-dvh flex-col overflow-hidden text-white"
    >
      <div className="absolute inset-0">
        <Image
          src={MISSION_ASSETS.background}
          alt=""
          fill
          priority
          sizes="100vw"
          className="object-cover object-center"
        />
        <span
          className="absolute inset-0 bg-[radial-gradient(circle_at_50%_45%,rgba(74,32,145,0.12),rgba(5,4,26,0.76)_72%)]"
          aria-hidden="true"
        />
      </div>

      <header className="relative z-50 flex items-start justify-between px-5 pt-5 sm:px-8 sm:pt-6">
        <Link
          href={homeHref}
          aria-label={copy.aria.back}
          className="grid size-11 place-items-center rounded-2xl border border-white/10 bg-[#1A1040]/80 text-white shadow-[0_8px_24px_rgba(0,0,0,0.35)] backdrop-blur-md transition hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300"
        >
          <ArrowLeft className="size-5 rtl:rotate-180" aria-hidden="true" />
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

        <span className="size-11 shrink-0" aria-hidden="true" />
      </header>

      <section className="relative z-10 grid flex-1 place-items-center px-4 pt-5 pb-6 sm:px-8 sm:pb-10">
        <div className="relative w-full max-w-3xl overflow-hidden rounded-[2rem] border border-violet-300/20 bg-[#10092F]/88 px-6 py-7 text-center shadow-[0_28px_90px_rgba(7,2,30,0.72),inset_0_1px_0_rgba(255,255,255,0.08)] backdrop-blur-xl sm:px-12 sm:py-10">
          <span
            className="absolute inset-x-20 top-0 h-px bg-gradient-to-r from-transparent via-violet-300/70 to-transparent"
            aria-hidden="true"
          />
          <span className="inline-flex items-center gap-2 rounded-full border border-violet-300/20 bg-violet-300/10 px-4 py-1.5 text-[0.68rem] font-black tracking-[0.18em] text-violet-100 uppercase">
            <Compass className="size-3.5" aria-hidden="true" />
            {copy.past.eyebrow}
          </span>

          <div
            className="relative mx-auto mt-6 grid size-32 place-items-center rounded-full border border-dashed border-violet-300/35 bg-[#090621]/75 shadow-[0_0_50px_rgba(109,61,224,0.28)] sm:size-36"
            aria-hidden="true"
          >
            <span className="absolute inset-3 rounded-full border border-violet-300/15" />
            <span className="absolute top-2 left-1/2 size-2 -translate-x-1/2 rounded-full bg-violet-300 shadow-[0_0_12px_rgba(196,181,253,0.85)]" />
            <span className="absolute right-4 bottom-7 size-1.5 rounded-full bg-fuchsia-300/80" />
            <MapPinOff className="size-12 text-violet-200 sm:size-14" />
          </div>

          <h1
            id="past-daily-mission-title"
            className="mx-auto mt-6 max-w-xl text-2xl leading-tight font-black tracking-[-0.02em] text-white uppercase sm:text-4xl"
          >
            {copy.past.title}
          </h1>
          <p className="mx-auto mt-3 max-w-lg text-sm leading-6 font-semibold text-violet-100/70 sm:text-base">
            {description}
          </p>

          <div className="mx-auto mt-6 flex max-w-md items-center justify-center gap-3 rounded-2xl border border-white/8 bg-white/[0.045] px-4 py-3 text-sm font-bold text-white/78">
            <CalendarX2
              className="size-5 shrink-0 text-violet-300"
              aria-hidden="true"
            />
            <span>{status}</span>
          </div>

          <p className="mt-6 text-xs font-semibold tracking-wide text-white/50">
            {copy.past.todayHint}
          </p>
          <Link
            href={todayHref}
            className="mt-3 inline-flex min-h-12 items-center justify-center gap-2 rounded-full bg-[linear-gradient(180deg,#9B7BFF_0%,#6B4EFF_48%,#5433D1_100%)] px-8 py-3 text-sm font-black tracking-[0.1em] text-white uppercase shadow-[0_12px_32px_rgba(91,52,220,0.48),inset_0_1px_0_rgba(255,255,255,0.24)] transition hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-violet-300"
          >
            {copy.past.todayCta}
            <ArrowRight className="size-4 rtl:rotate-180" aria-hidden="true" />
          </Link>
        </div>
      </section>
    </main>
  );
}
