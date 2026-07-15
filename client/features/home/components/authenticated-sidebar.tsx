import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import {
  BarChart3,
  Home,
  Settings,
  Trophy,
  UserRound,
  UsersRound,
} from "lucide-react";
import type { AppLocale } from "@/lib/i18n/routing";
import type {
  AuthenticatedNavCopy,
  AuthenticatedPremiumCopy,
  AuthenticatedSidebarItem,
} from "@/features/home/types";
import { AUTHENTICATED_SIDE_NAV } from "@/features/home/nav-config";
import {
  AUTHENTICATED_ASSETS,
  PLAY_CTA_SHADOW,
  PLAY_GRADIENT,
  PlayCta,
} from "./shared";

const SIDEBAR_ICONS: Record<
  AuthenticatedSidebarItem,
  typeof Home
> = {
  home: Home,
  profile: UserRound,
  stats: BarChart3,
  friends: UsersRound,
  missions: Trophy,
  settings: Settings,
};

type AuthenticatedSidebarProps = Readonly<{
  locale?: AppLocale;
  nav: AuthenticatedNavCopy;
  premium: Pick<
    AuthenticatedPremiumCopy,
    "cta" | "sidebarTitle" | "sidebarBody"
  >;
  dashboardAriaLabel: string;
  activeItem?: AuthenticatedSidebarItem;
  /** When false, hides the bottom Go Premium promo. */
  showPremiumPromo?: boolean;
}>;

/**
 * Left column — titles for backend-backed destinations (profiles, friends, challenges).
 */
export function AuthenticatedSidebar({
  locale = "en",
  nav,
  premium,
  dashboardAriaLabel,
  activeItem = "home",
  showPremiumPromo = true,
}: AuthenticatedSidebarProps) {
  const labelFor = (id: AuthenticatedSidebarItem): string => {
    switch (id) {
      case "home":
        return nav.home;
      case "profile":
        return nav.profile;
      case "stats":
        return nav.stats;
      case "friends":
        return nav.friends;
      case "missions":
        return nav.missions;
      case "settings":
        return nav.settings;
      default:
        return id;
    }
  };

  return (
    <aside className="hidden h-full min-h-0 flex-col lg:flex">
      <nav
        aria-label={dashboardAriaLabel}
        className="flex shrink-0 flex-col"
      >
        {AUTHENTICATED_SIDE_NAV.map(({ id }) => {
          const isActive = id === activeItem;
          const Icon = SIDEBAR_ICONS[id];
          const path = (
            id === "home" ? `/${locale}` : `#${id}`
          ) as Route;
          return (
            <Link
              key={id}
              href={path}
              className={`flex items-center gap-3 rounded-xl px-3 py-[0.55rem] text-[0.8rem] font-semibold transition ${
                isActive
                  ? `${PLAY_GRADIENT} ${PLAY_CTA_SHADOW} text-white`
                  : "text-white/70 hover:bg-white/[0.06] hover:text-white"
              }`}
            >
              <Icon
                className={`size-5 shrink-0 ${isActive ? "text-white" : "text-white/75"}`}
                strokeWidth={2}
              />
              {labelFor(id)}
            </Link>
          );
        })}
      </nav>

      {showPremiumPromo ? (
        <div className="mt-auto flex shrink-0 flex-col pt-4">
          <div className="relative mx-auto h-[6.75rem] w-[6.75rem]">
            <Image
              src={AUTHENTICATED_ASSETS.premiumGlobe}
              alt=""
              fill
              sizes="108px"
              className="object-contain drop-shadow-[0_14px_28px_rgba(0,0,0,0.5)]"
              priority
            />
          </div>
          <h2 className="mt-0.5 text-[0.95rem] font-bold italic leading-snug text-white">
            {premium.sidebarTitle}
          </h2>
          <p className="mt-1.5 text-[0.68rem] leading-snug text-white/55">
            {premium.sidebarBody}
          </p>
          <PlayCta href="#plans" className="mt-3 w-full py-2 text-[0.68rem]">
            {premium.cta}
          </PlayCta>
        </div>
      ) : null}
    </aside>
  );
}
