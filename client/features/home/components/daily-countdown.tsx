"use client";

import { useEffect, useState } from "react";

type DailyCountdownProps = Readonly<{
  /** ISO-8601 end timestamp from daily_challenge.countdown.reset_ends_at */
  resetEndsAt: string;
  className?: string;
}>;

function formatRemaining(ms: number): string {
  if (ms <= 0) return "00:00:00";
  const totalSec = Math.floor(ms / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  return [h, m, s].map((n) => String(n).padStart(2, "0")).join(":");
}

/**
 * Client-only ticker initialized from server-provided reset_ends_at.
 * Does not refetch /home.
 */
export function DailyCountdown({
  resetEndsAt,
  className = "",
}: DailyCountdownProps) {
  const endMs = Date.parse(resetEndsAt);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (Number.isNaN(endMs)) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [endMs]);

  const label = Number.isNaN(endMs)
    ? "00:00:00"
    : formatRemaining(endMs - now);

  return (
    <p
      data-testid="daily-countdown"
      className={className}
      suppressHydrationWarning
    >
      {label}
    </p>
  );
}
