"use client";

import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
} from "react";
import { MapPin, Maximize2 } from "lucide-react";
import {
  loadGoogleMaps,
  type GoogleMap,
  type GoogleMarker,
} from "@/features/gameplay/google-maps";

export type GuessPin = { latitude: number; longitude: number };

export type GuessReveal = Readonly<{
  guess: {
    latitude: number;
    longitude: number;
    distance_meters: number;
    score: number;
    timed_out?: boolean;
  };
  actual_location: {
    latitude: number;
    longitude: number;
  };
  max_score: number;
  score_percent: number;
}>;

export type GuessMapCopy = Readonly<{
  mapLabel: string;
  placePin: string;
  submitGuess: string;
  submitting: string;
  next: string;
  expandMap: string;
  correctLocation: string;
  timeExpired: string;
  distance: string;
}>;

export type GuessMapHandle = {
  changeZoom: (delta: number) => void;
  recenter: () => void;
  clearRevealArtifacts: () => void;
};

type GuessMapProps = Readonly<{
  /** Resets the map instance when this key changes (e.g. round id). */
  roundKey: string;
  googleMapsApiKey: string;
  locale: string;
  copy: GuessMapCopy;
  pin: GuessPin | null;
  onPinChange: (pin: GuessPin) => void;
  pinLocked?: boolean;
  reveal: GuessReveal | null;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
  isPending?: boolean;
  error?: string | null;
  onSubmit: () => void;
  onAdvance: () => void;
}>;

function formatDistance(distanceMeters: number) {
  return distanceMeters < 1_000
    ? `${distanceMeters} m`
    : `${(distanceMeters / 1_000).toFixed(1)} km`;
}

export const GuessMap = forwardRef<GuessMapHandle, GuessMapProps>(
  function GuessMap(
    {
      roundKey,
      googleMapsApiKey,
      locale,
      copy,
      pin,
      onPinChange,
      pinLocked = false,
      reveal,
      expanded,
      onExpandedChange,
      isPending = false,
      error = null,
      onSubmit,
      onAdvance,
    },
    ref,
  ) {
    const mapRef = useRef<HTMLDivElement>(null);
    const guessMapRef = useRef<GoogleMap | null>(null);
    const guessMarkerRef = useRef<GoogleMarker | null>(null);
    const answerMarkerRef = useRef<GoogleMarker | null>(null);
    const revealLineRef = useRef<{ setMap(map: GoogleMap | null): void } | null>(
      null,
    );
    const pinLockedRef = useRef(pinLocked);
    const onPinChangeRef = useRef(onPinChange);

    pinLockedRef.current = pinLocked;
    onPinChangeRef.current = onPinChange;

    const clearRevealArtifacts = () => {
      if (guessMarkerRef.current) guessMarkerRef.current.map = null;
      if (answerMarkerRef.current) answerMarkerRef.current.map = null;
      revealLineRef.current?.setMap(null);
      guessMarkerRef.current = null;
      answerMarkerRef.current = null;
      revealLineRef.current = null;
    };

    useImperativeHandle(ref, () => ({
      changeZoom(delta: number) {
        const map = guessMapRef.current;
        if (map) {
          map.setZoom(Math.max(0, Math.min(20, (map.getZoom() ?? 1) + delta)));
        }
      },
      recenter() {
        guessMapRef.current?.setCenter({ lat: 18, lng: 0 });
      },
      clearRevealArtifacts,
    }));

    useEffect(() => {
      if (!googleMapsApiKey || !mapRef.current) return;
      let active = true;
      let clickListener: { remove(): void } | null = null;
      let resizeObserver: ResizeObserver | null = null;
      let resizeFrame: number | null = null;

      loadGoogleMaps(googleMapsApiKey)
        .then((maps) => {
          if (!active || !mapRef.current) return;
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
            if (!event.latLng || pinLockedRef.current) return;
            const nextPin = {
              latitude: event.latLng.lat(),
              longitude: event.latLng.lng(),
            };
            onPinChangeRef.current(nextPin);
            const position = { lat: nextPin.latitude, lng: nextPin.longitude };
            if (guessMarkerRef.current)
              guessMarkerRef.current.position = position;
            else
              guessMarkerRef.current = new maps.marker.AdvancedMarkerElement({
                map,
                position,
              });
          });
        })
        .catch(() => undefined);

      return () => {
        active = false;
        clickListener?.remove();
        resizeObserver?.disconnect();
        if (resizeFrame !== null) window.cancelAnimationFrame(resizeFrame);
        clearRevealArtifacts();
        guessMapRef.current = null;
      };
    }, [googleMapsApiKey, roundKey]);

    useEffect(() => {
      if (!reveal || !guessMapRef.current || !googleMapsApiKey) return;
      let active = true;
      loadGoogleMaps(googleMapsApiKey)
        .then((maps) => {
          if (!active || !guessMapRef.current) return;
          const actual = {
            lat: reveal.actual_location.latitude,
            lng: reveal.actual_location.longitude,
          };
          const pinEl = document.createElement("div");
          pinEl.className =
            "grid size-8 place-items-center rounded-full border-4 border-white bg-emerald-500 text-sm font-black text-white shadow-lg";
          pinEl.textContent = "✓";
          answerMarkerRef.current = new maps.marker.AdvancedMarkerElement({
            map: guessMapRef.current,
            position: actual,
            title: copy.correctLocation,
            content: pinEl,
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

    // Keep the visible guess marker in sync if pin is cleared externally.
    useEffect(() => {
      if (pin) return;
      if (guessMarkerRef.current) {
        guessMarkerRef.current.map = null;
        guessMarkerRef.current = null;
      }
    }, [pin]);

    const mapPanelClass = expanded
      ? "absolute right-3 bottom-3 z-40 h-[min(75vh,40rem)] w-[min(86vw,54rem)] sm:right-6 sm:bottom-6"
      : "absolute right-3 bottom-3 z-20 h-[17rem] w-[min(92vw,21rem)] sm:right-6 sm:bottom-6";

    return (
      <section
        className={`${mapPanelClass} flex flex-col overflow-hidden rounded-xl border border-white/20 bg-[#101326]/95 shadow-2xl transition-[width,height] duration-300 ease-out motion-reduce:transition-none`}
        aria-label={copy.mapLabel}
        onMouseEnter={() => onExpandedChange(true)}
        onMouseLeave={() => {
          if (!reveal) onExpandedChange(false);
        }}
        onFocusCapture={() => onExpandedChange(true)}
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
                onClick={onAdvance}
                className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-2.5 text-xs font-black uppercase"
              >
                {copy.next}
              </button>
            </div>
          ) : !expanded ? (
            <button
              type="button"
              onClick={() => onExpandedChange(true)}
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
                onClick={onSubmit}
                disabled={isPending || !pin}
                className="w-full rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] py-2.5 text-xs font-black tracking-wider uppercase shadow-md shadow-violet-950/40 disabled:cursor-not-allowed disabled:opacity-45"
              >
                {isPending ? copy.submitting : copy.submitGuess}
              </button>
            </>
          )}
        </div>
        {!expanded && !reveal ? (
          <button
            type="button"
            aria-label={copy.expandMap}
            onClick={() => onExpandedChange(true)}
            className="absolute top-2 right-2 grid size-9 place-items-center rounded-full bg-[#111827]/85 shadow focus-visible:outline focus-visible:outline-2 focus-visible:outline-white"
          >
            <Maximize2 className="size-4" />
          </button>
        ) : null}
      </section>
    );
  },
);
