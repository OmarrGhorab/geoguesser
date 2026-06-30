import type { RoomGuessProgress } from "./types";

type GuessProgressProps = {
  progress?: RoomGuessProgress;
  labels: {
    progress: string;
    submitted: string;
  };
};

export function GuessProgress({ progress, labels }: GuessProgressProps) {
  const submitted = progress?.submitted_count ?? 0;
  const eligible = progress?.eligible_count ?? 0;

  return (
    <div className="rounded-sm border p-3" aria-live="polite">
      <p className="text-sm font-medium">{labels.progress}</p>
      <p className="text-2xl font-semibold">
        {submitted}/{eligible}
      </p>
      <p className="text-sm text-zinc-600">{labels.submitted}</p>
    </div>
  );
}
