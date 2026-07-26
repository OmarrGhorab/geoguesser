"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState, useTransition } from "react";
import type { Route } from "next";
import {
  GuessMap,
  MapControls,
  RoundHud,
  SceneViewer,
  type GuessMapHandle,
  type SceneViewerHandle,
} from "@/features/gameplay/components";
import {
  cancelRoomAction,
  leaveRoomSelfAction,
  startRoomAction,
} from "@/features/rooms/actions";
import { connectRoomRealtime } from "@/features/rooms/realtime";
import {
  roomSnapshotSchema,
  type RoomDTO,
} from "@/features/rooms/schemas";
import type { AppLocale } from "@/lib/i18n/routing";
import { formatRoundTimer } from "@/features/mission/round-stats";

export type RoomLobbyCopy = Readonly<{
  title: string;
  codeLabel: string;
  statusLabel: string;
  playersLabel: string;
  host: string;
  you: string;
  leave: string;
  cancel: string;
  start: string;
  starting: string;
  needPlayers: string;
  back: string;
  copyCode: string;
  copied: string;
  waiting: string;
  activeHint: string;
  submitted: string;
  waitingOthers: string;
  placePin: string;
  submitGuess: string;
  submitting: string;
  next: string;
  distance: string;
  correctLocation: string;
  timeExpired: string;
  mapLabel: string;
  total: string;
  zoomIn: string;
  zoomOut: string;
  recenter: string;
  expandMap: string;
  mapUnavailable: string;
  retryMap: string;
  selectLocation: string;
  live: string;
  polling: string;
  errors: {
    hostActionRequired: string;
    unavailable: string;
    needPlayers: string;
    invalidGuess: string;
    generic: string;
  };
}>;

type RoomLobbyScreenProps = Readonly<{
  locale: AppLocale;
  room: RoomDTO;
  googleMapsApiKey: string;
  copy: RoomLobbyCopy;
}>;

type Pin = { latitude: number; longitude: number };
type GuessReveal = {
  guess: {
    latitude: number;
    longitude: number;
    distance_meters: number;
    score: number;
    timed_out?: boolean;
  };
  actual_location: { latitude: number; longitude: number };
  max_score: number;
  score_percent: number;
};

