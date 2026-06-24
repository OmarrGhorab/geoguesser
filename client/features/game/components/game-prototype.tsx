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

export function GamePrototype() {
  const [location, setLocation] = useState<Location>(() => pickRandomLocation());
  const [seenLocationIds, setSeenLocationIds] = useState<readonly number[]>(() => []);
  const [guess, setGuess] = useState<LatLng | null>(null);
  const [phase, setPhase] = useState<GamePhase>("guessing");
  const [result, setResult] = useState<GuessResult | null>(null);

  const [totalScore, setTotalScore] = useState(0);
  const [round, setRound] = useState(1);

  const handlePlaceGuess = useCallback((point: LatLng) => {
    setGuess(point);
  }, []);

  const handleSubmit = useCallback(() => {
    if (!guess) return;

    const distanceKm = haversineDistanceKm(guess, location);
    const score = scoreFromDistanceKm(distanceKm);

    setResult({ distanceKm, score, guess, actual: { lat: location.lat, lng: location.lng } });
    setTotalScore((prev) => prev + score);
    setPhase("revealed");
  }, [guess, location]);

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
    <main className="relative h-screen w-screen overflow-hidden">
      <h1 className="sr-only">GeoGuess</h1>
      <StreetView location={location} />

      <ScoreBar totalScore={totalScore} round={round} />

      <GuessMap
        phase={phase}
        guess={guess}
        actual={{ lat: location.lat, lng: location.lng }}
        onPlaceGuess={handlePlaceGuess}
      />

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

      {phase === "revealed" && result && (
        <ResultCard location={location} result={result} onNext={handleNext} />
      )}
    </main>
  );
}
