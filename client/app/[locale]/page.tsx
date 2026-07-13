import { cookies } from "next/headers";
import { getTranslations, setRequestLocale } from "next-intl/server";
import {
  PublicLanding,
  type LandingCopy,
} from "@/features/landing/components/public-landing";
import { Link } from "@/lib/i18n/navigation";
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
        pricing: t("nav.pricing"),
        login: t("nav.login"),
        playFree: t("nav.playFree"),
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

  return (
    <main className="grid min-h-dvh place-items-center p-[var(--spacing-page)]">
      <div className="flex flex-col items-center gap-6 text-center">
        <h1 className="m-0 text-[clamp(2rem,6vw,4rem)] font-medium tracking-tight">
          {t("title")}
        </h1>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/sign-up"
            className="rounded-full bg-white px-6 py-2.5 text-sm font-medium text-black transition-colors hover:bg-neutral-200"
          >
            {t("signUpCta")}
          </Link>
          <Link
            href="/login"
            className="rounded-full border border-white/15 bg-transparent px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-white/10"
          >
            {t("logInCta")}
          </Link>
        </div>
      </div>
    </main>
  );
}
