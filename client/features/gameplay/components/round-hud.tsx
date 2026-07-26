"use client";

import Link from "next/link";
import Image from "next/image";
import type { Route } from "next";
import {
  COMPASS_TICK_WIDTH_PX,
  compassTickLabel,
  compassTrackOffset,
} from "@/features/mission/compass";

export type RoundHudCopy = Readonly<{
  backToMission: string;
  mapLabel: string;
  total: string;
}>;

export type RoundStatItem = Readonly<{
  roundNumber: number;
  active: boolean;
  value: string;
}>;

type RoundHudProps = Readonly<{
  locale: string;
  backHref: Route | string;
  copy: RoundHudCopy;
  heading: number;
  secondsRemaining: number;
  roundStats: readonly RoundStatItem[];
  totalScore: number;
  /** Seconds remaining at or below this value use the red timer border. */
  timerWarningSeconds?: number;
  logoSrc?: string;
  logoAlt?: string;
}>;

const COMPASS_TICKS = Array.from({ length: 65 }, (_, index) => index);

function formatRoundTimer(seconds: number) {
  const safeSeconds = Math.max(0, Math.floor(seconds));
  return `${Math.floor(safeSeconds / 60)}:${String(safeSeconds % 60).padStart(2, "0")}`;
}

function RoundStat({
  label,
  value,
  active = false,
}: {
  label: string;
  value: string;
  active?: boolean;
}) {
  return (
    <div
      className={`min-w-12 border-r border-white/10 px-2 py-1.5 text-[10px] font-black italic last:border-0 sm:min-w-15 ${active ? "bg-white/10 text-[#ffcc28]" : "text-[#c6a9ff]"}`}
    >
      <span>{label}</span>
      <strong className="block text-sm text-white">{value || "—"}</strong>
    </div>
  );
}

export function RoundHud({
  locale,
  backHref,
  copy,
  heading,
  secondsRemaining,
  roundStats,
  totalScore,
  timerWarningSeconds = 15,
  logoSrc = "/authentication/worldguesser-logo-transparent.png",
  logoAlt = "WorldGuess",
}: RoundHudProps) {
  const compassOffset = compassTrackOffset(heading);
  const timerLabel = formatRoundTimer(secondsRemaining);
  const timerUrgent = secondsRemaining <= timerWarningSeconds;

  return (
    <header className="absolute inset-x-0 top-0 z-30 flex items-start justify-between p-3 sm:p-5">
      <Link
        href={backHref as Route}
        aria-label={copy.backToMission}
        className="relative block h-10 w-40 drop-shadow-lg"
      >
        <Image
          src={logoSrc}
          alt={logoAlt}
          fill
          sizes="156px"
          className="object-contain object-left"
          priority
        />
      </Link>
      <div className="flex flex-col items-center gap-3">
        <div
          aria-label={copy.mapLabel}
          className="relative hidden h-8 w-60 overflow-hidden rounded-full bg-[#2d3c5c]/80 text-xs font-black tracking-[.3em] shadow-lg backdrop-blur sm:block"
        >
          <div
            className="absolute top-0 left-1/2 flex h-full items-center text-white/75 will-change-transform"
            style={{ transform: `translateX(${compassOffset}px)` }}
          >
            {COMPASS_TICKS.map((index) => {
              const label = compassTickLabel(index);
              return (
                <span
                  key={index}
                  className="relative flex h-full shrink-0 items-center justify-center"
                  style={{ width: COMPASS_TICK_WIDTH_PX }}
                >
                  {label ? (
                    <strong className="absolute top-2 text-[10px] tracking-normal text-white">
                      {label}
                    </strong>
                  ) : (
                    <i className="h-3 border-s border-white/45" />
                  )}
                </span>
              );
            })}
          </div>
          <span
            className="absolute inset-x-0 top-0 z-10 mx-auto h-2 w-px bg-white/80"
            aria-hidden="true"
          />
        </div>
        <div
          className={`rounded-full border-[5px] bg-[#262143]/95 px-7 py-1.5 text-xl font-black italic shadow-xl ${timerUrgent ? "border-[#f23945]" : "border-[#7046d9]"}`}
        >
          {timerLabel}
        </div>
      </div>
      <div className="flex overflow-hidden rounded-2xl border-2 border-[#5e3ba8] bg-[#160d3f]/95 shadow-2xl">
        {roundStats.map((stat) => (
          <RoundStat
            key={stat.roundNumber}
            active={stat.active}
            label={`R${stat.roundNumber}`}
            value={stat.value}
          />
        ))}
        <RoundStat
          label={copy.total}
          value={totalScore.toLocaleString(locale)}
        />
      </div>
    </header>
  );
}
