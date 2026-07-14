import Image from "next/image";
import { LandingFooter } from "./landing-footer";
import SplitText from "./landing-split-text";
import { Link } from "@/lib/i18n/navigation";
import type { AppLocale } from "@/lib/i18n/routing";
import { cn } from "@/lib/utils";
import type { SectionRevealDirection } from "@/features/landing/motion";
import { LandingHeroVideo } from "./landing-hero-video";
import { LandingMediaHover } from "./landing-media-hover";
import { LandingReveal } from "./landing-reveal";
import { LandingScrollRoot } from "./landing-scroll-root";

export type LandingCopy = {
  nav: {
    explore: string;
    multiplayer: string;
    leaderboards: string;
    login: string;
    playFree: string;
  };
  aria: {
    home: string;
    landingNavigation: string;
    switchLanguage: string;
    footerNavigation: string;
  };
  media: {
    heroLogo: string;
    discoverBackground: string;
    discoverCharacter: string;
    friendsCharacter: string;
    competeBackground: string;
    competeCharacter: string;
  };
  footer: {
    tagline: string;
    play: string;
    account: string;
    game: string;
    copyright: string;
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
    | { type: "video"; src: string }
    | { type: "color"; color: string };
  title: string;
  description: string;
  imageClassName?: string;
  /** Background image opacity 0–1 (image backgrounds only) */
  backgroundOpacity?: number;
  characterSrc?: string;
  characterAlt?: string;
  /** Wider character art (e.g. two-character pair) above the text stack */
  characterWide?: boolean;
  /** Landscape character art that must remain fully visible */
  characterLandscape?: boolean;
  /** Extra top padding for the hero under the absolute header */
  withHeaderOffset?: boolean;
  /** Optional brand lockup above the section title (hero) */
  brandLogoSrc?: string;
  brandLogoAlt?: string;
  priority?: boolean;
  /** Skip scroll motion (hero only). */
  skipMotion?: boolean;
  /** Entrance direction for post-hero sections. */
  direction?: SectionRevealDirection;
};

function LandingSection({
  id,
  background,
  title,
  description,
  imageClassName,
  backgroundOpacity,
  characterSrc,
  characterAlt,
  characterWide,
  characterLandscape,
  withHeaderOffset,
  brandLogoSrc,
  brandLogoAlt,
  priority,
  skipMotion = false,
  direction = "up",
}: LandingSectionProps) {
  const isSolidColor = background.type === "color";

  return (
    <LandingReveal
      id={id}
      className={cn(
        "landing-section relative isolate min-h-[28rem] overflow-hidden",
        characterLandscape ? "h-auto min-h-[95vh]" : "h-[95vh]",
      )}
      labelledBy={`${id}-title`}
      skipMotion={skipMotion}
      direction={direction}
    >
      {background.type === "video" ? (
        <LandingHeroVideo
          src={background.src}
          className={imageClassName}
          opacity={backgroundOpacity}
        />
      ) : background.type === "color" ? (
        <div
          className="absolute inset-0"
          style={{ backgroundColor: background.color }}
          aria-hidden="true"
        />
      ) : (
        <>
          {/* Base so reduced-opacity art doesn’t flash empty */}
          <div className="absolute inset-0 bg-[#04141a]" aria-hidden="true" />
          <div
            className="absolute inset-0"
            style={
              backgroundOpacity !== undefined
                ? { opacity: backgroundOpacity }
                : undefined
            }
            data-landing-parallax={skipMotion ? undefined : ""}
          >
            <Image
              src={background.src}
              alt={background.alt}
              fill
              priority={priority}
              sizes="100vw"
              className={cn("object-cover", imageClassName)}
            />
          </div>
        </>
      )}
      {!isSolidColor ? (
        <div className="absolute inset-0 bg-[linear-gradient(90deg,rgba(1,4,29,0.78)_0%,transparent_42%,transparent_68%,rgba(1,4,29,0.2)_100%)] sm:bg-[linear-gradient(90deg,rgba(1,4,29,0.55)_0%,transparent_48%)]" />
      ) : null}

      <div
        data-landing-content=""
        className={cn(
          landingContainerClassName,
          "relative z-10 flex h-full min-h-[inherit] flex-col items-start justify-center py-16 sm:py-20",
          characterLandscape && "items-center py-8 sm:py-10",
          withHeaderOffset && "pt-36 sm:pt-40",
        )}
      >
        {characterSrc && characterAlt ? (
          <LandingMediaHover
            enabled={!skipMotion}
            className={cn(
              "relative mb-2 shrink-0 overflow-hidden",
              characterLandscape
                ? "mb-0 aspect-[2/1] w-[min(92vw,64rem)]"
                : characterWide
                  ? "aspect-[5/6] w-[min(72vw,22rem)] sm:w-[min(34vw,24rem)]"
                  : "aspect-[3/4] w-[min(42vw,11rem)] sm:w-[min(18vw,13.5rem)]",
            )}
          >
            <div data-landing-media="" className="absolute inset-0">
              <Image
                src={characterSrc}
                alt={characterAlt}
                fill
                sizes={
                  characterLandscape
                    ? "(max-width: 640px) 92vw, 64rem"
                    : characterWide
                      ? "(max-width: 640px) 72vw, 24rem"
                      : "(max-width: 640px) 42vw, 220px"
                }
                className={cn(
                  "pointer-events-none object-top drop-shadow-[0_24px_28px_rgba(0,0,0,0.35)] select-none",
                  "object-cover",
                )}
              />
            </div>
          </LandingMediaHover>
        ) : null}

        <div
          className={cn(
            "w-full text-white drop-shadow-[0_3px_3px_rgba(0,0,0,0.55)]",
            brandLogoSrc
              ? "max-w-[min(100%,72rem)]"
              : "max-w-[min(100%,48rem)]",
            characterLandscape && "mt-10 w-fit max-w-full shrink-0 text-center",
          )}
        >
          {brandLogoSrc ? (
            <div className="relative mb-1 h-[clamp(2.75rem,7vw,4.5rem)] w-full max-w-[min(100%,36rem)] sm:mb-1.5">
              <Image
                src={brandLogoSrc}
                alt={brandLogoAlt ?? ""}
                fill
                priority={priority}
                sizes="(max-width: 640px) 80vw, 36rem"
                className="object-contain object-bottom object-left drop-shadow-[0_8px_24px_rgba(0,40,120,0.5)]"
              />
            </div>
          ) : null}
          {skipMotion ? (
            <>
              <h2
                id={`${id}-title`}
                className="font-auth-display text-[clamp(1.75rem,4.2vw,4.25rem)] leading-none font-black tracking-[0.01em] whitespace-nowrap uppercase italic"
              >
                {title}
              </h2>
              <p
                className={cn(
                  "mt-3 max-w-[30rem] text-[clamp(1rem,1.6vw,1.4rem)] leading-snug font-medium text-pretty text-white/95",
                  characterLandscape && "mt-1 max-w-none",
                )}
              >
                {description}
              </p>
            </>
          ) : (
            <>
              <SplitText
                tag="h2"
                id={`${id}-title`}
                text={title}
                textAlign={characterLandscape ? "center" : undefined}
                className="font-auth-display text-[clamp(1.75rem,4.2vw,4.25rem)] leading-none font-black tracking-[0.01em] whitespace-nowrap uppercase italic"
                delay={40}
                duration={0.55}
                ease="power3.out"
                splitType="chars"
                from={{ opacity: 0, y: 40 }}
                to={{ opacity: 1, y: 0 }}
                // Match section media reveal: fire around top 80% of viewport
                threshold={0.2}
                rootMargin="0px"
                replay
              />
              <SplitText
                tag="p"
                text={description}
                textAlign={characterLandscape ? "center" : undefined}
                className={cn(
                  "mt-3 max-w-[30rem] text-[clamp(1rem,1.6vw,1.4rem)] leading-snug font-medium text-pretty text-white/95",
                  characterLandscape && "mt-1 max-w-none",
                )}
                delay={30}
                duration={0.5}
                ease="power3.out"
                splitType="words"
                from={{ opacity: 0, y: 28 }}
                to={{ opacity: 1, y: 0 }}
                threshold={0.2}
                rootMargin="0px"
                replay
              />
            </>
          )}
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
    <LandingScrollRoot>
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
              aria-label={copy.aria.home}
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
              aria-label={copy.aria.landingNavigation}
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
            </nav>

            <div className="flex shrink-0 items-center gap-2 sm:gap-3">
              <Link
                href="/"
                locale={otherLocale}
                className="landing-nav-link rounded-sm px-2 py-1 text-xs font-bold tracking-wide sm:text-sm"
                aria-label={copy.aria.switchLanguage}
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
            }}
            title={copy.sections.explore.title}
            description={copy.sections.explore.description}
            imageClassName="object-[58%_center] sm:object-center"
            backgroundOpacity={0.72}
            brandLogoSrc="/logo-2.png"
            brandLogoAlt={copy.media.heroLogo}
            withHeaderOffset
            priority
            skipMotion
          />
        </div>

        <LandingSection
          id="discover"
          background={{
            type: "image",
            src: "/landing/section2-bg.png",
            alt: copy.media.discoverBackground,
          }}
          title={copy.sections.discover.title}
          description={copy.sections.discover.description}
          imageClassName="object-[57%_center] sm:object-center"
          backgroundOpacity={0.55}
          characterSrc="/landing/character-secondsection.png"
          characterAlt={copy.media.discoverCharacter}
          direction="left"
        />

        <LandingSection
          id="friends"
          background={{
            type: "color",
            color: "#0f5cad",
          }}
          title={copy.sections.friends.title}
          description={copy.sections.friends.description}
          characterSrc="/landing/friends-city.png"
          characterAlt={copy.media.friendsCharacter}
          characterWide
          direction="right"
        />

        <LandingSection
          id="compete"
          background={{
            type: "image",
            src: "/landing/section4-bg.png",
            alt: copy.media.competeBackground,
          }}
          title={copy.sections.compete.title}
          description={copy.sections.compete.description}
          imageClassName="object-[62%_center] sm:object-center"
          backgroundOpacity={0.42}
          characterSrc="/landing/compete-arena.png"
          characterAlt={copy.media.competeCharacter}
          characterLandscape
          direction="up"
        />

        <div data-landing-footer="">
          <LandingFooter
            brandName="WorldGuess"
            tagline={copy.footer.tagline}
            logoSrc="/logo-3.png"
            homeHref={`/${locale}`}
            copyright={copy.footer.copyright}
            navigationLabel={copy.aria.footerNavigation}
            columns={[
              {
                title: copy.footer.play,
                links: [
                  { label: copy.nav.explore, href: "#explore" },
                  { label: copy.nav.multiplayer, href: "#friends" },
                  { label: copy.nav.leaderboards, href: "#compete" },
                  { label: copy.nav.playFree, href: `/${locale}/sign-up` },
                ],
              },
              {
                title: copy.footer.account,
                links: [
                  { label: copy.nav.login, href: `/${locale}/login` },
                  { label: copy.nav.playFree, href: `/${locale}/sign-up` },
                ],
              },
              {
                title: copy.footer.game,
                links: [
                  { label: copy.sections.discover.title, href: "#discover" },
                  { label: copy.sections.friends.title, href: "#friends" },
                  { label: copy.sections.compete.title, href: "#compete" },
                ],
              },
            ]}
          />
        </div>
      </main>
    </LandingScrollRoot>
  );
}
