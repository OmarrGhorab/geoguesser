import type { Room } from "./types";

type HostControlsLabels = {
  title: string;
  roundCount: string;
  maxPlayers: string;
  timerSeconds: string;
  saveSettings: string;
  startRoom: string;
  removePlayer: string;
  ready: string;
  notReady: string;
  locked: string;
};

type HostControlsProps = {
  room: Room;
  currentPlayerId?: string | null;
  labels: HostControlsLabels;
  updateSettingsAction?: (formData: FormData) => void | Promise<void>;
  startAction?: () => void | Promise<void>;
  removePlayerAction?: (playerId: string) => void | Promise<void>;
};

export function HostControls({ room, currentPlayerId, labels, updateSettingsAction, startAction, removePlayerAction }: HostControlsProps) {
  const isHost = Boolean(currentPlayerId && room.host_player_id === currentPlayerId);
  const isLobby = room.status === "lobby";
  const canStart = isHost && isLobby && room.players.filter((player) => player.membership_status === "joined").length >= 2;

  return (
    <section className="flex flex-col gap-4 border-b pb-6">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold">{labels.title}</h2>
        {!isLobby ? <p className="text-sm text-zinc-500">{labels.locked}</p> : null}
      </div>

      <form action={updateSettingsAction} className="grid gap-3 sm:grid-cols-4">
        <label className="grid gap-1 text-sm">
          <span className="text-zinc-600">{labels.roundCount}</span>
          <input className="rounded-sm border px-3 py-2" name="round_count" type="number" min={1} max={10} defaultValue={room.round_count} disabled={!isHost || !isLobby} />
        </label>
        <label className="grid gap-1 text-sm">
          <span className="text-zinc-600">{labels.maxPlayers}</span>
          <input className="rounded-sm border px-3 py-2" name="max_players" type="number" min={2} max={50} defaultValue={room.max_players} disabled={!isHost || !isLobby} />
        </label>
        <label className="grid gap-1 text-sm">
          <span className="text-zinc-600">{labels.timerSeconds}</span>
          <input className="rounded-sm border px-3 py-2" name="timer_seconds" type="number" min={10} max={600} defaultValue={room.timer_seconds ?? ""} disabled={!isHost || !isLobby} />
        </label>
        <div className="flex items-end">
          <button className="w-full rounded-sm border px-3 py-2 text-sm font-medium hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50" type="submit" disabled={!isHost || !isLobby}>
            {labels.saveSettings}
          </button>
        </div>
      </form>

      <form action={startAction}>
        <button className="rounded-sm bg-zinc-900 px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50" type="submit" disabled={!canStart}>
          {labels.startRoom}
        </button>
      </form>

      <ul className="grid gap-2">
        {room.players.map((player) => (
          <li key={player.id} className="flex items-center justify-between gap-3 text-sm">
            <span>
              {player.display_name} · {player.is_ready ? labels.ready : labels.notReady}
            </span>
            {isHost && player.id !== room.host_player_id ? (
              <form action={removePlayerAction?.bind(null, player.id)}>
                <button className="rounded-sm border px-3 py-1.5 hover:bg-zinc-50" type="submit">
                  {labels.removePlayer}
                </button>
              </form>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  );
}
