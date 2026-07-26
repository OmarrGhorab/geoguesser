"use client";

import { useState, type ComponentType } from "react";
import type { Route } from "next";
import Image from "next/image";
import Link from "next/link";
import { Popover } from "@base-ui/react/popover";
import {
  ChevronDown,
  Compass,
  LayoutGrid,
  Search,
  Trophy,
  UsersRound,
  Zap,
} from "lucide-react";
import {
  AUTHENTICATED_GAME_MODE_GROUPS,
  AUTHENTICATED_GAME_MODES,
  type AuthenticatedGameModeGroupId,
} from "@/features/home/nav-config";
import type { AuthenticatedGameModesCopy } from "@/features/home/types";
import type { AppLocale } from "@/lib/i18n/routing";
import { cn } from "@/lib/utils";

type GameModeMenuProps = Readonly<{
  locale: AppLocale;
  label: string;
  copy: AuthenticatedGameModesCopy;
  active: boolean;
}>;

const groupIcons: Record<
  AuthenticatedGameModeGroupId,
  ComponentType<{ className?: string; "aria-hidden"?: boolean }>
> = {
  soloAdventures: Compass,
  quickMatch: Zap,
  competitive: Trophy,
  withFriends: UsersRound,
};

const toneStyles = {
  solo: {
    card: "border-violet-400/20 hover:border-violet-300/65 hover:bg-violet-400/10 focus-visible:ring-violet-300",
    glow: "bg-violet-400/15",
  },
  casual: {
    card: "border-cyan-400/20 hover:border-cyan-300/65 hover:bg-cyan-400/10 focus-visible:ring-cyan-300",
    glow: "bg-cyan-400/15",
  },
  ranked: {
    card: "border-amber-400/30 hover:border-amber-300/75 hover:bg-amber-400/10 focus-visible:ring-amber-300",
    glow: "bg-amber-400/15",
  },
  party: {
    card: "border-pink-400/25 hover:border-pink-300/70 hover:bg-pink-400/10 focus-visible:ring-pink-300",
    glow: "bg-pink-400/15",
  },
} as const;

function groupColumns(columns: number): string {
  switch (columns) {
    case 4:
      return "grid-cols-2 sm:grid-cols-4";
    case 3:
      return "grid-cols-2 sm:grid-cols-3";
    default:
      return "grid-cols-1 sm:grid-cols-2";
  }
}

