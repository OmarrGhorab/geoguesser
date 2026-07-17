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
import { getAuthenticatedHome } from "@/features/home/data";
import { loadHomeDecision } from "@/features/home/resolve-home";
import type { AppLocale } from "@/lib/i18n/routing";

type HomePageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

async function buildLandingCopy(
  t: Awaited<ReturnType<typeof getTranslations>>,
  appLocale: AppLocale,
): Promise<LandingCopy> {
  return {
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
}

async function buildDashboardCopy(
  dashboard: Awaited<ReturnType<typeof getTranslations>>,
): Promise<AuthenticatedHomeCopy> {
  return {
    brand: dashboard("brand"),
    // ICU placeholders are filled later from API data — use raw templates.
    welcome: dashboard.raw("welcome") as string,
    question: dashboard("question"),
    languageLabel: dashboard("languageLabel"),
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
    modes: dashboard("modes.title"),
    modeRows: ["battle", "duels", "team"].map((key) => ({
      title: dashboard(`modes.${key}.title`),
      description: dashboard(`modes.${key}.description`),
    })),
    daily: {
      title: dashboard("daily.title"),
      streak: dashboard("daily.streak"),
      ends: dashboard("daily.ends"),
      cta: dashboard("play.cta"),
      playersToday: dashboard.raw("daily.playersToday") as string,
      days: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"].map((day) =>
        dashboard(`daily.days.${day}`),
      ),
    },
    stats: {
      title: dashboard("stats.title"),
      played: dashboard("stats.played"),
      average: dashboard("stats.average"),
      best: dashboard("stats.best"),
      cta: dashboard("stats.cta"),
    },
    aria: {
      primary: dashboard("aria.primary"),
      dashboard: dashboard("aria.dashboard"),
      search: dashboard("aria.search"),
      menu: dashboard("aria.menu"),
    },
  };
}

function HomeUnavailable({
  title,
  body,
  retry,
}: {
  title: string;
  body: string;
  retry: string;
}) {
  return (
    <main
      data-testid="home-unavailable"
      className="grid min-h-dvh place-items-center bg-[#07061A] px-6 text-center text-white"
    >
      <div className="max-w-md space-y-3">
        <h1 className="text-2xl font-bold">{title}</h1>
        <p className="text-sm text-white/65">{body}</p>
        <a
          href=""
          className="inline-flex rounded-full bg-[#6B4EFF] px-6 py-2.5 text-sm font-bold tracking-wide uppercase"
        >
          {retry}
        </a>
      </div>
    </main>
  );
}

export default async function HomePage({ params }: HomePageProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);
  const t = await getTranslations("Home");
  const cookieStore = await cookies();
  const hasAuthCookie = Boolean(
    cookieStore.get("access_token")?.value ||
      cookieStore.get("refresh_token")?.value,
  );

  const decision = await loadHomeDecision(hasAuthCookie, getAuthenticatedHome);

  if (decision.kind === "public") {
    return (
      <PublicLanding
        copy={await buildLandingCopy(t, appLocale)}
        locale={appLocale}
      />
    );
  }

  if (decision.kind === "unavailable") {
    const dashboard = await getTranslations("AuthenticatedHome");
    return (
      <HomeUnavailable
        title={dashboard("errors.unavailableTitle")}
        body={dashboard("errors.unavailableBody")}
        retry={dashboard("errors.retry")}
      />
    );
  }

  const dashboard = await getTranslations("AuthenticatedHome");
  const dashboardCopy = await buildDashboardCopy(dashboard);
  return (
    <AuthenticatedHome
      locale={appLocale}
      copy={dashboardCopy}
      data={decision.data}
    />
  );
}
