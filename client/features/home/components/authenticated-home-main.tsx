import Image from "next/image";
import Link from "next/link";
import { ChevronRight, Gamepad2, Shield, Swords } from "lucide-react";
import type { AuthenticatedHomeCopy } from "@/features/home/types";
import {
  GAME_MODE_KEYS,
  PLAY_MODE_KEYS,
  RECOMMENDED_MAP_KEYS,
} from "@/features/home/layout-contract";
import {
  AUTHENTICATED_ASSETS,
  PLAY_TILE,
  Panel,
  PlayCta,
} from "./shared";

const mapThumbnails = [
  AUTHENTICATED_ASSETS.mapWorld,
  AUTHENTICATED_ASSETS.mapFamous,
  AUTHENTICATED_ASSETS.mapUsa,
  AUTHENTICATED_ASSETS.mapEurope,
] as const;

const modeIcons = [
  { Icon: Shield, well: "from-[#8B6CFF] to-[#4C35C4]" },
  { Icon: Swords, well: "from-[#5EC8FF] to-[#2B78D4]" },
  { Icon: Gamepad2, well: "from-[#FF8AB5] to-[#D14A78]" },
] as const;

type AuthenticatedHomeMainProps = Readonly<{
  copy: AuthenticatedHomeCopy;
}>;

function isEasyDifficulty(value: string) {
  const v = value.toLowerCase();
  return v === "easy" || v === "سهل";
}