export function RoomLobbyScreen({
  locale,
  room: initialRoom,
  googleMapsApiKey,
  copy,
}: RoomLobbyScreenProps) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [room, setRoom] = useState(initialRoom);
  const [linkStatus, setLinkStatus] = useState<
    "connecting" | "live" | "polling" | "offline"
  >("connecting");

  const sceneRef = useRef<SceneViewerHandle>(null);
  const guessMapRef = useRef<GuessMapHandle>(null);
  const [pin, setPin] = useState<Pin | null>(null);
  const [mapExpanded, setMapExpanded] = useState(false);
  const [heading, setHeading] = useState(0);
  const [reveal, setReveal] = useState<GuessReveal | null>(null);
  const [guessLocked, setGuessLocked] = useState(false);
  const [secondsRemaining, setSecondsRemaining] = useState(0);

  const players = room.players ?? [];
  const playerCount = players.length;
  const status = room.status;
  const canStartByRules = status === "lobby" && playerCount >= 2;
  const isActive = status === "active";
  const gameId = room.game_id ?? null;
  const currentRound = room.current_round ?? null;
  const guessProgress = room.guess_progress ?? null;

  const isHost =
    (room.current_player_id != null &&
      room.host_player_id != null &&
      room.current_player_id === room.host_player_id) ||
    players.some(
      (p) =>
        p.role === "host" &&
        room.current_player_id != null &&
        p.id === room.current_player_id,
    ) ||
    (room.current_player_id == null &&
      players.length === 1 &&
      players[0]?.role === "host");

  const applyRoom = useCallback((next: RoomDTO) => {
    setRoom(next);
    if (next.status === "active" && next.current_round?.ends_at) {
      const ends = new Date(next.current_round.ends_at).getTime();
      setSecondsRemaining(Math.max(0, Math.ceil((ends - Date.now()) / 1000)));
    }
    // Unlock guess on new round.
    if (next.current_round?.id) {
      setGuessLocked(false);
      setReveal(null);
      setPin(null);
    }
  }, []);

  const pollRoom = useCallback(async () => {
    try {
      const res = await fetch(`/api/rooms/${encodeURIComponent(room.code)}`, {
        cache: "no-store",
      });
      if (!res.ok) return;
      const json: unknown = await res.json();
      const parsed = roomSnapshotSchema.safeParse(json);
      if (!parsed.success) return;
      setRoom((prev) => {
        const next = parsed.data.room;
        // Only reset guess UI when round id changes.
        if (prev.current_round?.id !== next.current_round?.id) {
          setGuessLocked(false);
          setReveal(null);
          setPin(null);
          setError(null);
        }
        if (next.current_round?.ends_at) {
          const ends = new Date(next.current_round.ends_at).getTime();
          setSecondsRemaining(
            Math.max(0, Math.ceil((ends - Date.now()) / 1000)),
          );
        }
        return next;
      });
    } catch {
      // keep last good snapshot
    }
  }, [room.code]);

  useEffect(() => {
    const handle = connectRoomRealtime({
      roomCode: room.code,
      pollMs: 2_000,
      onPollTick: () => {
        void pollRoom();
      },
      onStatus: setLinkStatus,
    });
    return () => handle.stop();
  }, [room.code, pollRoom]);

  useEffect(() => {
    if (!isActive || !currentRound?.ends_at || reveal) return;
    const timer = window.setInterval(() => {
      setSecondsRemaining((s) => Math.max(0, s - 1));
    }, 1_000);
    return () => window.clearInterval(timer);
  }, [isActive, currentRound?.ends_at, currentRound?.id, reveal]);

  const leave = () => {
    setError(null);
    startTransition(async () => {
      const result = await leaveRoomSelfAction(room.code);
      if (!result.ok) {
        if (result.code === "host_action_required") {
          setError(copy.errors.hostActionRequired);
          return;
        }
        setError(copy.errors.generic);
        return;
      }
      router.replace(`/${locale}/rooms` as Route);
    });
  };

  const cancel = () => {
    setError(null);
    startTransition(async () => {
      const result = await cancelRoomAction(room.code);
      if (!result.ok) {
        setError(copy.errors.generic);
        return;
      }
      router.replace(`/${locale}/rooms` as Route);
    });
  };

  const start = () => {
    setError(null);
    if (!canStartByRules) {
      setError(copy.errors.needPlayers);
      return;
    }
    startTransition(async () => {
      const result = await startRoomAction(room.code);
      if (!result.ok) {
        if (
          result.code === "validation_failed" ||
          result.code === "need_players" ||
          result.code === "invalid_request"
        ) {
          setError(copy.errors.needPlayers);
          return;
        }
        setError(copy.errors.generic);
        return;
      }
      if (result.room) {
        applyRoom(result.room);
      }
      await pollRoom();
    });
  };

  const submitGuess = () => {
    if (!gameId || !currentRound || !pin || guessLocked) return;
    if (secondsRemaining === 0) {
      setError(copy.timeExpired);
      return;
    }
    setError(null);
    startTransition(async () => {
      try {
        const res = await fetch(
          `/api/games/${gameId}/rounds/${currentRound.id}/guesses`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              latitude: pin.latitude,
              longitude: pin.longitude,
              idempotencyKey: crypto.randomUUID(),
            }),
          },
        );
        const json: unknown = await res.json().catch(() => null);
        if (!res.ok) {
          setError(copy.errors.invalidGuess);
          return;
        }
        setGuessLocked(true);
        // Multiplayer: answer withheld until shared reveal.
        const body = json as {
          actual_location?: { latitude: number; longitude: number } | null;
          guess?: GuessReveal["guess"];
          max_score?: number;
          score_percent?: number;
          round_completed?: boolean;
        };
        if (body.actual_location && body.guess) {
          setReveal({
            guess: body.guess,
            actual_location: body.actual_location,
            max_score: body.max_score ?? 5000,
            score_percent: body.score_percent ?? 0,
          });
        }
        await pollRoom();
      } catch {
        setError(copy.errors.unavailable);
      }
    });
  };

  // When everyone has submitted, fetch shared reveal.
  useEffect(() => {
    if (
      !isActive ||
      !gameId ||
      !currentRound ||
      !guessProgress ||
      reveal ||
      guessProgress.submitted_count < guessProgress.eligible_count ||
      guessProgress.eligible_count < 1
    ) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetch(
          `/api/games/${gameId}/rounds/${currentRound.id}/results`,
          { cache: "no-store" },
        );
        if (!res.ok || cancelled) return;
        const data = (await res.json()) as {
          actual_location?: { latitude: number; longitude: number };
          guesses?: Array<{
            guess: GuessReveal["guess"];
            game_player_id?: string;
          }>;
        };
        if (!data.actual_location) return;
        const mine =
          data.guesses?.find(
            (g) =>
              room.current_player_id &&
              g.game_player_id === room.current_player_id,
          )?.guess ?? data.guesses?.[0]?.guess;
        if (!mine) return;
        setReveal({
          guess: mine,
          actual_location: data.actual_location,
          max_score: 5000,
          score_percent: Math.round((mine.score / 5000) * 100),
        });
      } catch {
        // ignore; poll will retry
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [
    isActive,
    gameId,
    currentRound,
    guessProgress,
    reveal,
    room.current_player_id,
  ]);

  const copyCode = async () => {
    try {
      await navigator.clipboard.writeText(room.code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };

  const linkLabel =
    linkStatus === "live"
      ? copy.live
      : linkStatus === "polling"
        ? copy.polling
        : linkStatus;

  // —— Active multiplayer round UI ——
  if (isActive && currentRound && gameId) {
    const media = currentRound.media ?? { type: "panorama" };
    const meSubmitted =
      guessLocked ||
      (room.current_player_id != null &&
        (guessProgress?.submitted_player_ids ?? []).includes(
          room.current_player_id,
        ));
    const timerLabel = formatRoundTimer(secondsRemaining);
    const roundStats = Array.from(
      { length: room.round_count ?? 5 },
      (_, index) => {
        const n = index + 1;
        return {
          roundNumber: n,
          active: n === currentRound.round_number,
          value:
            n === currentRound.round_number
              ? timerLabel
              : n < currentRound.round_number
                ? "✓"
                : "",
        };
      },
    );

    return (
      <main className="relative h-dvh overflow-hidden bg-[#08051d] font-sans text-white">
        <SceneViewer
          ref={sceneRef}
          media={{
            panorama_id: media.panorama_id,
            url: media.url,
            attribution: media.attribution,
          }}
          googleMapsApiKey={googleMapsApiKey}
          copy={{
            mapUnavailable: copy.mapUnavailable,
            retryMap: copy.retryMap,
          }}
          onHeadingChange={setHeading}
        />

        <RoundHud
          locale={locale}
          backHref={`/${locale}/rooms`}
          copy={{
            backToMission: copy.back,
            mapLabel: copy.mapLabel,
            total: copy.total,
          }}
          heading={heading}
          secondsRemaining={secondsRemaining}
          roundStats={roundStats}
          totalScore={
            players.find((p) => p.id === room.current_player_id)
              ?.total_score ?? 0
          }
        />

        <div className="absolute top-16 left-3 z-30 rounded-full bg-black/55 px-3 py-1 text-[10px] font-bold tracking-wide text-white/80 uppercase">
          {room.code} · {linkLabel}
          {guessProgress
            ? ` · ${guessProgress.submitted_count}/${guessProgress.eligible_count}`
            : ""}
        </div>

        <MapControls
          copy={{
            zoomIn: copy.zoomIn,
            zoomOut: copy.zoomOut,
            recenter: copy.recenter,
            openReactions: "",
          }}
          showReactions={false}
          onZoomIn={() => {
            sceneRef.current?.changeZoom(1);
            guessMapRef.current?.changeZoom(1);
          }}
          onZoomOut={() => {
            sceneRef.current?.changeZoom(-1);
            guessMapRef.current?.changeZoom(-1);
          }}
          onRecenter={() => {
            sceneRef.current?.recenter();
            guessMapRef.current?.recenter();
          }}
        />

        <GuessMap
          ref={guessMapRef}
          roundKey={currentRound.id}
          googleMapsApiKey={googleMapsApiKey}
          locale={locale}
          copy={{
            mapLabel: copy.mapLabel,
            placePin: copy.placePin,
            submitGuess: meSubmitted ? copy.waitingOthers : copy.submitGuess,
            submitting: copy.submitting,
            next: copy.next,
            expandMap: copy.expandMap,
            correctLocation: copy.correctLocation,
            timeExpired: copy.timeExpired,
            distance: copy.distance,
          }}
          pin={pin}
          onPinChange={setPin}
          pinLocked={meSubmitted || Boolean(reveal)}
          reveal={reveal}
          expanded={mapExpanded}
          onExpandedChange={setMapExpanded}
          isPending={isPending}
          error={error}
          onSubmit={submitGuess}
          onAdvance={() => {
            setReveal(null);
            setPin(null);
            setGuessLocked(false);
            void pollRoom();
          }}
        />

        {meSubmitted && !reveal ? (
          <p className="absolute bottom-40 left-1/2 z-30 -translate-x-1/2 rounded-full bg-black/70 px-4 py-2 text-xs font-bold text-white">
            {copy.waitingOthers}
          </p>
        ) : null}
      </main>
    );
  }

  // —— Lobby UI ——
  return (
    <main className="min-h-dvh bg-[#08051d] px-4 py-8 text-white sm:px-8">
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
        <Link
          href={`/${locale}/rooms` as Route}
          className="text-sm font-semibold text-violet-200 underline-offset-4 hover:underline"
        >
          {copy.back}
        </Link>

        <header className="space-y-2">
          <h1 className="text-3xl font-black tracking-tight">{copy.title}</h1>
          <p className="text-sm text-white/70">
            {status === "active" ? copy.activeHint : copy.waiting}
          </p>
          <p className="text-[10px] font-bold tracking-wide text-white/50 uppercase">
            {linkLabel}
          </p>
        </header>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-5">
          <p className="text-xs font-black tracking-wide text-white/60 uppercase">
            {copy.codeLabel}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <p className="text-3xl font-black tracking-[0.3em] text-[#ffbd59]">
              {room.code}
            </p>
            <button
              type="button"
              onClick={copyCode}
              className="rounded-full border border-white/20 px-4 py-1.5 text-xs font-bold uppercase"
            >
              {copied ? copy.copied : copy.copyCode}
            </button>
          </div>
          <p className="mt-3 text-sm text-white/70">
            {copy.statusLabel}:{" "}
            <strong className="text-white">{status}</strong>
          </p>
          {status === "lobby" && playerCount < 2 ? (
            <p className="mt-2 text-sm font-semibold text-amber-200">
              {copy.needPlayers}
            </p>
          ) : null}
        </section>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-5">
          <h2 className="mb-3 text-xs font-black tracking-wide text-white/60 uppercase">
            {copy.playersLabel} ({playerCount}
            {room.max_players ? `/${room.max_players}` : ""})
          </h2>
          <ul className="space-y-2">
            {players.map((player) => (
              <li
                key={player.id}
                className="flex items-center justify-between rounded-xl bg-black/20 px-3 py-2 text-sm"
              >
                <span className="font-semibold">{player.display_name}</span>
                <span className="text-xs font-bold text-violet-200 uppercase">
                  {player.role === "host" ? copy.host : player.role}
                  {room.current_player_id === player.id
                    ? ` · ${copy.you}`
                    : ""}
                </span>
              </li>
            ))}
          </ul>
        </section>

        {error ? (
          <p role="alert" className="text-sm font-semibold text-red-300">
            {error}
          </p>
        ) : null}

        <div className="flex flex-wrap gap-3">
          {isHost && status === "lobby" ? (
            <>
              <button
                type="button"
                onClick={start}
                disabled={isPending || !canStartByRules}
                className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-3 text-xs font-black tracking-wide uppercase disabled:cursor-not-allowed disabled:opacity-45"
              >
                {isPending ? copy.starting : copy.start}
              </button>
              <button
                type="button"
                onClick={cancel}
                disabled={isPending}
                className="rounded-full border border-red-400/40 bg-red-950/40 px-6 py-3 text-xs font-black tracking-wide uppercase disabled:opacity-45"
              >
                {copy.cancel}
              </button>
            </>
          ) : null}

          {!isHost && status === "lobby" ? (
            <button
              type="button"
              onClick={leave}
              disabled={isPending}
              className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-3 text-xs font-black tracking-wide uppercase disabled:opacity-45"
            >
              {copy.leave}
            </button>
          ) : null}
        </div>
      </div>
    </main>
  );
}
