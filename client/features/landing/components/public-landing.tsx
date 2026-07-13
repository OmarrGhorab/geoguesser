import Image from "next/image";
import { Link } from "@/lib/i18n/navigation";
import type { AppLocale } from "@/lib/i18n/routing";
import { cn } from "@/lib/utils";
import { LandingReveal } from "./landing-reveal";

export type LandingCopy = {
  nav: {
    explore: string;
    multiplayer: string;
    leaderboards: string;
    pricing: string;
    login: string;
    playFree: string;
  };
  sections: {
    explore: { title: string; description: string };
    discover: { title: string; description: string };
    friends: { title: string; description: string };
    compete: { title: string; description: string };
  };
};

/** Shared horizontal alignment for header + section content */
const landingContainerClassName =
  "mx-auto w-full max-w-[96rem] px-[clamp(1.25rem,3.5vw,3rem)]";

type LandingSectionProps = {
  id: string;
  background:
    | { type: "image"; src: string; alt: string }
    | { type: "video"; src: string; poster: string };
  title: string;
  description: string;
  imageClassName?: string;
  characterSrc?: string;
  characterAlt?: string;
  /** Wider character art (e.g. two-character pair) above the text stack */
  characterWide?: boolean;
  /** Extra top padding for the hero under the absolute header */
  withHeaderOffset?: boolean;
  priority?: boolean;
};

function LandingSection({
  id,
  background,
  title,
  description,
  imageClassName,
  characterSrc,
  characterAlt,
  characterWide,
  withHeaderOffset,
  priority,
}: LandingSectionProps) {
  return (
    <LandingReveal
      id={id}
      className="landing-section relative isolate h-[95vh] min-h-[28rem] overflow-hidden border-t border-white/15"
      labelledBy={`${id}-title`}
    >
      {background.type === "video" ? (
        <video
          className={cn(
            "absolute inset-0 size-full object-cover",
            imageClassName,
          )}
          autoPlay
          loop
          muted
          playsInline
          preload="metadata"
          poster={background.poster}
          aria-hidden="true"
        >
          <source src={background.src} type="video/mp4" />
        </video>
      ) : (
        <Image
          src={background.src}
          alt={background.alt}
          fill
          priority={priority}
          sizes="100vw"
          className={cn("object-cover", imageClassName)}
        />
      )}
      <div className="absolute inset-0 bg-[linear-gradient(90deg,rgba(1,4,29,0.78)_0%,transparent_42%,transparent_68%,rgba(1,4,29,0.2)_100%)] sm:bg-[linear-gradient(90deg,rgba(1,4,29,0.55)_0%,transparent_48%)]" />

      <div
        className={cn(
          landingContainerClassName,
          "relative z-10 flex h-full min-h-[inherit] flex-col items-start justify-center py-16 sm:py-20",
          withHeaderOffset && "pt-36 sm:pt-40",
        )}
      >
        {characterSrc && characterAlt ? (
          <div
            className={cn(
              "relative mb-2 shrink-0 overflow-hidden",
              characterWide
                ? "aspect-[5/6] w-[min(72vw,22rem)] sm:w-[min(34vw,24rem)]"
                : "aspect-[3/4] w-[min(42vw,11rem)] sm:w-[min(18vw,13.5rem)]",
            )}
          >
            <Image
              src={characterSrc}
              alt={characterAlt}
              fill
              sizes={
                characterWide
                  ? "(max-width: 640px) 72vw, 24rem"
                  : "(max-width: 640px) 42vw, 220px"
              }
              className="pointer-events-none object-cover object-top drop-shadow-[0_24px_28px_rgba(0,0,0,0.35)] select-none"
            />
          </div>
        ) : null}

        <div className="w-full max-w-[min(100%,48rem)] text-white drop-shadow-[0_3px_3px_rgba(0,0,0,0.55)]">
          <h2
            id={`${id}-title`}
            className="font-auth-display text-[clamp(1.75rem,4.2vw,4.25rem)] leading-none font-black tracking-[0.01em] whitespace-nowrap uppercase italic"
          >
            {title}
          </h2>
          <p className="mt-3 max-w-[30rem] text-[clamp(1rem,1.6vw,1.4rem)] leading-snug font-medium text-pretty text-white/95">
            {description}
          </p>
        </div>
      </div>
    </LandingReveal>
  );
}