/** Main column densified to fit the viewport without Y-scroll. */
export function AuthenticatedHomeMain({ copy }: AuthenticatedHomeMainProps) {
  return (
    <div
      data-testid="auth-home-main"
      className="flex h-full min-h-0 flex-col overflow-hidden"
    >
      <header data-section="welcome" className="mb-2 shrink-0">
        <h1 className="text-[1.35rem] font-bold tracking-tight italic text-white sm:text-[1.55rem]">
          {copy.welcome}
        </h1>
        <p className="mt-0.5 text-[0.8rem] text-white/55">{copy.question}</p>
      </header>

      <div
        data-section="playModes"
        className="grid shrink-0 grid-cols-2 gap-2 xl:grid-cols-4 xl:gap-2.5"
      >
        {PLAY_MODE_KEYS.map((key) => {
          const item = copy.play[key];
          return (
            <article
              key={key}
              data-play-mode={key}
              className={`group flex flex-col overflow-hidden rounded-xl border border-white/[0.12] ${PLAY_TILE} px-2.5 pt-2.5 pb-2.5 text-center shadow-[0_10px_28px_rgba(25,12,70,0.4)]`}
            >
              <div className="relative mx-auto h-[4.25rem] w-full sm:h-[4.75rem]">
                <Image
                  src={AUTHENTICATED_ASSETS[key]}
                  alt=""
                  fill
                  sizes="(max-width: 1280px) 45vw, 180px"
                  className="object-contain drop-shadow-[0_10px_20px_rgba(0,0,0,0.4)] transition duration-300 group-hover:scale-105"
                  priority={key === "singleplayer"}
                />
              </div>
              <h2 className="mt-1 text-[0.85rem] font-bold text-white">
                {item.title}
              </h2>
              <p className="mx-auto mt-0.5 line-clamp-2 min-h-[1.75rem] max-w-[11rem] text-[0.62rem] leading-snug text-white/65">
                {item.description}
              </p>
              <PlayCta href="#play" className="mt-2 w-full py-1.5 text-[0.62rem]">
                {item.cta}
              </PlayCta>
            </article>
          );
        })}
      </div>

      <Panel
        data-section="premiumBanner"
        className="mt-2.5 shrink-0 overflow-hidden border-white/[0.1] p-0"
      >
        <div className="grid min-h-[7.5rem] grid-cols-1 sm:grid-cols-[40%_1fr]">
          <div className="relative min-h-[7.5rem] overflow-hidden bg-[#1a1248]">
            <Image
              src={AUTHENTICATED_ASSETS.premium}
              alt=""
              fill
              sizes="(max-width: 640px) 100vw, 40vw"
              className="object-cover object-[center_28%]"
              priority
            />
            <span className="absolute inset-0 bg-gradient-to-r from-transparent via-transparent to-[#27156a]/95 max-sm:bg-gradient-to-t max-sm:from-[#0C0E28] max-sm:via-transparent max-sm:to-transparent" />
          </div>
          <div className="relative flex flex-col justify-center bg-gradient-to-br from-[#2E1874] via-[#3A2088] to-[#1A1148] px-4 py-3 sm:px-5 sm:py-3.5">
            <h2 className="text-base font-bold italic leading-tight text-white sm:text-lg">
              {copy.premium.title}
            </h2>
            <ul className="mt-1.5 space-y-0.5 text-[0.72rem] text-white/90">
              {copy.premium.benefits.map((benefit) => (
                <li key={benefit} className="flex items-start gap-1.5">
                  <span
                    className="mt-0.5 font-bold text-[#3DDC84]"
                    aria-hidden="true"
                  >
                    ✓
                  </span>
                  {benefit}
                </li>
              ))}
            </ul>
            <div className="mt-2.5 flex flex-wrap items-center justify-between gap-2">
              <span className="text-[0.7rem] text-white/50">
                {copy.premium.price}
              </span>
              <PlayCta href="#plans" className="min-w-[7rem] px-5 py-1.5 text-[0.62rem]">
                {copy.premium.cta}
              </PlayCta>
            </div>
          </div>
        </div>
      </Panel>

      <section data-section="recommended" className="mt-2.5 min-h-0 shrink">
        <div className="mb-1.5 flex items-center justify-between">
          <h2 className="text-[0.95rem] font-bold text-white">
            {copy.recommended}
          </h2>
          <Link
            href="#maps"
            className="text-[0.6rem] font-bold tracking-[0.1em] text-[#B8A6FF] transition hover:text-white"
          >
            {copy.seeAll}
          </Link>
        </div>
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          {copy.maps.map((map, index) => {
            const mapKey = RECOMMENDED_MAP_KEYS[index] ?? `map-${index}`;
            return (
              <Panel
                key={map.title}
                data-map={mapKey}
                className="group overflow-hidden p-0 transition hover:border-white/18"
              >
                <div className="relative h-[3.75rem] overflow-hidden sm:h-[4rem]">
                  <Image
                    src={mapThumbnails[index] ?? AUTHENTICATED_ASSETS.mapWorld}
                    alt=""
                    fill
                    sizes="180px"
                    className="object-cover transition duration-300 group-hover:scale-105"
                  />
                  <span className="absolute inset-0 bg-gradient-to-t from-[#0a0c24] via-[#0a0c24]/20 to-transparent" />
                  <span className="absolute top-1.5 right-1.5 grid size-6 place-items-center rounded-full bg-black/40 text-white backdrop-blur-sm">
                    <ChevronRight className="size-3.5" aria-hidden="true" />
                  </span>
                </div>
                <div className="px-2.5 pt-1.5 pb-2">
                  <h3 className="text-[0.75rem] font-bold text-white">
                    {map.title}
                  </h3>
                  <p
                    className={`text-[0.62rem] font-semibold ${
                      isEasyDifficulty(map.difficulty)
                        ? "text-[#3DDC84]"
                        : "text-[#F5A623]"
                    }`}
                  >
                    {map.difficulty}
                  </p>
                  <p className="text-[0.58rem] text-white/45">{map.players}</p>
                  <PlayCta
                    href="#play"
                    className="mt-1.5 w-full py-1 text-[0.58rem]"
                  >
                    {copy.play.singleplayer.cta}
                  </PlayCta>
                </div>
              </Panel>
            );
          })}
        </div>
      </section>

      <section
        data-section="gameModes"
        className="mt-2.5 min-h-0 shrink overflow-hidden"
      >
        <div className="mb-1.5 flex items-center justify-between">
          <h2 className="text-[0.95rem] font-bold text-white">{copy.modes}</h2>
          <Link
            href="#modes"
            className="text-[0.6rem] font-bold tracking-[0.1em] text-[#B8A6FF] transition hover:text-white"
          >
            {copy.seeAll}
          </Link>
        </div>
        <Panel className="p-1.5 sm:p-2">
          <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
            {copy.modeRows.slice(0, 2).map((mode, index) => {
              const { Icon, well } = modeIcons[index];
              const modeKey = GAME_MODE_KEYS[index] ?? `mode-${index}`;
              const showMeta = index === 1;
              return (
                <Link
                  href="#play"
                  key={mode.title}
                  data-game-mode={modeKey}
                  className="flex items-center gap-2.5 rounded-xl border border-white/[0.06] bg-white/[0.02] px-2 py-2 transition hover:bg-white/[0.05]"
                >
                  <span
                    className={`grid size-9 shrink-0 place-items-center rounded-xl bg-gradient-to-br ${well} shadow-[0_4px_12px_rgba(0,0,0,0.35)]`}
                  >
                    <Icon className="size-4 text-white" strokeWidth={2} />
                  </span>
                  <span className="min-w-0 flex-1">
                    <strong className="block text-[0.78rem] font-bold text-white">
                      {mode.title}
                    </strong>
                    <small className="mt-0.5 block truncate text-[0.62rem] text-white/50">
                      {mode.description}
                    </small>
                  </span>
                  {showMeta ? (
                    <>
                      <span className="hidden shrink-0 text-[0.68rem] font-semibold text-white/65 sm:block">
                        {mode.players}
                      </span>
                      <ChevronRight className="size-4 shrink-0 text-white/65" />
                    </>
                  ) : null}
                </Link>
              );
            })}
          </div>
          {copy.modeRows[2] ? (
            <Link
              href="#play"
              data-game-mode={GAME_MODE_KEYS[2] ?? "team"}
              className="mt-1.5 flex items-center gap-2.5 rounded-xl border border-white/[0.06] bg-white/[0.02] px-2 py-2 transition hover:bg-white/[0.05]"
            >
              <span
                className={`grid size-9 shrink-0 place-items-center rounded-xl bg-gradient-to-br ${modeIcons[2].well} shadow-[0_4px_12px_rgba(0,0,0,0.35)]`}
              >
                {(() => {
                  const Icon = modeIcons[2].Icon;
                  return <Icon className="size-4 text-white" strokeWidth={2} />;
                })()}
              </span>
              <span className="min-w-0 flex-1">
                <strong className="block text-[0.78rem] font-bold text-white">
                  {copy.modeRows[2].title}
                </strong>
                <small className="mt-0.5 block truncate text-[0.62rem] text-white/50">
                  {copy.modeRows[2].description}
                </small>
              </span>
              <span className="hidden shrink-0 text-[0.68rem] font-semibold text-white/65 sm:block">
                {copy.modeRows[2].players}
              </span>
              <ChevronRight className="size-4 shrink-0 text-white/65" />
            </Link>
          ) : null}
        </Panel>
      </section>
    </div>
  );
}
