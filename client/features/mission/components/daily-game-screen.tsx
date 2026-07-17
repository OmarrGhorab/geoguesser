"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, useTransition } from "react";
import { Crosshair, Flag, MapPin, Maximize2, Minus, Plus } from "lucide-react";
import type { AppLocale } from "@/lib/i18n/routing";
import {
  expireDailyRoundAction,
  submitDailyGuessAction,
} from "@/features/mission/actions";
import { dailyMissionResultsHref } from "@/features/mission/routes";
import type {
  DailyGame,
  DailyGuessResult,
  DailyRound,
} from "@/features/mission/schemas";
import type { DailyGameCopy } from "@/features/mission/types";
import {
  COMPASS_TICK_WIDTH_PX,
  compassTickLabel,
  compassTrackOffset,
  normalizedHeading,
} from "@/features/mission/compass";
import {
  formatRoundTimer,
  roundStatValue,
} from "@/features/mission/round-stats";
import {
  loadGoogleMaps,
  type GoogleMap,
  type GoogleMarker,
  type GoogleMapsNamespace,
} from "@/features/mission/google-maps";

type DailyGameScreenProps = Readonly<{
  locale: AppLocale;
  game: DailyGame;
  initialRound: DailyRound | null;
  googleMapsApiKey: string;
  copy: DailyGameCopy;
  retryAction: () => Promise<void>;
}>;
type Pin = { latitude: number; longitude: number };
type PendingOutcome =
  { kind: "next_round"; nextRound: DailyRound } | { kind: "completed" };

const REACTIONS = [
  {
    emoji: "🪝",
    label: "Hooked",
    position: "top-[7%] left-[50%] -translate-x-1/2",
  },
  { emoji: "🤯", label: "Mind blown", position: "top-[25%] right-[16%]" },
  { emoji: "😮", label: "Surprised", position: "bottom-[21%] right-[16%]" },
  {
    emoji: "😍",
    label: "Love it",
    position: "bottom-[5%] left-[50%] -translate-x-1/2",
  },
  { emoji: "🔥", label: "On fire", position: "bottom-[21%] left-[16%]" },
  { emoji: "5K", label: "Perfect score", position: "top-[25%] left-[16%]" },
] as const;

function formatDistance(distanceMeters: number) {
  return distanceMeters < 1_000
    ? `${distanceMeters} m`
    : `${(distanceMeters / 1_000).toFixed(1)} km`;
}

function remainingRoundSeconds(round: DailyRound | null, fallback: number) {
  if (!round?.ends_at) return fallback;
  return Math.max(
    0,
    Math.ceil((new Date(round.ends_at).getTime() - Date.now()) / 1_000),
  );
}

const COMPASS_TICKS = Array.from({ length: 65 }, (_, index) => index);

