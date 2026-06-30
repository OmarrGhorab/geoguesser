import type { Room } from "./types";
import { GuessProgress } from "./guess-progress";
import { RecoveryBanner } from "./recovery";
import { RoomCountdown } from "./room-countdown";
import { RoomResults } from "./room-results";

type RoomGameLabels = {
  activeTitle: string;
  round: string;
  countdown: string;
  untimed: string;
  progress: string;
  submitted: string;
  results: string;
  score: string;
  final: string;
  waiting: string;
  recovery: {
    reconnecting: string;
    degraded: string;
    disconnected: string;
    restored: string;
  };
};

type RoomGameProps = {
  room: Room;
  labels: RoomGameLabels;
};

export function RoomGame({ room, labels }: RoomGameProps) {
  if (room.status === "completed") {
    return <RoomResults room={room} labels={{ results: labels.results, score: labels.score, final: labels.final }} />;
  }

  const round = room.current_round;
  return (
    <main className="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-6 px-4 py-8">
      <section className="flex flex-col gap-3 border-b pb-6">
        <RecoveryBanner state="connected" labels={labels.recovery} />
        <p className="text-sm text-zinc-500">{room.code}</p>
        <h1 className="text-3xl font-semibold">{labels.activeTitle}</h1>
        {round ? (
          <div className="flex flex-wrap items-center gap-4">
            <p className="text-lg font-medium">
              {labels.round} {round.round_number}
            </p>
            <RoomCountdown endsAt={round.ends_at} labels={{ countdown: labels.countdown, untimed: labels.untimed }} />
          </div>
        ) : (
          <p className="text-sm text-zinc-600">{labels.waiting}</p>
        )}
      </section>

      {round?.media ? (
        <section className="overflow-hidden rounded-sm border">
          <iframe className="h-[520px] w-full" title={`${labels.round} ${round.round_number}`} src={round.media.url} />
        </section>
      ) : null}

      <GuessProgress progress={room.guess_progress} labels={{ progress: labels.progress, submitted: labels.submitted }} />
    </main>
  );
}
