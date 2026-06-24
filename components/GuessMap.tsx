"use client";

import { useEffect, useRef, useState } from "react";
import { loadGoogleMaps } from "@/lib/loadGoogleMaps";
import type { GamePhase, LatLng } from "@/lib/types";

const MAP_ID = process.env.NEXT_PUBLIC_GOOGLE_MAPS_MAP_ID ?? "DEMO_MAP_ID";

/**
 * Interactive world map for placing a guess.
 *
 * Responsibilities:
 *   • Load the Google Maps JS API (via the shared singleton loader).
 *   • Let the player click to drop exactly one marker (their guess).
 *   • On reveal (`phase === "revealed"`): show the true location, draw a
 *     line between guess and answer, fit the viewport to both, and lock
 *     further clicks.
 *
 * State ownership note: the map, markers and polyline are plain Google Maps
 * objects held in refs — NOT React state. They mutate outside React's render
 * cycle, which is the correct pattern for imperative map APIs. React state is
 * used only for props that should trigger re-render (`guess`, `phase`).
 */
export type GuessMapProps = {
  /** Current phase; gates click handling and triggers the reveal overlay. */
  phase: GamePhase;
  /** The player's current guess, or null if none placed yet. */
  guess: LatLng | null;
  /** True coordinates of the landmark being guessed. */
  actual: LatLng;
  /** Called when the player clicks the map to (re)place their guess. */
  onPlaceGuess: (point: LatLng) => void;
};

