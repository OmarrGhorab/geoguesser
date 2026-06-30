import type { RoomPresenceStatus } from "./types";

type PresenceBadgeProps = {
  status: RoomPresenceStatus;
  labels: {
    connected: string;
    disconnected: string;
  };
};

export function PresenceBadge({ status, labels }: PresenceBadgeProps) {
  const active = status === "connected";

  return (
    <span
      className="inline-flex items-center gap-2 rounded-sm border px-2 py-1 text-xs font-medium"
      aria-label={active ? labels.connected : labels.disconnected}
    >
      <span className={active ? "size-2 rounded-full bg-emerald-500" : "size-2 rounded-full bg-zinc-400"} aria-hidden="true" />
      {active ? labels.connected : labels.disconnected}
    </span>
  );
}
