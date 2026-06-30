import type { Room } from "./types";

type RoomResultsProps = {
  room: Room;
  labels: {
    results: string;
    score: string;
    final: string;
  };
};

export function RoomResults({ room, labels }: RoomResultsProps) {
  return (
    <section className="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-4 px-4 py-8">
      <h1 className="text-3xl font-semibold">{room.status === "completed" ? labels.final : labels.results}</h1>
      <ul className="grid gap-2">
        {room.players.map((player) => (
          <li key={player.id} className="flex items-center justify-between rounded-sm border p-3">
            <span className="font-medium">{player.display_name}</span>
            <span>
              {labels.score}: {player.total_score}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
