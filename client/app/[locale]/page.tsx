import { cookies } from "next/headers";
import { getTranslations, setRequestLocale } from "next-intl/server";
import {
  PublicLanding,
  type LandingCopy,
} from "@/features/landing/components/public-landing";
import {
  AuthenticatedHome,
  type AuthenticatedHomeCopy,
} from "@/features/home/components/authenticated-home";
import type { AppLocale } from "@/lib/i18n/routing";

type HomePageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export default async function HomePage({ params }: HomePageProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);
  const t = await getTranslations("Home");
  const cookieStore = await cookies();
  const isAuthenticated = Boolean(
    cookieStore.get("access_token")?.value ||
    cookieStore.get("refresh_token")?.value,
  );

  if (!isAuthenticated) {
    const copy: LandingCopy = {
      nav: {
        explore: t("nav.explore"),
        multiplayer: t("nav.multiplayer"),
        leaderboards: t("nav.leaderboards"),
        login: t("nav.login"),
        playFree: t("nav.playFree"),
      },
      aria: {
        home: t("aria.home"),
        landingNavigation: t("aria.landingNavigation"),
        switchLanguage: t("aria.switchLanguage", {
          locale: appLocale === "en" ? "AR" : "EN",
        }),
        footerNavigation: t("aria.footerNavigation"),
      },
      media: {
        heroLogo: t("media.heroLogo"),
        discoverBackground: t("media.discoverBackground"),
        discoverCharacter: t("media.discoverCharacter"),
        friendsCharacter: t("media.friendsCharacter"),
        competeBackground: t("media.competeBackground"),
        competeCharacter: t("media.competeCharacter"),
      },
      footer: {
        tagline: t("footer.tagline"),
        play: t("footer.play"),
        account: t("footer.account"),
        game: t("footer.game"),
        copyright: t("footer.copyright", { year: new Date().getFullYear() }),
      },
      sections: {
        explore: {
          title: t("sections.explore.title"),
          description: t("sections.explore.description"),
        },
        discover: {
          title: t("sections.discover.title"),
          description: t("sections.discover.description"),
        },
        friends: {
          title: t("sections.friends.title"),
          description: t("sections.friends.description"),
        },
        compete: {
          title: t("sections.compete.title"),
          description: t("sections.compete.description"),
        },
      },
    };

    return <PublicLanding copy={copy} locale={appLocale} />;
  }

  const dashboard = await getTranslations("AuthenticatedHome");
  const dashboardCopy: AuthenticatedHomeCopy = {
    brand: dashboard("brand"),
    welcome: dashboard("welcome"),
    question: dashboard("question"),
    nav: {
      singleplayer: dashboard("nav.singleplayer"),
      multiplayer: dashboard("nav.multiplayer"),
      party: dashboard("nav.party"),
      challenges: dashboard("nav.challenges"),
      maps: dashboard("nav.maps"),
      leaderboards: dashboard("nav.leaderboards"),
      friends: dashboard("nav.friends"),
      home: dashboard("nav.home"),
      profile: dashboard("nav.profile"),
      stats: dashboard("nav.stats"),
      missions: dashboard("nav.missions"),
      settings: dashboard("nav.settings"),
    },
    play: Object.fromEntries(
      ["singleplayer", "multiplayer", "party", "quiz"].map((key) => [
        key,
        {
          title: dashboard(`play.${key}.title`),
          description: dashboard(`play.${key}.description`),
          cta: dashboard("play.cta"),
        },
      ]),
    ) as AuthenticatedHomeCopy["play"],
    premium: {
      title: dashboard("premium.title"),
      body: dashboard("premium.body"),
      sidebarTitle: dashboard("premium.sidebarTitle"),
      sidebarBody: dashboard("premium.sidebarBody"),
      benefits: ["adFree", "maps", "platforms"].map((key) =>
        dashboard(`premium.benefits.${key}`),
      ),
      price: dashboard("premium.price"),
      cta: dashboard("premium.cta"),
    },
    recommended: dashboard("recommended"),
    seeAll: dashboard("seeAll"),
    maps: ["world", "famous", "usa", "europe"].map((key) => ({
      title: dashboard(`maps.${key}.title`),
      difficulty: dashboard(`maps.${key}.difficulty`),
      players: dashboard(`maps.${key}.players`),
    })),
    modes: dashboard("modes.title"),
    modeRows: ["battle", "duels", "team"].map((key) => ({
      title: dashboard(`modes.${key}.title`),
      description: dashboard(`modes.${key}.description`),
      players: dashboard(`modes.${key}.players`),
    })),
    daily: {
      title: dashboard("daily.title"),
      streak: dashboard("daily.streak"),
      month: dashboard("daily.month"),
      days: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"].map((day) =>
        dashboard(`daily.days.${day}`),
      ),
      ends: dashboard("daily.ends"),
      cta: dashboard("play.cta"),
      playersToday: dashboard("daily.playersToday"),
    },
    stats: {
      title: dashboard("stats.title"),
      played: dashboard("stats.played"),
      streak: dashboard("stats.streak"),
      best: dashboard("stats.best"),
      cta: dashboard("stats.cta"),
    },
    friendsOnline: dashboard("friends.title"),
    friends: [
      {
        name: dashboard("friends.mapMaster"),
        status: dashboard("friends.online"),
      },
      {
        name: dashboard("friends.geoWizard"),
        status: dashboard("friends.online"),
      },
      {
        name: dashboard("friends.explorer"),
        status: dashboard("friends.inGame"),
      },
      {
        name: dashboard("friends.worldWalker"),
        status: dashboard("friends.online"),
      },
    ],
    profile: {
      name: dashboard("profile.name"),
      level: dashboard("profile.level"),
      credits: dashboard("profile.credits"),
    },
    languageLabel: dashboard("languageLabel"),
    aria: {
      primary: dashboard("aria.primary"),
      dashboard: dashboard("aria.dashboard"),
      search: dashboard("aria.search"),
      menu: dashboard("aria.menu"),
    },
  };

  return <AuthenticatedHome locale={appLocale} copy={dashboardCopy} />;
}