export default function GuessMap({ phase, guess, actual, onPlaceGuess }: GuessMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [mapReady, setMapReady] = useState(false);
  // The live map instance — persists across renders, never triggers re-render.
  const mapRef = useRef<google.maps.Map | null>(null);
  // The user's marker. Reused/updated across clicks to avoid leaks.
  const guessMarkerRef = useRef<google.maps.marker.AdvancedMarkerElement | null>(null);
  // Reveal-only artifacts, torn down on the next round.
  const actualMarkerRef = useRef<google.maps.marker.AdvancedMarkerElement | null>(null);
  const lineRef = useRef<google.maps.Polyline | null>(null);
  // Click listener handle so we can detach it when revealing.
  const clickListenerRef = useRef<google.maps.MapsEventListener | null>(null);
  // Keeps the map redraw in sync with its CSS-driven container size. The
  // wrapper resizes on hover and snaps full-screen on reveal; Google Maps
  // won't repaint that on its own, so we nudge it via the resize event.
  const resizeObserverRef = useRef<ResizeObserver | null>(null);
  // Lets the click handler read the latest callback without re-registering.
  // Updated inside an effect (not during render) per the React hooks rule.
  const onPlaceGuessRef = useRef(onPlaceGuess);
  useEffect(() => {
    onPlaceGuessRef.current = onPlaceGuess;
  }, [onPlaceGuess]);

  // --- Initialize the map once on mount -------------------------------------
  useEffect(() => {
    let cancelled = false;

    loadGoogleMaps()
      .then(() => {
        if (cancelled || !containerRef.current) return;

        const map = new google.maps.Map(containerRef.current, {
          // Start centered on the world so no region is hinted.
          center: { lat: 20, lng: 0 },
          zoom: 2,
          minZoom: 2,
          mapId: MAP_ID,
          // Keep the map chrome minimal: just zoom controls, no map type
          // switcher or streetview pegman (those would leak location info).
          disableDefaultUI: true,
          zoomControl: true,
          // One-finger pan on mobile.
          gestureHandling: "greedy",
          clickableIcons: false,
        });
        mapRef.current = map;
        setMapReady(true);

        // The wrapper resizes via CSS (hover to expand, full-screen on
        // reveal). The Maps JS API won't notice those layout changes on its
        // own, so a ResizeObserver tells it to recompute its viewport —
        // otherwise tiles/markers render stale at the old size.
        resizeObserverRef.current = new ResizeObserver(() => {
          google.maps.event.trigger(map, "resize");
        });
        resizeObserverRef.current.observe(containerRef.current);

      })
      .catch((err) => {
        // Surface a clear message in the container instead of silently failing.
        console.error(err);
        if (containerRef.current) {
          containerRef.current.innerHTML =
            '<div style="padding:1rem;color:#71717a;font-family:system-ui">Failed to load the map. Check your Google Maps API key.</div>';
        }
      });

    return () => {
      cancelled = true;
      // Tear down listeners, observer and instances on unmount.
      resizeObserverRef.current?.disconnect();
      resizeObserverRef.current = null;
      clickListenerRef.current?.remove();
      if (guessMarkerRef.current) guessMarkerRef.current.map = null;
      if (actualMarkerRef.current) actualMarkerRef.current.map = null;
      lineRef.current?.setMap(null);
      mapRef.current = null;
      setMapReady(false);
    };
  }, []);

  // --- Enable map clicks only while a round is actively guessing ------------
  useEffect(() => {
    const map = mapRef.current;
    if (!mapReady || !map) return;

    clickListenerRef.current?.remove();
    clickListenerRef.current = null;

    if (phase !== "guessing") return;

    // Register the click -> guess handler. We read the latest callback from a
    // ref so this listener never goes stale across renders.
    clickListenerRef.current = map.addListener("click", (e: google.maps.MapMouseEvent) => {
      if (!e.latLng) return;
      onPlaceGuessRef.current({ lat: e.latLng.lat(), lng: e.latLng.lng() });
    });

    return () => {
      clickListenerRef.current?.remove();
      clickListenerRef.current = null;
    };
  }, [mapReady, phase]);

  // --- Keep the player's marker in sync with the `guess` prop ---------------
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    if (!guess) {
      if (guessMarkerRef.current) guessMarkerRef.current.map = null;
      guessMarkerRef.current = null;
      return;
    }

    if (guessMarkerRef.current) {
      // Reuse existing marker - cheaper than recreating and avoids flicker.
      guessMarkerRef.current.position = guess;
    } else {
      const pin = new google.maps.marker.PinElement({
        glyph: "?",
        glyphColor: "#ffffff",
        background: "#2563eb",
        borderColor: "#1d4ed8",
      });

      guessMarkerRef.current = new google.maps.marker.AdvancedMarkerElement({
        position: guess,
        map,
        title: "Your guess",
        content: pin.element,
      });
    }
  }, [guess]);

  // --- On reveal: draw the answer, connect, and frame both points ----------
  useEffect(() => {
    const map = mapRef.current;
    if (!map || phase !== "revealed") return;

    // Lock further guesses once the round is revealed.
    clickListenerRef.current?.remove();
    clickListenerRef.current = null;

    // Red pin for the true location.
    const pin = new google.maps.marker.PinElement({
      glyph: "✓",
      glyphColor: "#ffffff",
      background: "#dc2626",
      borderColor: "#991b1b",
    });

    actualMarkerRef.current = new google.maps.marker.AdvancedMarkerElement({
      position: actual,
      map,
      title: "Correct location",
      content: pin.element,
    });

    // Straight-ish line across the map connecting guess → answer.
    lineRef.current = new google.maps.Polyline({
      path: [guess, actual].filter(Boolean) as LatLng[],
      map,
      geodesic: true,
      strokeColor: "#16a34a",
      strokeOpacity: 0.9,
      strokeWeight: 3,
    });

    // Frame both points with a little breathing room.
    const bounds = new google.maps.LatLngBounds();
    if (guess) bounds.extend(guess);
    bounds.extend(actual);
    map.fitBounds(bounds, 60);

    return () => {
      // Clear the reveal overlay when leaving the revealed phase (next round).
      if (actualMarkerRef.current) actualMarkerRef.current.map = null;
      actualMarkerRef.current = null;
      lineRef.current?.setMap(null);
      lineRef.current = null;
    };
  }, [phase, guess, actual]);

  return (
    <div
      className={[
        // Corner-docked, expandable guessing map (GeoGuessr's signature panel).
        // In the guessing phase it's a small box bottom-right that grows on
        // hover/focus; on reveal it snaps to full-screen so both points and
        // the connecting line are visible.
        "absolute z-10 overflow-hidden rounded-lg border-2 border-white/80 bg-zinc-100 shadow-2xl transition-all duration-200 ease-out",
        // Default (guessing): compact corner box.
        "bottom-4 right-4 h-40 w-56",
        // Expand on hover/focus while guessing.
        "hover:h-[60vh] hover:w-[min(45vw,560px)] focus-within:h-[60vh] focus-within:w-[min(45vw,560px)]",
        // Reveal: take over the whole viewport. Positioned/​sized with ! to
        // override the hover/focus states above.
        phase === "revealed"
          ? "!inset-0 !h-full !w-full !rounded-none !border-0"
          : "",
      ].join(" ")}
    >
      <div ref={containerRef} className="h-full w-full" />
    </div>
  );
}
