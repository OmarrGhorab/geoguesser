/**
 * Floating glass "HUD" pill pinned top-left: title + running score + round.
 * Sits above the full-viewport Street View (z-20). Purely presentational.
 */
export type ScoreBarProps = {
  totalScore: number;
  round: number;
};

export default function ScoreBar({ totalScore, round }: ScoreBarProps) {
  return (
    <header className="pointer-events-none absolute left-4 top-4 z-20 select-none">
      <div className="flex items-center gap-3 rounded-full bg-black/40 px-4 py-2 text-sm text-white shadow-lg ring-1 ring-white/15 backdrop-blur-md">
        <span className="text-base font-semibold tracking-tight">GeoGuessr</span>

        <span className="h-4 w-px bg-white/20" aria-hidden />

        <span className="text-white/70">
          Round <span className="font-semibold text-white">{round}</span>
        </span>

        <span className="text-white/70">
          Score <span className="font-semibold text-white">{totalScore.toLocaleString()}</span>
        </span>
      </div>
    </header>
  );
}
