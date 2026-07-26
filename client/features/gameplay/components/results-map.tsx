"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { MapPin, RotateCcw } from "lucide-react";
import {
  loadGoogleMaps,
  type GoogleLatLngBounds,
  type GoogleMap,
  type GoogleMarker,
} from "@/features/gameplay/google-maps";
import type { AppLocale } from "@/lib/i18n/routing";
import { cn } from "@/lib/utils";

/** Neutral per-round result payload for map markers and polylines. */
export type ResultRound = {
  roundNumber: number;
  guess: { latitude: number; longitude: number; timed_out?: boolean } | null;
  actual: { latitude: number; longitude: number };
  /** Optional score for the active-step readout (solo / daily results chrome). */
  score?: number;
  /** Optional distance for the active-step readout. */
  distanceMeters?: number;
};

export type ResultsMapSelection = number | "overview";

export type ResultsMapProps = Readonly<{
  rounds: readonly ResultRound[];
  locale: AppLocale;
  googleMapsApiKey: string;
  totalScore?: number;
  /** Controlled selection: round number or overview. Omit for internal autoplay. */
  selectedRound?: ResultsMapSelection;
  onSelectRound?: (selection: ResultsMapSelection) => void;
  roundLabel: string;
  scoreLabel: string;
  distanceLabel: string;
  overviewLabel: string;
  breakdownLabel: string;
  nextLabel: string;
  finalScoreLabel: string;
  pointsOfLabel: string;
  yourGuessLabel: string;
  correctPositionLabel: string;
  unavailableLabel: string;
  ariaLabel: string;
  className?: string;
  showZoomControls?: boolean;
}>;

type MapLine = { setMap(map: GoogleMap | null): void };
type RoundVisual = {
  markers: GoogleMarker[];
  line: MapLine | null;
  bounds: GoogleLatLngBounds;
};

const RESULT_BREAKDOWN_STEP_MS = 2_400;

function nextStepIndex(current: number, roundCount: number) {
  return Math.min(Math.max(0, current) + 1, Math.max(0, roundCount));
}

function createMarkerContent(
  label: string,
  variant: "guess" | "answer",
): HTMLDivElement {
  const element = document.createElement("div");
  element.textContent = label;
  element.className =
    variant === "answer"
      ? "grid size-10 place-items-center rounded-full border-[3px] border-white bg-[#f19a3e] text-xs font-black text-white shadow-[0_4px_22px_rgba(241,154,62,.7)] animate-[pulse_1.8s_ease-in-out_infinite] motion-reduce:animate-none"
      : "grid size-9 place-items-center rounded-full border-[3px] border-white bg-[#6d38dc] text-xs font-black text-white shadow-[0_4px_22px_rgba(109,56,220,.7)] animate-[pulse_1.8s_ease-in-out_infinite] motion-reduce:animate-none";
  return element;
}