export function GameModeMenu({
  locale,
  label,
  copy,
  active,
}: GameModeMenuProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const normalizedQuery = query.trim().toLocaleLowerCase(locale);

  const visibleModes = AUTHENTICATED_GAME_MODES.filter((mode) => {
    if (!normalizedQuery) return true;
    const text = `${copy.items[mode.id].title} ${copy.items[mode.id].description}`;
    return text.toLocaleLowerCase(locale).includes(normalizedQuery);
  });

  return (
    <Popover.Root
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) setQuery("");
      }}
    >
      <Popover.Trigger
        className={cn(
          "group relative flex items-center gap-1.5 rounded-xl border px-3.5 py-2.5 whitespace-nowrap transition-colors",
          "border-transparent text-white/75 hover:border-white/10 hover:bg-white/[0.06] hover:text-white",
          "focus-visible:ring-2 focus-visible:ring-violet-300 focus-visible:ring-offset-2 focus-visible:ring-offset-[#0A0918] focus-visible:outline-none",
          "data-popup-open:border-violet-400/55 data-popup-open:bg-violet-400/10 data-popup-open:text-white",
          active && "text-white",
        )}
      >
        <LayoutGrid className="size-3.5" aria-hidden="true" />
        {label}
        <ChevronDown
          className="size-3.5 opacity-70 transition-transform duration-200 group-data-[popup-open]:rotate-180 motion-reduce:transition-none"
          aria-hidden="true"
        />
        {active ? (
          <span
            className="absolute inset-x-2 -bottom-[0.92rem] h-[3px] rounded-full bg-[#FF2D8A]"
            aria-hidden="true"
          />
        ) : null}
      </Popover.Trigger>

      <Popover.Portal>
        <Popover.Positioner
          side="bottom"
          align="start"
          sideOffset={8}
          collisionPadding={16}
          className="z-50"
        >
          <Popover.Popup
            className={cn(
              "max-h-[calc(100vh-6rem)] w-[min(70rem,calc(100vw-2rem))] overflow-y-auto rounded-2xl border border-violet-400/30",
              "origin-[var(--transform-origin)] bg-[#11162F]/98 p-5 text-white shadow-[0_28px_90px_rgba(2,4,24,0.72)] backdrop-blur-2xl",
              "transition-[transform,opacity] duration-150 ease-out data-ending-style:scale-[0.985] data-ending-style:opacity-0 data-starting-style:scale-[0.985] data-starting-style:opacity-0 motion-reduce:transition-none",
              "focus:outline-none",
            )}
          >
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <Popover.Title className="text-xl font-black tracking-tight">
                {copy.title}
              </Popover.Title>

              <label className="relative block w-full sm:max-w-72">
                <span className="sr-only">{copy.searchLabel}</span>
                <Search
                  className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-white/40"
                  aria-hidden="true"
                />
                <input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder={copy.searchPlaceholder}
                  className="h-10 w-full rounded-xl border border-white/10 bg-white/[0.04] ps-10 pe-3 text-sm text-white outline-none placeholder:text-white/35 focus:border-violet-300/70 focus:ring-2 focus:ring-violet-300/20"
                />
              </label>
            </div>

            <nav aria-label={copy.title} className="mt-5">
              <div className="grid gap-x-5 gap-y-6 lg:grid-cols-2">
                {AUTHENTICATED_GAME_MODE_GROUPS.map((group) => {
                  const modes = visibleModes.filter(
                    (mode) => mode.group === group.id,
                  );
                  if (modes.length === 0) return null;
                  const GroupIcon = groupIcons[group.id];

                  return (
                    <section
                      key={group.id}
                      aria-labelledby={`mode-group-${group.id}`}
                    >
                      <div className="mb-2.5 flex items-center gap-2 border-b border-white/[0.08] pb-2.5">
                        <GroupIcon
                          className="size-4 text-violet-300"
                          aria-hidden={true}
                        />
                        <h3
                          id={`mode-group-${group.id}`}
                          className="text-[0.68rem] font-black tracking-[0.09em] text-white/72 uppercase"
                        >
                          {copy.groups[group.id]}
                        </h3>
                      </div>

                      <div
                        className={cn(
                          "grid gap-2.5",
                          groupColumns(group.columns),
                        )}
                      >
                        {modes.map((mode) => {
                          const itemCopy = copy.items[mode.id];
                          const styles = toneStyles[mode.tone];
                          const href = `/${locale}${mode.href}` as Route;
                          const recommended =
                            "recommended" in mode && mode.recommended;

                          return (
                            <Link
                              key={mode.backendMode}
                              href={href}
                              onClick={() => setOpen(false)}
                              className={cn(
                                "group/mode relative flex min-h-36 flex-col items-center justify-center overflow-hidden rounded-xl border bg-white/[0.035] px-2.5 py-3 text-center",
                                "transition-[border-color,background-color,transform] duration-150 hover:-translate-y-0.5 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-[#11162F] focus-visible:outline-none motion-reduce:transform-none motion-reduce:transition-none",
                                styles.card,
                              )}
                            >
                              <span
                                className={cn(
                                  "absolute top-3 size-14 rounded-full blur-2xl transition-opacity group-hover/mode:opacity-100",
                                  styles.glow,
                                )}
                                aria-hidden="true"
                              />
                              {recommended ? (
                                <span className="absolute inset-x-2 top-2 rounded-full bg-cyan-400/15 px-1.5 py-1 text-[0.5rem] font-black tracking-wide text-cyan-200 uppercase">
                                  {copy.recommended}
                                </span>
                              ) : null}
                              <Image
                                src={mode.asset}
                                alt=""
                                width={72}
                                height={72}
                                className={cn(
                                  "relative size-11 object-contain",
                                  recommended && "mt-3",
                                )}
                              />
                              <strong className="relative mt-1.5 text-xs leading-tight font-extrabold text-white">
                                {itemCopy.title}
                              </strong>
                              <span className="relative mt-1 line-clamp-2 text-[0.62rem] leading-snug text-white/50">
                                {itemCopy.description}
                              </span>
                            </Link>
                          );
                        })}
                      </div>
                    </section>
                  );
                })}
              </div>

              {visibleModes.length === 0 ? (
                <p
                  role="status"
                  className="grid min-h-44 place-items-center text-sm text-white/55"
                >
                  {copy.empty}
                </p>
              ) : null}
            </nav>

            <div className="mt-5 flex items-center justify-between border-t border-white/[0.08] pt-4 text-xs text-white/45">
              <Link
                href={`/${locale}/play` as Route}
                onClick={() => setOpen(false)}
                className="inline-flex items-center gap-2 rounded-lg text-white/60 transition hover:text-white focus-visible:ring-2 focus-visible:ring-violet-300 focus-visible:outline-none"
              >
                <LayoutGrid className="size-3.5" aria-hidden="true" />
                {copy.viewAll}
              </Link>
              <span>{copy.escapeHint}</span>
            </div>
          </Popover.Popup>
        </Popover.Positioner>
      </Popover.Portal>
    </Popover.Root>
  );
}
