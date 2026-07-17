import Link from "next/link";
import type { HTMLAttributes, ReactNode } from "react";

export const AUTHENTICATED_ASSETS = {
  /** Blue full globe + red pin (home-loggedin.png singleplayer tile) */
  singleplayer: "/authenticated-home/globe.png",
  multiplayer: "/authenticated-home/multiplayer.png",
  party: "/authenticated-home/goldencrown.png",
  quiz: "/authenticated-home/quiz-bubble.svg",
  premium: "/authenticated-home/pricing.png",
  calendar: "/authenticated-home/calender-achivements.png",
  players: "/authenticated-home/players-today.png",
  /** Map thumbs — landmark scenes matching design (not landing marketing art) */
  mapWorld: "/authenticated-home/map-world.png",
  mapFamous: "/authenticated-home/map-famous.png",
  mapUsa: "/authenticated-home/map-usa.png",
  mapEurope: "/authenticated-home/map-europe.png",
  /** Sidebar / premium promo globe with pin */
  premiumGlobe: "/authenticated-home/earth-marked.png",
} as const;

/** Page canvas — deep ink with soft purple bloom (home-loggedin.png) */
export const PAGE_BG =
  "bg-[#07061A] [background-image:radial-gradient(ellipse_70%_55%_at_42%_28%,#3B1D7A_0%,#1A0F42_38%,#0C0A24_68%,#07061A_100%)]";

/**
 * Primary CTA purple — sampled PLAY pills on home-loggedin.png.
 */
export const PLAY_GRADIENT =
  "bg-[#4A3498] [background-image:linear-gradient(180deg,#5C45B0_0%,#4A3498_42%,#3A2884_100%)]";

/** Soft play-mode tile (glass purple) */
export const PLAY_TILE =
  "bg-[#32247A]/90 [background-image:linear-gradient(165deg,#3F2F94_0%,#32247A_52%,#281C66_100%)]";

/** Rail / panel card chrome */
export const CARD_SURFACE =
  "rounded-2xl border border-white/[0.08] bg-[#0C0E28]/92 shadow-[0_12px_36px_rgba(2,2,20,0.48)] backdrop-blur-md";

export const RAIL_CARD =
  "rounded-[1.1rem] border border-white/[0.08] bg-[#0B0D26]/96 shadow-[0_10px_28px_rgba(2,4,24,0.5)]";

export const PLAY_CTA_SHADOW =
  "shadow-[0_6px_16px_rgba(18,8,55,0.5),inset_0_1px_0_rgba(255,255,255,0.1)]";

export type PlayCtaHref =
  | "#"
  | "#play"
  | "#plans"
  | "#stats"
  | "#maps"
  | "#modes"
  | "#friends"
  | "#profile";

export function Panel({
  children,
  className = "",
  ...rest
}: {
  children: ReactNode;
  className?: string;
} & HTMLAttributes<HTMLElement>) {
  return (
    <section className={`${CARD_SURFACE} ${className}`} {...rest}>
      {children}
    </section>
  );
}

export function PlayCta({
  href,
  children,
  className = "",
}: {
  href: PlayCtaHref;
  children: ReactNode;
  className?: string;
}) {
  return (
    <Link
      href={href}
      className={`${PLAY_GRADIENT} ${PLAY_CTA_SHADOW} inline-flex items-center justify-center rounded-full px-5 py-2.5 text-center text-[0.7rem] font-black tracking-[0.07em] text-white uppercase transition hover:brightness-110 ${className}`}
    >
      {children}
    </Link>
  );
}
