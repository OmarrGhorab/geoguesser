"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { Route } from "next";
import {
  createRoomAction,
  joinRoomAction,
} from "@/features/rooms/actions";
import type { PlayableMap } from "@/features/play/schemas";
import type { AppLocale } from "@/lib/i18n/routing";

export type RoomEntryCopy = Readonly<{
  title: string;
  subtitle: string;
  createTitle: string;
  joinTitle: string;
  mapLabel: string;
  codeLabel: string;
  codePlaceholder: string;
  create: string;
  creating: string;
  join: string;
  joining: string;
  back: string;
  noMaps: string;
  errors: {
    validation: string;
    unavailable: string;
    notFound: string;
    hostActionRequired: string;
    generic: string;
  };
}>;

type RoomEntryScreenProps = Readonly<{
  locale: AppLocale;
  maps: PlayableMap[];
  copy: RoomEntryCopy;
  homeHref: Route | string;
}>;

function errorMessage(code: string, copy: RoomEntryCopy): string {
  switch (code) {
    case "validation_failed":
      return copy.errors.validation;
    case "not_found":
    case "room_not_found":
      return copy.errors.notFound;
    case "host_action_required":
      return copy.errors.hostActionRequired;
    case "unavailable":
      return copy.errors.unavailable;
    default:
      return copy.errors.generic;
  }
}

export function RoomEntryScreen({
  locale,
  maps,
  copy,
  homeHref,
}: RoomEntryScreenProps) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const activeMaps = maps.filter((m) => m.status === "active");
  const selectable = activeMaps.length > 0 ? activeMaps : maps;
  const [mapId, setMapId] = useState(selectable[0]?.id ?? "");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);

  const create = () => {
    setError(null);
    startTransition(async () => {
      if (!mapId) {
        setError(copy.errors.validation);
        return;
      }
      const result = await createRoomAction(
        {
          mapId,
          roundCount: 5,
          timerSeconds: 60,
          maxPlayers: 8,
          idempotencyKey: `room-create-${crypto.randomUUID()}`,
        },
        locale,
      );
      if (!result.ok) {
        setError(errorMessage(result.code, copy));
        return;
      }
      router.push(`/${locale}/rooms/${result.roomCode}` as Route);
    });
  };

  const join = () => {
    setError(null);
    startTransition(async () => {
      const result = await joinRoomAction(
        { code: code.trim().toUpperCase() },
        locale,
      );
      if (!result.ok) {
        setError(errorMessage(result.code, copy));
        return;
      }
      router.push(`/${locale}/rooms/${result.roomCode}` as Route);
    });
  };

  return (
    <main className="min-h-dvh bg-[#08051d] px-4 py-8 text-white sm:px-8">
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-8">
        <Link
          href={homeHref as Route}
          className="text-sm font-semibold text-violet-200 underline-offset-4 hover:underline"
        >
          {copy.back}
        </Link>
        <header className="space-y-2">
          <h1 className="text-3xl font-black tracking-tight sm:text-4xl">
            {copy.title}
          </h1>
          <p className="text-sm text-white/70">{copy.subtitle}</p>
        </header>

        <section className="space-y-4 rounded-2xl border border-white/10 bg-white/5 p-5">
          <h2 className="text-sm font-black tracking-wide text-violet-200 uppercase">
            {copy.createTitle}
          </h2>
          {selectable.length === 0 ? (
            <p className="text-sm text-amber-200">{copy.noMaps}</p>
          ) : (
            <label className="block space-y-2">
              <span className="text-xs font-black text-white/70 uppercase">
                {copy.mapLabel}
              </span>
              <select
                value={mapId}
                onChange={(e) => setMapId(e.target.value)}
                className="w-full rounded-xl border border-white/15 bg-[#120b2e] px-3 py-3 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-violet-400"
              >
                {selectable.map((map) => (
                  <option key={map.id} value={map.id}>
                    {map.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          <button
            type="button"
            onClick={create}
            disabled={isPending || selectable.length === 0}
            className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-3 text-xs font-black tracking-wide uppercase disabled:opacity-45"
          >
            {isPending ? copy.creating : copy.create}
          </button>
        </section>

        <section className="space-y-4 rounded-2xl border border-white/10 bg-white/5 p-5">
          <h2 className="text-sm font-black tracking-wide text-violet-200 uppercase">
            {copy.joinTitle}
          </h2>
          <label className="block space-y-2">
            <span className="text-xs font-black text-white/70 uppercase">
              {copy.codeLabel}
            </span>
            <input
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              placeholder={copy.codePlaceholder}
              maxLength={12}
              className="w-full rounded-xl border border-white/15 bg-[#120b2e] px-3 py-3 text-sm font-semibold tracking-widest uppercase outline-none focus-visible:ring-2 focus-visible:ring-violet-400"
            />
          </label>
          <button
            type="button"
            onClick={join}
            disabled={isPending || code.trim().length < 4}
            className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-3 text-xs font-black tracking-wide uppercase disabled:opacity-45"
          >
            {isPending ? copy.joining : copy.join}
          </button>
        </section>

        {error ? (
          <p role="alert" className="text-sm font-semibold text-red-300">
            {error}
          </p>
        ) : null}
      </div>
    </main>
  );
}
