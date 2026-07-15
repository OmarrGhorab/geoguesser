import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import { TrendingUp, Zap } from "lucide-react";
import type { AuthenticatedHomeCopy } from "@/features/home/types";
import type { AppLocale } from "@/lib/i18n/routing";
import {
  DAILY_TODAY,
  DAILY_WEEK_DAYS,
} from "@/features/home/layout-contract";
import {
  AUTHENTICATED_ASSETS,
  PLAY_CTA_SHADOW,
  PLAY_GRADIENT,
  PlayCta,
  RAIL_CARD,
} from "./shared";

const avatarFocus = [
  "18% 45%",
  "50% 45%",
  "82% 45%",
  "50% 45%",
] as const;

const friendRings = [
  "ring-[#F5C542]/80",
  "ring-[#A78BFA]/80",
  "ring-[#D4A574]/80",
  "ring-[#5EC8FF]/80",
] as const;

function AvatarFace({
  index,
  sizeClass,
  ringClass = "",
}: {
  index: number;
  sizeClass: string;
  ringClass?: string;
}) {
  const pos = avatarFocus[index % avatarFocus.length];
  return (
    <span
      className={`${sizeClass} shrink-0 rounded-full bg-cover bg-no-repeat ring-2 ${ringClass}`}
      style={{
        backgroundImage: `url(${AUTHENTICATED_ASSETS.players})`,
        backgroundPosition: pos,
        backgroundSize: "280%",
      }}
      aria-hidden="true"
    />
  );
}

type AuthenticatedHomeRailProps = Readonly<{
  locale: AppLocale;
  copy: AuthenticatedHomeCopy;
}>;

/** Right rail densified to fit the viewport without Y-scroll. */
export function AuthenticatedHomeRail({
  locale,
  copy,
}: AuthenticatedHomeRailProps) {
  const dailyMissionHref = `/${locale}/daily-mission` as Route;
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
            <span className="flex items-center gap-0.5 text-[0.85rem] font-bold text-[#C4B5FD]">
              <Zap
                className="size-3 fill-[#A78BFA] text-[#A78BFA]"
                aria-hidden="true"
              />
              0
            </span>
            <p className="mt-0.5 text-[0.55rem] text-white/45">
              {copy.daily.streak}
            </p>
          </div>
        </div>

        <div className="mt-2.5 text-center">
          <p className="text-[0.72rem] font-semibold text-white">
            {copy.daily.month}
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
            {DAILY_WEEK_DAYS.map((day) => {
              const isToday = day === DAILY_TODAY;
              return (
                <span
                  key={day}
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
              <AvatarFace
                key={i}
                index={i}
                sizeClass="size-5"
                ringClass="ring-[#0B0D26]"
              />
            ))}
          </div>
          <p className="text-[0.6rem] leading-snug text-white/50">
            {copy.daily.playersToday}
          </p>
        </div>

        <div className="mt-2.5 flex items-center justify-between gap-2">
          <div className="min-w-0">
            <p className="text-[0.62rem] text-white/50">{copy.daily.ends}</p>
            <p className="text-[1.25rem] leading-none font-bold tracking-tight text-[#F5B942]">
              13:23:44
            </p>
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
        className={`${RAIL_CARD} shrink-0 px-3 pt-2.5 pb-3`}
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
            <p className="mt-0.5 text-base font-bold text-white">248</p>
          </div>
          <div className="border-x border-white/[0.08] px-0.5">
            <p className="text-[0.55rem] leading-tight text-white/45">
              {copy.stats.streak}
            </p>
            <p className="mt-0.5 text-base font-bold text-white">7</p>
          </div>
          <div className="px-0.5">
            <p className="text-[0.55rem] leading-tight text-white/45">
              {copy.stats.best}
            </p>
            <p className="mt-0.5 text-base font-bold text-white">24,680</p>
          </div>
        </div>
        <PlayCta href="#stats" className="mt-2.5 w-full py-1.5 text-[0.62rem]">
          {copy.stats.cta}
        </PlayCta>
      </section>

      <section
        data-section="friendsOnline"
        className={`${RAIL_CARD} min-h-0 flex-1 overflow-hidden px-3 pt-2.5 pb-2`}
      >
        <div className="mb-0.5 flex items-center justify-between">
          <h2 className="text-[0.95rem] font-bold text-white">
            {copy.friendsOnline}
          </h2>
          <Link
            href="#friends"
            className="text-[0.58rem] font-bold tracking-[0.1em] text-[#A78BFA] transition hover:text-white"
          >
            {copy.seeAll}
          </Link>
        </div>
        <ul>
          {copy.friends.map((friend, index) => (
            <li key={friend.name}>
              <Link
                href="#friends"
                className="flex items-center gap-2 rounded-lg px-0.5 py-1.5 transition hover:bg-white/[0.03]"
              >
                <AvatarFace
                  index={index}
                  sizeClass="size-7"
                  ringClass={friendRings[index % friendRings.length]}
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[0.72rem] font-semibold text-white">
                    {friend.name}
                  </span>
                  <span className="block text-[0.58rem] text-white/45">
                    {friend.status}
                  </span>
                </span>
                <span
                  className="size-2 shrink-0 rounded-[3px] bg-[#22C55E] shadow-[0_0_8px_rgba(34,197,94,0.55)]"
                  aria-hidden="true"
                  data-online-marker
                />
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