export function PublicLanding({
  copy,
  locale,
}: {
  copy: LandingCopy;
  locale: AppLocale;
}) {
  const otherLocale: AppLocale = locale === "en" ? "ar" : "en";
  const localeLabel = locale.toUpperCase();

  return (
    <main
      data-testid="public-landing"
      className="landing-page min-h-dvh overflow-x-clip bg-[#020522] text-white"
    >
      <div className="relative">
        <header
          className={cn(
            landingContainerClassName,
            "absolute inset-x-0 top-0 z-20 flex h-20 items-center justify-between gap-4 sm:h-24",
          )}
        >
          <Link
            href="/"
            className="relative h-16 w-[43px] shrink-0 rounded-sm focus-visible:ring-2 focus-visible:ring-[#5865f2] focus-visible:ring-offset-2 focus-visible:ring-offset-[#020522] focus-visible:outline-none sm:h-20 sm:w-[54px]"
            aria-label="WorldGuesser home"
          >
            <Image
              src="/logo-3.png"
              alt=""
              fill
              sizes="(min-width: 640px) 54px, 43px"
              className="object-contain"
              preload
            />
          </Link>

          <nav
            aria-label="Landing page"
            className="hidden items-center gap-6 text-sm font-bold lg:flex lg:gap-8 xl:gap-10"
          >
            <a className="landing-nav-link" href="#explore">
              {copy.nav.explore}
            </a>
            <a className="landing-nav-link" href="#friends">
              {copy.nav.multiplayer}
            </a>
            <a className="landing-nav-link" href="#compete">
              {copy.nav.leaderboards}
            </a>
            <a className="landing-nav-link" href="#pricing">
              {copy.nav.pricing}
            </a>
          </nav>

          <div className="flex shrink-0 items-center gap-2 sm:gap-3">
            <Link
              href="/"
              locale={otherLocale}
              className="landing-nav-link rounded-sm px-2 py-1 text-xs font-bold tracking-wide sm:text-sm"
              aria-label={`Switch language to ${otherLocale.toUpperCase()}`}
            >
              {localeLabel}
            </Link>
            <Link
              href="/login"
              className="landing-nav-link rounded-full px-3 py-1.5 text-xs font-bold sm:px-4 sm:text-sm"
            >
              {copy.nav.login}
            </Link>
            <Link
              href="/sign-up"
              className="rounded-full bg-[#5865f2] px-3 py-1.5 text-xs font-bold text-white shadow-[0_0_20px_rgba(88,101,242,0.45)] transition-colors hover:bg-[#4752c4] focus-visible:ring-2 focus-visible:ring-[#5865f2] focus-visible:ring-offset-2 focus-visible:ring-offset-[#020522] focus-visible:outline-none sm:px-4 sm:py-2 sm:text-sm"
            >
              {copy.nav.playFree}
            </Link>
          </div>
        </header>

        <LandingSection
          id="explore"
          background={{
            type: "video",
            src: "/landing/hero-vid.mp4",
            poster: "/landing/explore-background.png",
          }}
          title={copy.sections.explore.title}
          description={copy.sections.explore.description}
          imageClassName="object-[58%_center] sm:object-center"
          withHeaderOffset
          priority
        />
      </div>

      <LandingSection
        id="discover"
        background={{
          type: "image",
          src: "/landing/section2-bg.png",
          alt: "A world map connecting destinations across the globe",
        }}
        title={copy.sections.discover.title}
        description={copy.sections.discover.description}
        imageClassName="object-[57%_center] sm:object-center"
        characterSrc="/landing/character-secondsection.png"
        characterAlt="A traveler holding a glowing map"
      />

      <LandingSection
        id="friends"
        background={{
          type: "image",
          src: "/landing/section3-bg.png",
          alt: "A moonlit mountain valley ready to explore with friends",
        }}
        title={copy.sections.friends.title}
        description={copy.sections.friends.description}
        imageClassName="object-[62%_center] sm:object-center"
        characterSrc="/landing/friends-city.png"
        characterAlt="Two friends fist-bumping"
        characterWide
      />

      <LandingSection
        id="compete"
        background={{
          type: "image",
          src: "/landing/section4-bg.png",
          alt: "A golden WorldGuesser competition crest and leaderboard arena",
        }}
        title={copy.sections.compete.title}
        description={copy.sections.compete.description}
        imageClassName="object-[62%_center] sm:object-center"
        characterSrc="/landing/compete-arena.png"
        characterAlt="Two competitors facing off in a friendly challenge"
        characterWide
      />
    </main>
  );
}
