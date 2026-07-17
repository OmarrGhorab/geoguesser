import Image from "next/image";
import { Flag, MapPin } from "lucide-react";
import { MISSION_ASSETS } from "@/features/mission/assets";
import type { DailyGameResults } from "@/features/mission/schemas";

type MissionRouteMapProps = Readonly<{
  rounds: DailyGameResults["rounds"];
  roundLabel: string;
  ariaLabel: string;
}>;

type Point = { x: number; y: number };

function project(latitude: number, longitude: number): Point {
  return {
    x: ((longitude + 180) / 360) * 100,
    y: ((90 - latitude) / 180) * 100,
  };
}

export function MissionRouteMap({
  rounds: unsortedRounds,
  roundLabel,
  ariaLabel,
}: MissionRouteMapProps) {
  const rounds = [...unsortedRounds].sort(
    (left, right) => left.round_number - right.round_number,
  );

  return (
    <div role="img" aria-label={ariaLabel} className="absolute inset-0">
      <Image
        src={MISSION_ASSETS.background}
        alt=""
        fill
        priority
        sizes="(max-width: 960px) 100vw, 920px"
        className="object-cover"
      />
      <div className="absolute inset-0 bg-[linear-gradient(180deg,rgba(7,5,29,.02),rgba(7,5,29,.22))]" />

      <svg
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        aria-hidden="true"
        className="absolute inset-0 size-full overflow-visible"
      >
        {rounds.map((round) => {
          const guess = round.guesses[0];
          if (!guess) return null;
          const from = project(guess.latitude, guess.longitude);
          const to = project(
            round.actual_location.latitude,
            round.actual_location.longitude,
          );
          return (
            <line
              key={round.round_id}
              x1={from.x}
              y1={from.y}
              x2={to.x}
              y2={to.y}
              vectorEffect="non-scaling-stroke"
              stroke="#f6b71d"
              strokeWidth="2"
              strokeDasharray="5 5"
              strokeLinecap="round"
              className="drop-shadow-[0_0_5px_rgba(246,183,29,.75)]"
            />
          );
        })}
      </svg>

      {rounds.flatMap((round) => {
        const guess = round.guesses[0];
        const answer = project(
          round.actual_location.latitude,
          round.actual_location.longitude,
        );
        const markers = [
          <span
            key={`${round.round_id}-answer`}
            title={`${roundLabel} ${round.round_number}`}
            style={{ left: `${answer.x}%`, top: `${answer.y}%` }}
            className="absolute grid size-8 -translate-1/2 place-items-center rounded-full border-2 border-white bg-[linear-gradient(145deg,#ad43ff,#5f22c7)] text-white shadow-[0_0_15px_rgba(157,70,255,.85)] sm:size-9"
          >
            <Flag className="size-4 fill-white" aria-hidden="true" />
          </span>,
        ];
        if (guess) {
          const guessed = project(guess.latitude, guess.longitude);
          markers.push(
            <span
              key={`${round.round_id}-guess`}
              title={`${roundLabel} ${round.round_number}`}
              style={{ left: `${guessed.x}%`, top: `${guessed.y}%` }}
              className="absolute grid size-8 -translate-1/2 place-items-center rounded-full border-2 border-white bg-[linear-gradient(145deg,#ffb048,#e05b46)] text-[.65rem] font-black text-white shadow-[0_0_15px_rgba(255,147,54,.75)] sm:size-9"
            >
              <MapPin
                className="absolute size-5 opacity-25"
                aria-hidden="true"
              />
              <span className="relative">{round.round_number}</span>
            </span>,
          );
        }
        return markers;
      })}
    </div>
  );
}
