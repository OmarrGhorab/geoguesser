import type { Location } from "@/lib/locations";
import { MAX_SCORE } from "@/lib/scoring";
import type { GuessResult } from "@/lib/types";

/**
 * Post-submit result panel: reveals the answer and the round's score, and
 * offers the "Next Location" action. Presentational only — all data and the
 * onNext handler come from the parent.
 */
export type ResultCardProps = {
  /** The landmark that was being guessed. */
  location: Location;
  /** Computed distances/score for this round. */
  result: GuessResult;
  /** Advance to the next round. */
  onNext: () => void;
};

export default function ResultCard({ location, result, onNext }: ResultCardProps) {
  return (
    // Centered glass overlay floating above the full-screen revealed map.
    <section className="absolute bottom-6 left-1/2 z-30 w-[min(92vw,640px)] -translate-x-1/2 rounded-2xl bg-black/60 px-5 py-4 text-white shadow-2xl ring-1 ring-white/15 backdrop-blur-md">
      <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold">
            {location.name}
            <span className="font-normal text-white/60">
              {" "}
              — {location.city}, {location.country}
            </span>
          </div>
          <div className="mt-0.5 text-xs text-white/60">
            {location.region} · {location.difficulty} · Your guess was{" "}
            <span className="font-medium text-white/90">
              {result.distanceKm.toLocaleString()} km
            </span>{" "}
            off
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="text-right">
            <div className="text-xs uppercase tracking-wide text-white/50">Score</div>
            <div className="text-lg font-bold text-emerald-400">
              {result.score.toLocaleString()}
              <span className="text-sm font-normal text-white/40">
                {" "}
                / {MAX_SCORE.toLocaleString()}
              </span>
            </div>
          </div>

          <button
            type="button"
            onClick={onNext}
            className="rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-zinc-900 transition-colors hover:bg-white/90 focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-black/60"
          >
            Next Location
          </button>
        </div>
      </div>
    </section>
  );
}
