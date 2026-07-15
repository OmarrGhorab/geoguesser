import Image from "next/image";
import Link from "next/link";
import type { Route } from "next";
import { ChevronDown, Menu, Search } from "lucide-react";
import type { AppLocale } from "@/lib/i18n/routing";
import type {
  AuthenticatedAriaCopy,
  AuthenticatedNavCopy,
  AuthenticatedProfileCopy,
} from "@/features/home/types";
import {
  AUTHENTICATED_TOP_NAV,
  type AuthenticatedTopNavId,
} from "@/features/home/nav-config";
import { AUTHENTICATED_ASSETS } from "./shared";

type AuthenticatedNavbarProps = Readonly<{
  locale: AppLocale;
  brand: string;
  nav: AuthenticatedNavCopy;
  profile: AuthenticatedProfileCopy;
  aria: AuthenticatedAriaCopy;
  /** Active top-nav id (backend-backed). Defaults to multiplayer. */
  activeTopNavId?: AuthenticatedTopNavId;
}>;

export function AuthenticatedNavbar({
  locale,
  brand,
  nav,
  profile,
  aria,
  activeTopNavId = "multiplayer",
}: AuthenticatedNavbarProps) {
  const labelFor = (id: AuthenticatedTopNavId): string => {
    switch (id) {
      case "singleplayer":
        return nav.singleplayer;
      case "multiplayer":
        return nav.multiplayer;
      case "party":
        return nav.party;
      case "challenges":
        return nav.challenges;
      case "maps":
        return nav.maps;
      case "leaderboards":
        return nav.leaderboards;
      case "friends":
        return nav.friends;
      default:
        return id;
    }
  };

  return (
    <header className="z-30 flex h-[4.5rem] shrink-0 items-center gap-4 border-b border-white/[0.07] bg-[#0A0918]/95 px-4 backdrop-blur-xl lg:px-7">
      <Link href={`/${locale}`} aria-label={brand} className="shrink-0">
        <Image
          src="/logo-2.png"
          alt={brand}
          width={154}
          height={48}
          className="h-auto w-[8.5rem] object-contain"
          priority
        />
      </Link>

      <nav
        aria-label={aria.primary}
        className="hidden min-w-0 flex-1 items-center gap-0.5 text-[0.7rem] font-bold tracking-[0.04em] text-white/70 uppercase xl:flex"
      >
        {AUTHENTICATED_TOP_NAV.map((item) => {
          const isActive = item.id === activeTopNavId;
          const label = labelFor(item.id);
          // Frontend routes not all shipped yet — titles map to backend domains.
          const href = `#${item.id}` as Route;
          return (
            <Link
              key={item.id}
              href={href}
              title={label}
              className={`relative flex items-center gap-1 px-3 py-4 whitespace-nowrap transition hover:text-white ${
                isActive ? "text-white" : ""
              }`}
            >
              {label}
              {item.hasDropdown ? (
                <ChevronDown className="size-3 opacity-70" aria-hidden="true" />
              ) : null}
              {isActive ? (
                <span
                  className="absolute inset-x-2 bottom-0 h-[3px] rounded-full bg-[#FF2D8A]"
                  aria-hidden="true"
                />
              ) : null}
            </Link>
          );
        })}
      </nav>

      <div className="ml-auto flex items-center gap-3">
        <button
          type="button"
          aria-label={aria.search}
          className="hidden rounded-full p-2 text-white/75 transition hover:bg-white/10 hover:text-white sm:block"
        >
          <Search className="size-5" strokeWidth={2} />
        </button>

        <div className="hidden items-center gap-2 border-l border-white/10 pl-3 text-sm font-bold text-white sm:flex">
          <span
            className="grid size-7 place-items-center rounded-full bg-gradient-to-br from-[#FFE08A] to-[#F5A623] text-[0.7rem] text-[#3a1f00] shadow-[0_0_14px_rgba(245,166,35,0.5)]"
            aria-hidden="true"
          >
            ●
          </span>
          {profile.credits}
        </div>

        <Link
          href={"#profile" as Route}
          className="flex items-center gap-2.5 border-l border-white/10 pl-3"
          aria-label={profile.name}
        >
          <span className="relative size-9 overflow-hidden rounded-full border-2 border-[#F5C542] shadow-[0_0_16px_rgba(245,197,66,0.4)]">
            <Image
              src={AUTHENTICATED_ASSETS.players}
              alt=""
              fill
              sizes="36px"
              className="object-cover object-[50%_42%]"
            />
          </span>
          <span className="hidden leading-tight sm:block">
            <strong className="block text-xs font-bold text-white">
              {profile.name}
            </strong>
            <small className="text-[0.68rem] text-white/55">{profile.level}</small>
          </span>
          <ChevronDown className="hidden size-4 text-white/55 sm:block" />
        </Link>

        <button
          type="button"
          aria-label={aria.menu}
          className="rounded-lg p-2 text-white/80 transition hover:bg-white/10 xl:hidden"
        >
          <Menu className="size-5" />
        </button>
      </div>
    </header>
  );
}
