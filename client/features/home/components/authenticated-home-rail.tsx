import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import { TrendingUp, Zap } from "lucide-react";
import type { AuthenticatedHomeCopy } from "@/features/home/types";
import type { AuthenticatedHomeData } from "@/features/home/schemas";
import type { AppLocale } from "@/lib/i18n/routing";
import {
  formatAverageScore,
  formatNumber,
  formatPlayedToday,
  weekContaining,
} from "@/features/home/format";
import {
  AUTHENTICATED_ASSETS,
  PLAY_CTA_SHADOW,
  PLAY_GRADIENT,
  PlayCta,
  RAIL_CARD,
} from "./shared";
import { DailyCountdown } from "./daily-countdown";

type AuthenticatedHomeRailProps = Readonly<{
  locale: AppLocale;
  copy: AuthenticatedHomeCopy;
  data: AuthenticatedHomeData;
}>;

/** Right rail: daily challenge + stats (no friends panel until backend supports it). */
export function AuthenticatedHomeRail({
  locale,
  copy,
  data,
}: AuthenticatedHomeRailProps) {
  const dailyMissionHref = `/${locale}/daily-mission` as Route;
  const challengeDate = data.daily_challenge.challenge.challenge_date;
  const week = weekContaining(challengeDate ?? undefined);
  const streak = data.daily_challenge.streak.current_count;
  const participants = data.daily_challenge.leaderboard_summary.participants;
  const playersToday = formatPlayedToday(
    locale,
    copy.daily.playersToday,
    participants,
  );
  const monthLabel = challengeDate
    ? new Intl.DateTimeFormat(locale === "ar" ? "ar" : "en", {
        month: "long",
        year: "numeric",
        timeZone: "UTC",
      }).format(new Date(`${challengeDate}T12:00:00.000Z`))
    : "";

  const gamesPlayed = formatNumber(locale, data.stats.games_played);
  const averageScore = formatAverageScore(locale, data.stats.average_score);
  const bestScore = formatNumber(locale, data.stats.best_score);
  const resetEndsAt = data.daily_challenge.countdown?.reset_ends_at;

  return (
    <div
      data-testid="auth-home-rail"
      className="flex h-full min-h-0 flex-col gap-2 overflow-hidden"
    >
      <section
        data-section="dailyChallenge"
        className={`${RAIL_CARD} shrink-0 px-3 pt-2.5 pb-3`}
      >
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5">
            <span className="relative size-7 shrink-0">
              <Image
                src={AUTHENTICATED_ASSETS.calendar}
                alt=""
                fill
                sizes="28px"
                className="object-contain drop-shadow-[0_4px_10px_rgba(0,0,0,0.35)]"
              />
            </span>
            <h2 className="truncate text-[0.85rem] font-bold text-white">
              {copy.daily.title}
            </h2>
          </div>
          <div className="flex shrink-0 flex-col items-end leading-none">
            <span
              data-testid="daily-streak"
              className="flex items-center gap-0.5 text-[0.85rem] font-bold text-[#C4B5FD]"
            >
              <Zap
                className="size-3 fill-[#A78BFA] text-[#A78BFA]"
                aria-hidden="true"
              />
              {formatNumber(locale, streak)}
            </span>
            <p className="mt-0.5 text-[0.55rem] text-white/45">
              {copy.daily.streak}
            </p>
          </div>
        </div>

        <div className="mt-2.5 text-center">
          <p
            data-testid="daily-month"
            className="text-[0.72rem] font-semibold text-white"
          >
            {monthLabel}
          </p>
          <div className="mt-1.5 grid grid-cols-7 gap-x-0.5 text-[0.52rem] font-semibold tracking-wide text-white/40 uppercase">
            {copy.daily.days.map((day) => (
              <span key={day} className="py-0.5">
                {day.slice(0, 3)}
              </span>
            ))}
          </div>
          <div
            className="mt-0.5 grid grid-cols-7 gap-x-0.5"
            data-testid="daily-week"
          >
            {week.days.map((day) => {
              const isToday = day === week.todayDay;
              return (
                <span
                  key={`${week.year}-${week.monthIndex}-${day}`}
                  data-day={day}
                  data-today={isToday ? "true" : "false"}
                  className={`mx-auto grid size-7 place-items-center rounded-full text-[0.72rem] ${
                    isToday
                      ? "border-2 border-[#7DD3FC] bg-[#0EA5E9]/28 font-bold text-white shadow-[0_0_12px_rgba(56,189,248,0.35)]"
                      : "font-medium text-white/70"
                  }`}
                >
                  {day}
                </span>
              );
            })}
          </div>
        </div>

        <div className="mt-2.5 flex items-center gap-2">
          <div className="flex shrink-0 -space-x-1.5">
            {[0, 1, 2].map((i) => (
              <span
                key={i}
                className="size-5 shrink-0 rounded-full bg-cover bg-no-repeat ring-2 ring-[#0B0D26]"
                style={{
                  backgroundImage: `url(${AUTHENTICATED_ASSETS.players})`,
                  backgroundPosition: ["18% 45%", "50% 45%", "82% 45%"][i],
                  backgroundSize: "280%",
                }}
                aria-hidden="true"
              />
            ))}
          </div>
          <p
            data-testid="daily-players-today"
            className="text-[0.6rem] leading-snug text-white/50"
          >
            {playersToday}
          </p>
        </div>

        <div className="mt-2.5 flex items-center justify-between gap-2">
          <div className="min-w-0">
            <p className="text-[0.62rem] text-white/50">{copy.daily.ends}</p>
            {resetEndsAt ? (
              <DailyCountdown
                resetEndsAt={resetEndsAt}
                className="text-[1.25rem] leading-none font-bold tracking-tight text-[#F5B942]"
              />
            ) : (
              <p
                data-testid="daily-countdown"
                className="text-[1.25rem] leading-none font-bold tracking-tight text-[#F5B942]"
              >
                --:--:--
              </p>
            )}
          </div>
          <Link
            href={dailyMissionHref}
            data-testid="daily-challenge-play"
            className={`${PLAY_GRADIENT} ${PLAY_CTA_SHADOW} inline-flex min-w-[4.5rem] shrink-0 items-center justify-center rounded-full px-4 py-1.5 text-center text-[0.6rem] font-black tracking-[0.07em] text-white uppercase transition hover:brightness-110`}
          >
            {copy.daily.cta}
          </Link>
        </div>
      </section>

      <section
        data-section="yourStats"
        className={`${RAIL_CARD} min-h-0 flex-1 px-3 pt-2.5 pb-3`}
      >
        <div className="flex items-center justify-between">
          <h2 className="text-[0.95rem] font-bold text-white">
            {copy.stats.title}
          </h2>
          <TrendingUp
            className="size-3.5 text-[#8B7CF6]"
            strokeWidth={2.25}
            aria-hidden="true"
          />
        </div>
        <div className="mt-2.5 grid grid-cols-3 text-center">
          <div className="px-0.5">
            <p className="text-[0.55rem] leading-tight text-white/45">
              {copy.stats.played}
            </p>
            <p
              data-testid="stats-games-played"
              className="mt-0.5 text-base font-bold text-white"
            >
              {gamesPlayed}
            </p>
          </div>
          <div className="border-x border-white/[0.08] px-0.5">
            <p className="text-[0.55rem] leading-tight text-white/45">
              {copy.stats.average}
            </p>
            <p
              data-testid="stats-average-score"
              className="mt-0.5 text-base font-bold text-white"
            >
              {averageScore}
            </p>
          </div>
          <div className="px-0.5">
            <p className="text-[0.55rem] leading-tight text-white/45">
              {copy.stats.best}
            </p>
            <p
              data-testid="stats-best-score"
              className="mt-0.5 text-base font-bold text-white"
            >
              {bestScore}
            </p>
          </div>
        </div>
        <PlayCta href="#stats" className="mt-2.5 w-full py-1.5 text-[0.62rem]">
          {copy.stats.cta}
        </PlayCta>
      </section>
    </div>
  );
}