function formatDistance(locale: AppLocale, distanceMeters: number) {
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(distanceMeters / 1_000)} km`;
}

function visibleGuess(guess: ResultRound["guess"]) {
  if (!guess || guess.timed_out) return null;
  return guess;
}

function selectionToIndex(
  selection: ResultsMapSelection,
  ordered: readonly ResultRound[],
): number {
  if (selection === "overview") return ordered.length;
  const index = ordered.findIndex((round) => round.roundNumber === selection);
  return index >= 0 ? index : ordered.length;
}

function indexToSelection(
  index: number,
  ordered: readonly ResultRound[],
): ResultsMapSelection {
  if (index >= ordered.length) return "overview";
  return ordered[index]?.roundNumber ?? "overview";
}

/**
 * Generic results map: guess vs actual markers, polylines, and step breakdown
 * chrome shared by Daily and Solo result surfaces.
 */
export function ResultsMap({
  rounds: unsortedRounds,
  locale,
  googleMapsApiKey,
  totalScore = 0,
  selectedRound,
  onSelectRound,
  roundLabel,
  scoreLabel,
  distanceLabel,
  overviewLabel,
  breakdownLabel,
  nextLabel,
  finalScoreLabel,
  pointsOfLabel,
  yourGuessLabel,
  correctPositionLabel,
  unavailableLabel,
  ariaLabel,
  className,
  showZoomControls = false,
}: ResultsMapProps) {
  const mapElementRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<GoogleMap | null>(null);
  const visualsRef = useRef<RoundVisual[]>([]);
  const overviewBoundsRef = useRef<GoogleLatLngBounds | null>(null);
  const [mapUnavailable, setMapUnavailable] = useState(!googleMapsApiKey);
  const [mapReady, setMapReady] = useState(false);
  const [internalIndex, setInternalIndex] = useState(0);
  const [autoPlay, setAutoPlay] = useState(selectedRound === undefined);

  const ordered = useMemo(
    () =>
      [...unsortedRounds].sort((left, right) => left.roundNumber - right.roundNumber),
    [unsortedRounds],
  );

  const isControlled = selectedRound !== undefined;
  const activeIndex = isControlled
    ? selectionToIndex(selectedRound, ordered)
    : internalIndex;
  const showingOverview = activeIndex >= ordered.length;
  const activeRound = showingOverview ? null : (ordered[activeIndex] ?? null);
  const totalDistance = ordered.reduce(
    (sum, round) => sum + (round.distanceMeters ?? 0),
    0,
  );

  const setActiveIndex = (index: number) => {
    if (isControlled) {
      onSelectRound?.(indexToSelection(index, ordered));
      return;
    }
    setInternalIndex(index);
  };

  useEffect(() => {
    if (!googleMapsApiKey || !mapElementRef.current) return;
    let active = true;
    const markers: GoogleMarker[] = [];
    const lines: MapLine[] = [];

    loadGoogleMaps(googleMapsApiKey)
      .then((maps) => {
        if (!active || !mapElementRef.current) return;
        const map = new maps.Map(mapElementRef.current, {
          center: { lat: 18, lng: 8 },
          zoom: 2,
          disableDefaultUI: true,
          clickableIcons: false,
          gestureHandling: "cooperative",
          keyboardShortcuts: false,
          zoomControl: showZoomControls,
          mapId: "DEMO_MAP_ID",
        });
        const overviewBounds = new maps.LatLngBounds();
        const visuals = ordered.map((round) => {
          const roundBounds = new maps.LatLngBounds();
          const answerPosition = {
            lat: round.actual.latitude,
            lng: round.actual.longitude,
          };
          roundBounds.extend(answerPosition);
          overviewBounds.extend(answerPosition);
          const answerMarker = new maps.marker.AdvancedMarkerElement({
            map,
            position: answerPosition,
            title: `${correctPositionLabel} · ${roundLabel} ${round.roundNumber}`,
            content: createMarkerContent(String(round.roundNumber), "answer"),
          });
          answerMarker.map = null;
          markers.push(answerMarker);
          const roundMarkers = [answerMarker];
          let line: MapLine | null = null;

          const guess = visibleGuess(round.guess);
          if (guess) {
            const guessPosition = {
              lat: guess.latitude,
              lng: guess.longitude,
            };
            roundBounds.extend(guessPosition);
            overviewBounds.extend(guessPosition);
            const guessMarker = new maps.marker.AdvancedMarkerElement({
              map,
              position: guessPosition,
              title: `${yourGuessLabel} · ${roundLabel} ${round.roundNumber}`,
              content: createMarkerContent(String(round.roundNumber), "guess"),
            });
            guessMarker.map = null;
            markers.push(guessMarker);
            roundMarkers.push(guessMarker);
            line = new maps.Polyline({
              map,
              path: [guessPosition, answerPosition],
              geodesic: true,
              strokeColor: "#f3a62b",
              strokeOpacity: 0.95,
              strokeWeight: 3,
            });
            line.setMap(null);
            lines.push(line);
          }

          return { markers: roundMarkers, line, bounds: roundBounds };
        });

        mapRef.current = map;
        visualsRef.current = visuals;
        overviewBoundsRef.current = overviewBounds;
        map.fitBounds(overviewBounds);
        setMapUnavailable(false);
        setMapReady(true);
      })
      .catch(() => {
        if (active) setMapUnavailable(true);
      });

    return () => {
      active = false;
      markers.forEach((marker) => {
        marker.map = null;
      });
      lines.forEach((line) => line.setMap(null));
      visualsRef.current = [];
      overviewBoundsRef.current = null;
      mapRef.current = null;
    };
  }, [
    correctPositionLabel,
    googleMapsApiKey,
    ordered,
    roundLabel,
    showZoomControls,
    yourGuessLabel,
  ]);

  useEffect(() => {
    const map = mapRef.current;
    if (!mapReady || !map) return;
    const overview = activeIndex >= visualsRef.current.length;

    visualsRef.current.forEach((visual, index) => {
      const visible = overview || index === activeIndex;
      visual.markers.forEach((marker) => {
        marker.map = visible ? map : null;
      });
      visual.line?.setMap(visible ? map : null);
    });

    const bounds = overview
      ? overviewBoundsRef.current
      : visualsRef.current[activeIndex]?.bounds;
    if (bounds) map.fitBounds(bounds);
  }, [activeIndex, mapReady]);

  useEffect(() => {
    if (
      isControlled ||
      !mapReady ||
      !autoPlay ||
      showingOverview ||
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    )
      return;
    const timer = window.setTimeout(() => {
      setInternalIndex((current) => nextStepIndex(current, ordered.length));
    }, RESULT_BREAKDOWN_STEP_MS);
    return () => window.clearTimeout(timer);
  }, [
    autoPlay,
    isControlled,
    mapReady,
    ordered.length,
    showingOverview,
  ]);

  const selectStep = (index: number) => {
    setAutoPlay(false);
    setActiveIndex(index);
  };
  const advance = () => {
    setAutoPlay(false);
    const next =
      activeIndex >= ordered.length
        ? 0
        : nextStepIndex(activeIndex, ordered.length);
    setActiveIndex(next);
  };
  const displayedScore = activeRound?.score ?? totalScore;
  const displayedDistance = activeRound?.distanceMeters ?? totalDistance;

  return (
    <div
      role="region"
      aria-label={ariaLabel}
      className={cn(
        "relative overflow-hidden bg-[radial-gradient(circle_at_50%_35%,#243a65_0%,#141536_45%,#090720_100%)]",
        className,
      )}
    >
      <div ref={mapElementRef} className="absolute inset-0" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,rgba(8,5,31,.5),transparent_32%,transparent_62%,rgba(8,5,31,.82))]" />

      <div className="pointer-events-none absolute start-3 top-3 z-10 flex flex-col gap-2 rounded-xl border border-white/15 bg-[#100b2e]/90 p-3 text-[.65rem] font-bold shadow-xl backdrop-blur-md">
        <span className="flex items-center gap-2">
          <i className="size-3 rounded-full border-2 border-white bg-[#6d38dc]" />
          {yourGuessLabel}
        </span>
        <span className="flex items-center gap-2">
          <i className="size-3 rounded-full border-2 border-white bg-[#f19a3e]" />
          {correctPositionLabel}
        </span>
      </div>

      <div className="pointer-events-none absolute top-3 right-3 z-10 rounded-2xl border border-violet-300/25 bg-[#100b2e]/90 px-5 py-3 text-end shadow-2xl backdrop-blur-md sm:right-1/2 sm:translate-x-1/2 sm:text-center">
        <p className="text-[.6rem] font-black tracking-[.16em] text-violet-200 uppercase">
          {finalScoreLabel}
        </p>
        <p className="bg-[linear-gradient(110deg,#ffd34c,#ff875c_50%,#df53ff)] bg-clip-text text-4xl leading-none font-black text-transparent sm:text-5xl">
          {totalScore.toLocaleString(locale)}
        </p>
        <p className="mt-1 text-[.6rem] font-bold text-white/70">
          {pointsOfLabel}
        </p>
      </div>

      <div className="pointer-events-none absolute inset-x-3 bottom-3 z-10 flex justify-center">
        <div className="pointer-events-auto w-full max-w-3xl rounded-2xl border border-violet-300/25 bg-[#100b2e]/94 p-3 shadow-2xl backdrop-blur-xl sm:p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div aria-live="polite">
              <p className="text-[.62rem] font-black tracking-[.14em] text-violet-300 uppercase">
                {activeRound
                  ? `${roundLabel} ${activeRound.roundNumber}`
                  : overviewLabel}
              </p>
              <div className="mt-1 flex items-baseline gap-4">
                <p className="text-xl font-black text-white">
                  {displayedScore.toLocaleString(locale)}
                  <span className="ms-1 text-[.6rem] text-white/55 uppercase">
                    {scoreLabel}
                  </span>
                </p>
                <p className="text-sm font-bold text-white/75">
                  {formatDistance(locale, displayedDistance)}
                  <span className="ms-1 text-[.58rem] text-white/45 uppercase">
                    {distanceLabel}
                  </span>
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={advance}
              className="inline-flex items-center gap-2 rounded-full bg-[linear-gradient(180deg,#a84cff,#6729d1)] px-6 py-2.5 text-xs font-black uppercase shadow-[0_0_20px_rgba(151,67,255,.4)] transition hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            >
              {showingOverview ? (
                <RotateCcw className="size-3.5" aria-hidden="true" />
              ) : (
                <MapPin className="size-3.5" aria-hidden="true" />
              )}
              {showingOverview ? breakdownLabel : nextLabel}
            </button>
          </div>

          <div
            className="mt-3 grid grid-cols-6 gap-1.5"
            aria-label={breakdownLabel}
          >
            {ordered.map((round, index) => (
              <button
                key={round.roundNumber}
                type="button"
                aria-label={`${roundLabel} ${round.roundNumber}`}
                aria-current={activeIndex === index ? "step" : undefined}
                onClick={() => selectStep(index)}
                className={`rounded-lg border px-1 py-2 text-[.65rem] font-black transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white ${
                  activeIndex === index
                    ? "border-violet-200 bg-violet-500 text-white"
                    : "border-white/10 bg-white/5 text-white/60 hover:bg-white/10 hover:text-white"
                }`}
              >
                {round.roundNumber}
              </button>
            ))}
            <button
              type="button"
              aria-label={overviewLabel}
              aria-current={showingOverview ? "step" : undefined}
              onClick={() => selectStep(ordered.length)}
              className={`rounded-lg border px-1 py-2 text-[.6rem] font-black uppercase transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white ${
                showingOverview
                  ? "border-violet-200 bg-violet-500 text-white"
                  : "border-white/10 bg-white/5 text-white/60 hover:bg-white/10 hover:text-white"
              }`}
            >
              {overviewLabel}
            </button>
          </div>
        </div>
      </div>

      {mapUnavailable ? (
        <div className="absolute inset-0 z-[5] grid place-items-center bg-[#100b2e]/85 p-5 text-center">
          <p className="rounded-full border border-white/10 bg-[#100b2e]/90 px-5 py-3 text-sm font-bold text-white/70 backdrop-blur">
            {unavailableLabel}
          </p>
        </div>
      ) : null}
    </div>
  );
}
