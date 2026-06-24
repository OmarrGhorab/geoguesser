"use client";

import { useCallback, useState } from "react";
import GuessMap from "@/components/GuessMap";
import ResultCard from "@/components/ResultCard";
import ScoreBar from "@/components/ScoreBar";
import StreetView from "@/components/StreetView";
import { haversineDistanceKm } from "@/lib/haversine";
import { type Location, pickRandomLocation } from "@/lib/locations";
import { scoreFromDistanceKm } from "@/lib/scoring";
import type { GamePhase, GuessResult, LatLng } from "@/lib/types";

/**
 * The single source of truth for a game session.
 *
 * All game state lives here; the child components are presentational or own
 * only imperative (map) state. We deliberately keep `location`/`guess`/
 * `phase` as separate fields rather than a single reducer — the transitions
 * are simple enough that explicit useState calls read more clearly.
 *
 * The render is an immersive, full-viewport stage: the Street View panorama
 * fills the screen and every control floats above it (see the z-layer map in
 * the JSX below). This component owns only the layout/wiring — sizing,
 * glass styling and the map's expand/reveal animation live in the children.
 */
export default function Home() {
  // Round 1 starts with a random landmark.
  const [location, setLocation] = useState<Location>(() => pickRandomLocation());
  const [seenLocationIds, setSeenLocationIds] = useState<readonly number[]>(() => []);
  const [guess, setGuess] = useState<LatLng | null>(null);
  const [phase, setPhase] = useState<GamePhase>("guessing");
  const [result, setResult] = useState<GuessResult | null>(null);

  const [totalScore, setTotalScore] = useState(0);
  const [round, setRound] = useState(1);

  /** Player dropped/moved their guess marker. */
  const handlePlaceGuess = useCallback((point: LatLng) => {
    setGuess(point);
  }, []);

  /** Lock in the guess: compute distance + score, reveal the answer. */
  const handleSubmit = useCallback(() => {
    if (!guess) return; // Submit button is disabled when there's no guess; defensive guard.

    const distanceKm = haversineDistanceKm(guess, location);
    const score = scoreFromDistanceKm(distanceKm);

    setResult({ distanceKm, score, guess, actual: { lat: location.lat, lng: location.lng } });
    setTotalScore((prev) => prev + score);
    setPhase("revealed");
  }, [guess, location]);

  /** Advance to the next round: fresh landmark, reset transient state. */
  const handleNext = useCallback(() => {
    const nextSeenIds =
      seenLocationIds.length >= 149 ? [location.id] : [...seenLocationIds, location.id];

    setSeenLocationIds(nextSeenIds);
    setLocation(pickRandomLocation({ excludeIds: nextSeenIds }));
    setGuess(null);
    setResult(null);
    setPhase("guessing");
    setRound((prev) => prev + 1);
  }, [location.id, seenLocationIds]);

  const canSubmit = guess !== null && phase === "guessing";

  return (
    // One full-viewport stage. The Street View panorama fills it (z-0) and
    // every interactive element floats above as a glass overlay:
    //   • ScoreBar  — top-left pill (z-20)
    //   • submit button — top-right (z-20), only while guessing
    //   • GuessMap  — corner panel → full-screen on reveal (z-10)
    //   • ResultCard — centered overlay (z-30), only after submit
    <div className="relative h-screen w-screen overflow-hidden">
      <StreetView location={location} />

      <ScoreBar totalScore={totalScore} round={round} />

      {/* Interactive guessing map (corner panel → full-screen on reveal). */}
      <GuessMap
        phase={phase}
        guess={guess}
        actual={{ lat: location.lat, lng: location.lng }}
        onPlaceGuess={handlePlaceGuess}
      />

      {/* Submit — disabled until a marker is placed, hidden after reveal. */}
      {phase === "guessing" && (
        <div className="absolute right-4 top-4 z-20">
          <button
            type="button"
            onClick={handleSubmit}
            disabled={!canSubmit}
            className="rounded-lg bg-white px-6 py-2.5 text-sm font-semibold text-zinc-900 shadow-lg ring-1 ring-black/5 transition-colors hover:bg-white/90 focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-black/40 disabled:cursor-not-allowed disabled:bg-zinc-500 disabled:text-zinc-300 disabled:hover:bg-zinc-500"
          >
            {guess ? "Submit Guess" : "Click the map to guess"}
          </button>
        </div>
      )}

      {/* Result overlay — only after submission, above the revealed map. */}
      {phase === "revealed" && result && (
        <ResultCard location={location} result={result} onNext={handleNext} />
      )}
    </div>
  );
}
