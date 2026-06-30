"use client";

import { useEffect, useMemo, useState } from "react";

type RoomCountdownProps = {
  endsAt: string | null;
  labels: {
    countdown: string;
    untimed: string;
  };
};

export function RoomCountdown({ endsAt, labels }: RoomCountdownProps) {
  const deadline = useMemo(() => (endsAt ? new Date(endsAt).getTime() : null), [endsAt]);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!deadline) {
      return;
    }
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [deadline]);

  if (!deadline) {
    return <p className="text-sm text-zinc-600">{labels.untimed}</p>;
  }

  const remaining = Math.max(0, Math.ceil((deadline - now) / 1000));
  return (
    <p className="text-sm font-medium text-zinc-700" aria-live="polite">
      {labels.countdown}: {remaining}s
    </p>
  );
}
