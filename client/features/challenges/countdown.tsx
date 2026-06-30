"use client";

import { useEffect, useMemo, useState } from "react";

type CountdownProps = {
  resetEndsAt: string;
  label: string;
};

export function ChallengeCountdown({ resetEndsAt, label }: CountdownProps) {
  const target = useMemo(() => new Date(resetEndsAt).getTime(), [resetEndsAt]);
  const [remaining, setRemaining] = useState(() => Math.max(0, target - Date.now()));

  useEffect(() => {
    const id = window.setInterval(() => {
      setRemaining(Math.max(0, target - Date.now()));
    }, 1000);
    return () => window.clearInterval(id);
  }, [target]);

  const totalSeconds = Math.floor(remaining / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  return (
    <p aria-live="polite" className="text-sm font-medium text-slate-700">
      {label}: {hours.toString().padStart(2, "0")}:{minutes.toString().padStart(2, "0")}:
      {seconds.toString().padStart(2, "0")}
    </p>
  );
}