export function DailyGameScreen({
  locale,
  game,
  initialRound,
  googleMapsApiKey,
  copy,
  retryAction,
}: DailyGameScreenProps) {
  const router = useRouter();
  const panoramaRef = useRef<HTMLDivElement>(null);
  const panoramaInstanceRef = useRef<InstanceType<
    GoogleMapsNamespace["StreetViewPanorama"]
  > | null>(null);
  const mapRef = useRef<HTMLDivElement>(null);
  const guessMapRef = useRef<GoogleMap | null>(null);
  const guessMarkerRef = useRef<GoogleMarker | null>(null);
  const answerMarkerRef = useRef<GoogleMarker | null>(null);
  const revealLineRef = useRef<{ setMap(map: GoogleMap | null): void } | null>(
    null,
  );
  const expiredRoundRef = useRef<string | null>(null);
  const [round, setRound] = useState(initialRound);
  const [pin, setPin] = useState<Pin | null>(null);
  const [reveal, setReveal] = useState<DailyGuessResult | null>(null);
  const [outcome, setOutcome] = useState<PendingOutcome | null>(null);
  const [totalScore, setTotalScore] = useState(game.total_score);
  const [mapError, setMapError] = useState(!googleMapsApiKey);
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const [reactionOpen, setReactionOpen] = useState(false);
  const [reaction, setReaction] = useState<string | null>(null);
  const [mapExpanded, setMapExpanded] = useState(false);
  const [heading, setHeading] = useState(0);
  const [roundTimeLabels, setRoundTimeLabels] = useState<
    Record<number, string>
  >({});
  const roundTimerSeconds = game.timer_seconds ?? 180;
  const [secondsRemaining, setSecondsRemaining] = useState(() =>
    remainingRoundSeconds(initialRound, roundTimerSeconds),
  );
  const roundClosedRef = useRef(false);

  useEffect(() => {
    if (!round || !googleMapsApiKey || !mapRef.current) return;
    let active = true;
    let clickListener: { remove(): void } | null = null;
    let povListener: { remove(): void } | null = null;
    let resizeObserver: ResizeObserver | null = null;
    let resizeFrame: number | null = null;
    loadGoogleMaps(googleMapsApiKey)
      .then((maps) => {
        if (!active || !mapRef.current) return;
        setMapError(false);
        const map = new maps.Map(mapRef.current, {
          center: { lat: 18, lng: 0 },
          zoom: 1,
          disableDefaultUI: true,
          clickableIcons: false,
          gestureHandling: "greedy",
          mapId: "DEMO_MAP_ID",
          keyboardShortcuts: true,
          zoomControl: false,
        });
        guessMapRef.current = map;
        resizeObserver = new ResizeObserver(() => {
          if (resizeFrame !== null) window.cancelAnimationFrame(resizeFrame);
          resizeFrame = window.requestAnimationFrame(() => {
            maps.event.trigger(map, "resize");
          });
        });
        resizeObserver.observe(mapRef.current);
        clickListener = map.addListener("click", (event) => {
          if (!event.latLng || roundClosedRef.current) return;
          const nextPin = {
            latitude: event.latLng.lat(),
            longitude: event.latLng.lng(),
          };
          setPin(nextPin);
          const position = { lat: nextPin.latitude, lng: nextPin.longitude };
          if (guessMarkerRef.current)
            guessMarkerRef.current.position = position;
          else
            guessMarkerRef.current = new maps.marker.AdvancedMarkerElement({
              map,
              position,
            });
        });
        if (round.media.panorama_id && panoramaRef.current) {
          panoramaInstanceRef.current = new maps.StreetViewPanorama(
            panoramaRef.current,
            {
              pano: round.media.panorama_id,
              pov: { heading: 0, pitch: 0 },
              zoom: 0,
              addressControl: false,
              fullscreenControl: false,
              motionTracking: false,
              showRoadLabels: false,
            },
          );
          const panorama = panoramaInstanceRef.current;
          setHeading(normalizedHeading(panorama.getPov().heading));
          povListener = panorama.addListener("pov_changed", () => {
            setHeading(normalizedHeading(panorama.getPov().heading));
          });
        }
      })
      .catch(() => {
        if (active) setMapError(true);
      });
    return () => {
      active = false;
      clickListener?.remove();
      povListener?.remove();
      resizeObserver?.disconnect();
      if (resizeFrame !== null) window.cancelAnimationFrame(resizeFrame);
      if (guessMarkerRef.current) guessMarkerRef.current.map = null;
      if (answerMarkerRef.current) answerMarkerRef.current.map = null;
      revealLineRef.current?.setMap(null);
      guessMarkerRef.current = null;
      answerMarkerRef.current = null;
      revealLineRef.current = null;
      guessMapRef.current = null;
      panoramaInstanceRef.current = null;
    };
  }, [googleMapsApiKey, round]);

  useEffect(() => {
    if (!round || reveal) return;
    const timer = window.setInterval(
      () => setSecondsRemaining((remaining) => Math.max(0, remaining - 1)),
      1_000,
    );
    return () => window.clearInterval(timer);
  }, [reveal, round]);

  useEffect(() => {
    if (
      !round ||
      secondsRemaining !== 0 ||
      reveal ||
      isPending ||
      expiredRoundRef.current === round.id
    )
      return;
    expiredRoundRef.current = round.id;
    startTransition(async () => {
      const response = await expireDailyRoundAction({
        gameId: game.id,
        roundId: round.id,
      });
      if (!response.ok) {
        setError(copy.errors.unavailable);
        return;
      }
      setRoundTimeLabels((labels) => ({
        ...labels,
        [round.round_number]: "0:00",
      }));
      roundClosedRef.current = true;
      setReveal(response.reveal);
      setOutcome(
        response.kind === "completed"
          ? { kind: "completed" }
          : { kind: "next_round", nextRound: response.nextRound },
      );
    });
  }, [
    copy.errors.unavailable,
    game.id,
    isPending,
    reveal,
    round,
    secondsRemaining,
  ]);

  useEffect(() => {
    if (!reveal || !guessMapRef.current) return;
    let active = true;
    loadGoogleMaps(googleMapsApiKey)
      .then((maps) => {
        if (!active || !guessMapRef.current) return;
        const actual = {
          lat: reveal.actual_location.latitude,
          lng: reveal.actual_location.longitude,
        };
        const pin = document.createElement("div");
        pin.className =
          "grid size-8 place-items-center rounded-full border-4 border-white bg-emerald-500 text-sm font-black text-white shadow-lg";
        pin.textContent = "✓";
        answerMarkerRef.current = new maps.marker.AdvancedMarkerElement({
          map: guessMapRef.current,
          position: actual,
          title: copy.correctLocation,
          content: pin,
        });
        const bounds = new maps.LatLngBounds();
        bounds.extend(actual);
        if (!reveal.guess.timed_out) {
          const guessed = {
            lat: reveal.guess.latitude,
            lng: reveal.guess.longitude,
          };
          bounds.extend(guessed);
          revealLineRef.current = new maps.Polyline({
            map: guessMapRef.current,
            path: [guessed, actual],
            geodesic: true,
            strokeColor: "#7c3aed",
            strokeOpacity: 0.9,
            strokeWeight: 3,
          });
        }
        guessMapRef.current.fitBounds(bounds);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [copy.correctLocation, googleMapsApiKey, reveal]);

  if (!round)
    return (
      <main className="grid min-h-dvh place-items-center bg-[#08051d] text-white">
        <p>{copy.errors.unavailable}</p>
      </main>
    );

  const submit = () => {
    if (secondsRemaining === 0) {
      setError(copy.timeExpired);
      return;
    }
    if (!pin) {
      setError(copy.selectLocation);
      return;
    }
    const submittedTimerLabel = formatRoundTimer(secondsRemaining);
    setError(null);
    startTransition(async () => {
      const response = await submitDailyGuessAction({
        gameId: game.id,
        roundId: round.id,
        latitude: pin.latitude,
        longitude: pin.longitude,
        idempotencyKey: crypto.randomUUID(),
      });
      if (!response.ok) {
        setError(
          response.code === "invalid_guess"
            ? copy.errors.invalidGuess
            : copy.errors.unavailable,
        );
        return;
      }
      setRoundTimeLabels((labels) => ({
        ...labels,
        [round.round_number]: submittedTimerLabel,
      }));
      roundClosedRef.current = true;
      setReveal(response.reveal);
      setTotalScore((score) => score + response.reveal.guess.score);
      setOutcome(
        response.kind === "completed"
          ? { kind: "completed" }
          : { kind: "next_round", nextRound: response.nextRound },
      );
    });
  };
  const advance = () => {
    if (!outcome) return;
    if (outcome.kind === "completed") {
      router.replace(dailyMissionResultsHref(locale, game.id));
      return;
    }
    if (guessMarkerRef.current) guessMarkerRef.current.map = null;
    if (answerMarkerRef.current) answerMarkerRef.current.map = null;
    revealLineRef.current?.setMap(null);
    guessMarkerRef.current = null;
    answerMarkerRef.current = null;
    revealLineRef.current = null;
    roundClosedRef.current = false;
    expiredRoundRef.current = null;
    setSecondsRemaining(
      remainingRoundSeconds(outcome.nextRound, roundTimerSeconds),
    );
    setRound(outcome.nextRound);
    setPin(null);
    setReveal(null);
    setOutcome(null);
    setError(null);
  };
  const hasPlayableMedia = Boolean(round.media.panorama_id || round.media.url);
  const mapPanelClass = mapExpanded
    ? "absolute right-3 bottom-3 z-40 h-[min(75vh,40rem)] w-[min(86vw,54rem)] sm:right-6 sm:bottom-6"
    : "absolute right-3 bottom-3 z-20 h-[17rem] w-[min(92vw,21rem)] sm:right-6 sm:bottom-6";
  const compassOffset = compassTrackOffset(heading);
  const timerLabel = formatRoundTimer(secondsRemaining);
  const roundStats = Array.from({ length: game.round_count }, (_, index) => {
    const roundNumber = index + 1;
    return {
      roundNumber,
      active: roundNumber === round.round_number,
      value: roundStatValue(
        roundNumber,
        round.round_number,
        timerLabel,
        roundTimeLabels,
      ),
    };
  });
  const changePanoramaZoom = (delta: number) => {
    const panorama = panoramaInstanceRef.current;
    if (panorama)
      panorama.setZoom(Math.max(0, Math.min(5, panorama.getZoom() + delta)));
  };
  const changeMapZoom = (delta: number) => {
    const map = guessMapRef.current;
    if (map)
      map.setZoom(Math.max(0, Math.min(20, (map.getZoom() ?? 1) + delta)));
  };
  const recenter = () => {
    panoramaInstanceRef.current?.setPov({ heading: 0, pitch: 0 });
    guessMapRef.current?.setCenter({ lat: 18, lng: 0 });
  };

  return (
    <main className="relative h-dvh overflow-hidden bg-[#08051d] font-sans text-white">
      <div
        ref={panoramaRef}
        className="absolute inset-0 bg-cover bg-center"
        style={
          round.media.url
            ? { backgroundImage: `url(${JSON.stringify(round.media.url)})` }
            : undefined
        }
      />
      {!hasPlayableMedia || (mapError && !round.media.url) ? (
        <div className="absolute inset-0 grid place-items-center bg-[radial-gradient(circle_at_center,#25205d,#08051d)]">
          <div className="flex flex-col items-center gap-3 rounded-3xl bg-black/35 px-6 py-5 text-center">
            <p className="font-bold text-white/75">{copy.mapUnavailable}</p>
            {!hasPlayableMedia ? (
              <form action={retryAction}>
                <button
                  type="submit"
                  className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-7 py-2.5 text-sm font-black tracking-wide uppercase"
                >
                  {copy.retryMap}
                </button>
              </form>
            ) : null}
          </div>
        </div>
      ) : null}

      <header className="absolute inset-x-0 top-0 z-30 flex items-start justify-between p-3 sm:p-5">
        <Link
          href={`/${locale}/daily-mission`}
          aria-label={copy.backToMission}
          className="relative block h-10 w-40 drop-shadow-lg"
        >
          <Image
            src="/authentication/worldguesser-logo-transparent.png"
            alt="WorldGuess"
            fill
            sizes="156px"
            className="object-contain object-left"
            priority
          />
        </Link>
        <div className="flex flex-col items-center gap-3">
          <div
            aria-label={copy.mapLabel}
            className="relative hidden h-8 w-60 overflow-hidden rounded-full bg-[#2d3c5c]/80 text-xs font-black tracking-[.3em] shadow-lg backdrop-blur sm:block"
          >
            <div
              className="absolute top-0 left-1/2 flex h-full items-center text-white/75 will-change-transform"
              style={{ transform: `translateX(${compassOffset}px)` }}
            >
              {COMPASS_TICKS.map((index) => {
                const label = compassTickLabel(index);
                return (
                  <span
                    key={index}
                    className="relative flex h-full shrink-0 items-center justify-center"
                    style={{ width: COMPASS_TICK_WIDTH_PX }}
                  >
                    {label ? (
                      <strong className="absolute top-2 text-[10px] tracking-normal text-white">
                        {label}
                      </strong>
                    ) : (
                      <i className="h-3 border-s border-white/45" />
                    )}
                  </span>
                );
              })}
            </div>
            <span
              className="absolute inset-x-0 top-0 z-10 mx-auto h-2 w-px bg-white/80"
              aria-hidden="true"
            />
          </div>
          <div
            className={`rounded-full border-[5px] bg-[#262143]/95 px-7 py-1.5 text-xl font-black italic shadow-xl ${secondsRemaining <= 15 ? "border-[#f23945]" : "border-[#7046d9]"}`}
          >
            {timerLabel}
          </div>
        </div>
        <div className="flex overflow-hidden rounded-2xl border-2 border-[#5e3ba8] bg-[#160d3f]/95 shadow-2xl">
          {roundStats.map((stat) => (
            <RoundStat
              key={stat.roundNumber}
              active={stat.active}
              label={`R${stat.roundNumber}`}
              value={stat.value}
            />
          ))}
          <RoundStat
            label={copy.total}
            value={totalScore.toLocaleString(locale)}
          />
        </div>
      </header>

      <div className="absolute bottom-7 left-4 z-20 flex flex-col gap-2">
        <Control
          label={copy.zoomIn}
          onClick={() => {
            changePanoramaZoom(1);
            changeMapZoom(1);
          }}
        >
          <Plus className="size-5" />
        </Control>
        <Control
          label={copy.zoomOut}
          onClick={() => {
            changePanoramaZoom(-1);
            changeMapZoom(-1);
          }}
        >
          <Minus className="size-5" />
        </Control>
        <Control label={copy.recenter} onClick={recenter}>
          <Crosshair className="size-5" />
        </Control>
      </div>
      <button
        type="button"
        aria-label={copy.openReactions}
        aria-pressed={reactionOpen}
        onClick={() => setReactionOpen(true)}
        className="absolute bottom-7 left-20 z-20 grid size-11 place-items-center rounded-full bg-black/60 text-white shadow-lg backdrop-blur"
      >
        <Flag className="size-5" />
      </button>

      <section
        className={`${mapPanelClass} flex flex-col overflow-hidden rounded-xl border border-white/20 bg-[#101326]/95 shadow-2xl transition-[width,height] duration-300 ease-out motion-reduce:transition-none`}
        aria-label={copy.mapLabel}
        onMouseEnter={() => setMapExpanded(true)}
        onMouseLeave={() => {
          if (!reveal) setMapExpanded(false);
        }}
        onFocusCapture={() => setMapExpanded(true)}
      >
        <div ref={mapRef} className="min-h-0 w-full flex-1 bg-[#282146]" />
        <div className="shrink-0 border-t border-white/10 bg-[#111320] p-3">
          {reveal ? (
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-[10px] font-bold text-emerald-300 uppercase">
                  {copy.correctLocation}
                </p>
                <p className="text-xl font-black text-[#ffbd59]">
                  {reveal.guess.score.toLocaleString(locale)} /{" "}
                  {reveal.max_score.toLocaleString(locale)}
                </p>
                <p className="text-xs text-white/60">
                  {reveal.guess.timed_out
                    ? copy.timeExpired
                    : `${copy.distance}: ${formatDistance(reveal.guess.distance_meters)}`}{" "}
                  · {reveal.score_percent}%
                </p>
              </div>
              <button
                type="button"
                onClick={advance}
                className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-2.5 text-xs font-black uppercase"
              >
                {copy.next}
              </button>
            </div>
          ) : !mapExpanded ? (
            <button
              type="button"
              onClick={() => setMapExpanded(true)}
              className="flex w-full items-center justify-center gap-2 rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] py-2.5 text-xs font-black tracking-wider uppercase shadow-lg focus-visible:outline focus-visible:outline-2 focus-visible:outline-white"
            >
              <MapPin className="size-4" />
              {copy.placePin}
            </button>
          ) : (
            <>
              <div className="mb-2 flex items-center gap-2 text-xs font-black text-white/80 uppercase">
                <MapPin className="size-4 text-violet-300" />
                {copy.placePin}
              </div>
              {error ? (
                <p
                  role="alert"
                  className="mb-2 text-xs font-semibold text-red-300"
                >
                  {error}
                </p>
              ) : null}
              <button
                type="button"
                onClick={submit}
                disabled={isPending || !pin}
                className="w-full rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] py-2.5 text-xs font-black tracking-wider uppercase shadow-md shadow-violet-950/40 disabled:cursor-not-allowed disabled:opacity-45"
              >
                {isPending ? copy.submitting : copy.submitGuess}
              </button>
            </>
          )}
        </div>
        {!mapExpanded && !reveal ? (
          <button
            type="button"
            aria-label={copy.expandMap}
            onClick={() => setMapExpanded(true)}
            className="absolute top-2 right-2 grid size-9 place-items-center rounded-full bg-[#111827]/85 shadow focus-visible:outline focus-visible:outline-2 focus-visible:outline-white"
          >
            <Maximize2 className="size-4" />
          </button>
        ) : null}
      </section>

      {reactionOpen ? (
        <div
          className="absolute inset-0 z-35 grid place-items-center bg-black/30 backdrop-blur-md"
          role="dialog"
          aria-modal="true"
          aria-label={copy.reactionsTitle}
          onClick={() => setReactionOpen(false)}
        >
          <div
            className="relative size-[min(74vw,21rem)]"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="absolute inset-[18%] rounded-full border-8 border-[#6f7180] bg-[conic-gradient(#242632_0deg_58deg,#6e6e75_59deg_61deg,#242632_62deg_118deg,#6e6e75_119deg_121deg,#242632_122deg_178deg,#6e6e75_179deg_181deg,#242632_182deg_238deg,#6e6e75_239deg_241deg,#242632_242deg_298deg,#6e6e75_299deg_301deg,#242632_302deg_360deg)] shadow-2xl" />
            {REACTIONS.map((item, index) => (
              <button
                key={item.label}
                type="button"
                aria-label={item.label}
                onClick={() => {
                  setReaction(item.emoji);
                  setReactionOpen(false);
                }}
                className={`absolute ${item.position} grid size-16 place-items-center rounded-full text-3xl transition hover:scale-110 focus-visible:outline focus-visible:outline-2 focus-visible:outline-white`}
              >
                <span
                  className={
                    index === 5
                      ? "text-2xl font-black text-[#ffad21] italic [text-shadow:2px_2px_0_#9d4200]"
                      : ""
                  }
                >
                  {item.emoji}
                </span>
              </button>
            ))}
            <button
              type="button"
              aria-label={copy.closeReactions}
              onClick={() => setReactionOpen(false)}
              className="absolute inset-[40%] grid place-items-center rounded-full border-2 border-white bg-[#4b4a55] text-lg"
            >
              ×
            </button>
          </div>
          <p className="absolute bottom-[14%] max-w-52 text-center text-sm font-black text-white italic">
            {reaction
              ? `${reaction} ${copy.reactionSent}`
              : copy.reactionsPrompt}
          </p>
        </div>
      ) : null}
    </main>
  );
}

function RoundStat({
  label,
  value,
  active = false,
}: {
  label: string;
  value: string;
  active?: boolean;
}) {
  return (
    <div
      className={`min-w-12 border-r border-white/10 px-2 py-1.5 text-[10px] font-black italic last:border-0 sm:min-w-15 ${active ? "bg-white/10 text-[#ffcc28]" : "text-[#c6a9ff]"}`}
    >
      <span>{label}</span>
      <strong className="block text-sm text-white">{value || "—"}</strong>
    </div>
  );
}
function Control({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="grid size-11 place-items-center rounded-full bg-black/65 text-white shadow-lg backdrop-blur transition hover:bg-black/85 focus-visible:outline focus-visible:outline-2 focus-visible:outline-white"
    >
      {children}
    </button>
  );
}
