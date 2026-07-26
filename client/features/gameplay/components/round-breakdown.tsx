import { Trophy } from "lucide-react";
import type { AppLocale } from "@/lib/i18n/routing";
import { cn } from "@/lib/utils";

/** Neutral per-round score/distance row for solo and daily results. */
export type RoundBreakdownItem = {
  roundNumber: number;
  score: number;
  distanceMeters: number;
};

export type RoundBreakdownProps = Readonly<{
  rounds: readonly RoundBreakdownItem[];
  locale: AppLocale;
  totalScore: number;
  totalDistanceMeters: number;
  /** Per-round maximum score; used for high-score trophy threshold (80%). */
  maxScore?: number;
  roundLabel: string;
  totalLabel: string;
  className?: string;
}>;

function formatDistance(locale: AppLocale, distanceMeters: number) {
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(distanceMeters / 1_000)} km`;
}

/**
 * Per-round score/distance grid with total cell. Styling matches Daily results.
 */
export function RoundBreakdown({
  rounds,
  locale,
  totalScore,
  totalDistanceMeters,
  maxScore = 5_000,
  roundLabel,
  totalLabel,
  className,
}: RoundBreakdownProps) {
  const trophyThreshold = maxScore * 0.8;

  return (
    <ol
      className={cn(
        "grid grid-cols-2 gap-px border-t border-violet-400/15 bg-violet-400/10 p-px sm:grid-cols-6",
        className,
      )}
    >
      {rounds.map((round) => (
        <li key={round.roundNumber} className="bg-[#0d092b] p-3">
          <p className="flex items-center gap-1 text-[.62rem] font-black italic">
            <span>
              {roundLabel} {round.roundNumber}
            </span>
            {round.score >= trophyThreshold ? (
              <Trophy
                className="size-3 fill-[#ffc341] text-[#ffc341]"
                aria-hidden="true"
              />
            ) : null}
          </p>
          <p className="mt-2 text-sm font-black">
            {round.score.toLocaleString(locale)} pts
          </p>
          <p className="mt-1 text-[.6rem] text-white/50">
            {formatDistance(locale, round.distanceMeters)}
          </p>
        </li>
      ))}
      <li className="bg-[#0d092b] p-3">
        <p className="text-[.62rem] font-black italic">{totalLabel}</p>
        <p className="mt-2 text-sm font-black">
          {totalScore.toLocaleString(locale)} pts
        </p>
        <p className="mt-1 text-[.6rem] text-white/50">
          {formatDistance(locale, totalDistanceMeters)}
        </p>
      </li>
    </ol>
  );
}
