import type { Room } from "./types";
import { PresenceBadge } from "./presence-badge";

type LobbyProps = {
  room: Room;
  labels: {
    roomCode: string;
    copyInvite: string;
    players: string;
    host: string;
    connected: string;
    disconnected: string;
    rounds: string;
    timer: string;
    noTimer: string;
    hostControls: {
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
  };
  currentPlayerId?: string | null;
  hostControls?: React.ReactNode;
};

export function Lobby({ room, labels, hostControls }: LobbyProps) {
  const invitePath = `/rooms/${room.code}`;

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-6 px-4 py-8">
      <section className="flex flex-col gap-4 border-b pb-6">
        <div>
          <p className="text-sm text-zinc-500">{labels.roomCode}</p>
          <h1 className="text-4xl font-semibold tracking-normal">{room.code}</h1>
        </div>
        <div className="flex flex-wrap gap-3 text-sm text-zinc-600">
          <span>
            {labels.rounds}: {room.round_count}
          </span>
          <span>
            {labels.timer}: {room.timer_seconds ? `${room.timer_seconds}s` : labels.noTimer}
          </span>
          <span>
            {labels.players}: {room.players.length}/{room.max_players}
          </span>
        </div>
        <button
          className="w-fit rounded-sm border px-3 py-2 text-sm font-medium hover:bg-zinc-50"
          type="button"
          data-invite-path={invitePath}
        >
          {labels.copyInvite}
        </button>
      </section>

      {hostControls}

      <section className="flex flex-col gap-3" aria-live="polite">
        <h2 className="text-lg font-semibold">{labels.players}</h2>
        <ul className="grid gap-2">
          {room.players.map((player) => (
            <li key={player.id} className="flex items-center justify-between rounded-sm border p-3">
              <div>
                <p className="font-medium">{player.display_name}</p>
                {player.id === room.host_player_id ? <p className="text-sm text-zinc-500">{labels.host}</p> : null}
              </div>
              <PresenceBadge status={player.presence_status} labels={{ connected: labels.connected, disconnected: labels.disconnected }} />
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
