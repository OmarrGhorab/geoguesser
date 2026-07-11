import type { MatchmakingStatusResponse } from "./types";

type PanelProps = {
  title: string;
  message: string;
  children?: React.ReactNode;
  role?: "status" | "alert";
  indicator?: string;
};

export function MatchmakingStatePanel({
  title,
  message,
  children,
  role = "status",
  indicator,
}: PanelProps) {
  return (
    <section
      className="mx-auto flex w-full max-w-xl flex-col gap-3 rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-950"
      role={role}
      aria-live={role === "status" ? "polite" : undefined}
    >
      <div className="flex items-start justify-between gap-3">
        <h1 className="text-xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">{title}</h1>
        {indicator ? (
          <span className="inline-flex items-center gap-2 rounded-full bg-zinc-100 px-2.5 py-1 text-xs font-medium text-zinc-700 dark:bg-zinc-900 dark:text-zinc-200">
            <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-500" aria-hidden="true" />
            <span>{indicator}</span>
          </span>
        ) : null}
      </div>
      <p className="text-sm text-zinc-600 dark:text-zinc-300">{message}</p>
      {children}
    </section>
  );
}

export function MatchmakingLoadingSkeleton() {
  return (
    <div
      className="mx-auto w-full max-w-xl animate-pulse rounded-xl border border-zinc-200 p-6 dark:border-zinc-800"
      aria-busy="true"
      aria-label="Loading matchmaking"
    >
      <div className="mb-3 h-6 w-1/2 rounded bg-zinc-200 dark:bg-zinc-800" />
      <div className="h-4 w-3/4 rounded bg-zinc-200 dark:bg-zinc-800" />
    </div>
  );
}

export function summarizeStatus(status: MatchmakingStatusResponse): string {
  switch (status.status) {
    case "not_queued":
      return "not_queued";
    case "searching":
      return "searching";
    case "matched":
      return "matched";
    case "temporarily_unavailable":
      return "temporarily_unavailable";
    default:
      return "unknown";
  }
}
