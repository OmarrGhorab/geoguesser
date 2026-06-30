type RecoveryState = "connecting" | "connected" | "reconnecting" | "degraded" | "closed";

type RecoveryBannerProps = {
  state: RecoveryState;
  labels: {
    reconnecting: string;
    degraded: string;
    disconnected: string;
    restored: string;
  };
};

export function RecoveryBanner({ state, labels }: RecoveryBannerProps) {
  if (state === "connected") {
    return (
      <div className="rounded-sm border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900" role="status">
        {labels.restored}
      </div>
    );
  }
  if (state === "reconnecting" || state === "connecting") {
    return (
      <div className="rounded-sm border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950" role="status">
        {labels.reconnecting}
      </div>
    );
  }
  if (state === "degraded") {
    return (
      <div className="rounded-sm border border-orange-200 bg-orange-50 px-3 py-2 text-sm text-orange-950" role="alert">
        {labels.degraded}
      </div>
    );
  }
  return (
    <div className="rounded-sm border border-zinc-300 bg-zinc-50 px-3 py-2 text-sm text-zinc-800" role="alert">
      {labels.disconnected}
    </div>
  );
}
