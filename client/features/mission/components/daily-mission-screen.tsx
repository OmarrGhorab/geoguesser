import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import { ArrowLeft, ChevronDown, ChevronLeft, ChevronRight } from "lucide-react";
import type { AppLocale } from "@/lib/i18n/routing";
import type { DailyMissionCopy } from "@/features/mission/types";
import { MISSION_ASSETS } from "@/features/mission/assets";

type DailyMissionScreenProps = Readonly<{
  locale: AppLocale;
  copy: DailyMissionCopy;
}>;

const playerCropPositions = ["18% 50%", "50% 50%", "82% 50%"] as const;

/**
 * Daily mission landing screen (mission-play.png).
 * Does **not** render the design’s top-left “A” badge.
 */
export function DailyMissionScreen({ locale, copy }: DailyMissionScreenProps) {
  const homeHref = `/${locale}` as Route;

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
      <div className="absolute inset-0 -z-10">
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

      {/* Top: back (no A badge) + TODAY control */}
      <header className="relative z-10 flex items-start justify-between px-5 pt-5 sm:px-8 sm:pt-6">
        <Link
          href={homeHref}
          aria-label={copy.aria.back}
          className="grid size-11 place-items-center rounded-2xl border border-white/10 bg-[#1A1040]/75 text-white shadow-[0_8px_24px_rgba(0,0,0,0.35)] backdrop-blur-md transition hover:bg-white/10"
        >
          <ArrowLeft className="size-5" strokeWidth={2.25} />
          <span className="sr-only">{copy.back}</span>
        </Link>

        <div
          role="group"
          aria-label={copy.aria.dayPicker}
          className="absolute left-1/2 top-5 flex -translate-x-1/2 items-center gap-1 rounded-full border border-white/15 bg-[#1A1040]/80 px-2 py-1.5 shadow-[0_10px_30px_rgba(40,10,90,0.55)] backdrop-blur-md sm:top-6"
        >
          <button
            type="button"
            aria-label={copy.prevDay}
            className="grid size-9 place-items-center rounded-full text-white/80 transition hover:bg-white/10 hover:text-white"
          >
            <ChevronLeft className="size-5" />
          </button>
          <button
            type="button"
            data-testid="mission-today-control"
            className="flex min-w-[7.5rem] items-center justify-center gap-1 px-3 text-sm font-black tracking-[0.14em] text-white uppercase"
          >
            {copy.today}
            <ChevronDown className="size-4 opacity-80" aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label={copy.nextDay}
            className="grid size-9 place-items-center rounded-full text-white/80 transition hover:bg-white/10 hover:text-white"
          >
            <ChevronRight className="size-5" />
          </button>
        </div>

        {/* Spacer balances the back button so TODAY stays centered */}
        <span className="size-11 shrink-0" aria-hidden="true" />
      </header>

      {/* Center feature cards */}
      <div className="relative z-10 flex flex-1 items-center justify-center px-4">
        <ul
          data-testid="mission-feature-cards"
          className="grid w-full max-w-4xl grid-cols-1 gap-4 sm:grid-cols-3 sm:gap-5"
        >
          {cards.map((card) => (
            <li key={card.key}>
              <article
                data-mission-card={card.key}
                className="flex h-full flex-col items-center rounded-[1.35rem] border border-[#7B5CFF]/55 bg-[#120B35]/72 px-5 py-7 text-center shadow-[0_16px_40px_rgba(20,5,60,0.55),inset_0_1px_0_rgba(255,255,255,0.08)] backdrop-blur-md"
              >
                <div className="relative mb-5 h-[5.5rem] w-full sm:h-[6.25rem]">
                  <Image
                    src={card.src}
                    alt=""
                    fill
                    sizes="200px"
                    className="object-contain drop-shadow-[0_12px_28px_rgba(0,0,0,0.45)]"
                    priority
                  />
                </div>
                <h2 className="text-[0.78rem] font-black tracking-[0.06em] text-white uppercase sm:text-[0.82rem]">
                  {card.label}
                </h2>
              </article>
            </li>
          ))}
        </ul>
      </div>

      {/* Bottom bar: avatars + played-today + PLAY */}
      <footer className="relative z-10 px-4 pb-6 sm:px-8 sm:pb-8">
        <div className="mx-auto flex w-full max-w-5xl flex-wrap items-center justify-between gap-3 rounded-full border border-white/10 bg-[#1A1040]/80 px-4 py-3 shadow-[0_14px_40px_rgba(20,5,60,0.55)] backdrop-blur-md sm:px-5">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex shrink-0 -space-x-2">
              {playerCropPositions.map((pos, i) => (
                <span
                  key={pos}
                  className="size-10 shrink-0 rounded-full bg-cover bg-no-repeat ring-2 ring-[#2A1860] sm:size-11"
                  style={{
                    backgroundImage: `url(${MISSION_ASSETS.players})`,
                    backgroundPosition: pos,
                    backgroundSize: "280%",
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
          </div>

          <Link
            href={"#mission-play" as Route}
            data-testid="mission-play-cta"
            className="inline-flex min-w-[8.5rem] items-center justify-center rounded-full bg-[#6B4EFF] px-10 py-3 text-sm font-black tracking-[0.12em] text-white uppercase shadow-[0_10px_28px_rgba(90,50,220,0.55),inset_0_1px_0_rgba(255,255,255,0.15)] [background-image:linear-gradient(180deg,#8B72FF_0%,#6B4EFF_45%,#5538D4_100%)] transition hover:brightness-110 sm:min-w-[10rem] sm:py-3.5 sm:text-base"
          >
            {copy.play}
          </Link>
        </div>
      </footer>
    </main>
  );
}
