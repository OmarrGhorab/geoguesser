"use client";

import { useCallback, useEffect, useRef, useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { joinMatchmakingAction, leaveMatchmakingAction } from "./actions";
import { MatchmakingStatePanel } from "./matchmaking-states";
import {
  MATCHMAKING_MODE_RANKED_STANDARD,
  type MatchmakingActionState,
  type MatchmakingStatusResponse,
} from "./types";

const POLL_INTERVAL_MS = 2000;
const MAX_BACKOFF_MS = 16000;

type MatchmakingPanelProps = {
  locale: string;
  initialStatus: MatchmakingStatusResponse;
};

export function MatchmakingPanel({ locale, initialStatus }: MatchmakingPanelProps) {
  const t = useTranslations("Matchmaking");
  const router = useRouter();
  const [status, setStatus] = useState<MatchmakingStatusResponse>(initialStatus);
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const [leaving, setLeaving] = useState(false);
  const [searchStartedAt] = useState(() =>
    initialStatus.status === "searching" ? initialStatus.queue.search_started_at : null,
  );
  const [clockMs, setClockMs] = useState(0);
  const pollingRef = useRef(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const backoffRef = useRef(POLL_INTERVAL_MS);
  const focusTargetRef = useRef<HTMLButtonElement | null>(null);

  const applyAction = useCallback((result: MatchmakingActionState) => {
    if (result.ok) {
      setStatus(result.status);
      setError(null);
      return;
    }
    if (result.code === "matchmaking_claim_in_progress") {
      setError(result.message);
      return;
    }
    setError(result.message);
  }, []);

  const pollStatus = useCallback(async (): Promise<number | null> => {
    if (pollingRef.current) {
      return null;
    }
    pollingRef.current = true;
    try {
      const response = await fetch("/api/matchmaking/status", {
        method: "GET",
        credentials: "same-origin",
        cache: "no-store",
      });
      if (response.status === 429) {
        const retryAfter = Number(response.headers.get("Retry-After") || "2");
        setError(t("errors.rateLimited"));
        const delay = Math.min(Math.max(retryAfter, 2) * 1000, MAX_BACKOFF_MS);
        backoffRef.current = delay;
        return delay;
      }
      if (!response.ok) {
        if (response.status === 503) {
          setStatus({ status: "temporarily_unavailable", queue: null, match: null });
          setError(t("errors.unavailable"));
          backoffRef.current = Math.min(backoffRef.current * 2, MAX_BACKOFF_MS);
          return backoffRef.current;
        }
        setError(t("errors.unexpected"));
        backoffRef.current = Math.min(backoffRef.current * 2, MAX_BACKOFF_MS);
        return backoffRef.current;
      }
      const next = (await response.json()) as MatchmakingStatusResponse;
      setStatus(next);
      setError(null);
      backoffRef.current = POLL_INTERVAL_MS;
      if (next.status === "matched" && next.match) {
        router.push(`/${locale}${next.match.destination}`);
        return null;
      }
      if (next.status !== "searching") {
        return null;
      }
      return POLL_INTERVAL_MS;
    } catch {
      setError(t("errors.unexpected"));
      backoffRef.current = Math.min(backoffRef.current * 2, MAX_BACKOFF_MS);
      return backoffRef.current;
    } finally {
      pollingRef.current = false;
    }
  }, [locale, router, t]);

  useEffect(() => {
    if (status.status !== "searching") {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      return;
    }

    let cancelled = false;
    const schedule = async () => {
      const delay = await pollStatus();
      if (cancelled || delay == null) {
        return;
      }
      timerRef.current = setTimeout(schedule, delay);
    };
    timerRef.current = setTimeout(schedule, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [status.status, pollStatus]);

  useEffect(() => {
    if (status.status === "matched" && status.match) {
      router.push(`/${locale}${status.match.destination}`);
    }
  }, [status, locale, router]);

  useEffect(() => {
    if (status.status !== "searching") {
      return;
    }
    const boot = window.setTimeout(() => setClockMs(Date.now()), 0);
    const id = window.setInterval(() => setClockMs(Date.now()), 5000);
    return () => {
      window.clearTimeout(boot);
      window.clearInterval(id);
    };
  }, [status.status]);

  const onJoin = () => {
    startTransition(async () => {
      const result = await joinMatchmakingAction(MATCHMAKING_MODE_RANKED_STANDARD);
      applyAction(result);
    });
  };

  const onLeave = () => {
    setLeaving(true);
    startTransition(async () => {
      const result = await leaveMatchmakingAction();
      if (!result.ok && result.code === "matchmaking_claim_in_progress") {
        setError(t("errors.claimInProgress"));
        await pollStatus();
        setLeaving(false);
        return;
      }
      applyAction(result);
      setLeaving(false);
      // Preserve focus on primary control after leave.
      queueMicrotask(() => focusTargetRef.current?.focus());
    });
  };

  const onRetry = () => {
    startTransition(async () => {
      await pollStatus();
    });
  };

  if (status.status === "matched" && status.match) {
    return (
      <MatchmakingStatePanel title={t("matched.title")} message={t("matched.message")}>
        <p className="text-sm text-zinc-500" aria-live="polite">
          {t("matched.redirecting")}
        </p>
      </MatchmakingStatePanel>
    );
  }

  if (status.status === "temporarily_unavailable") {
    return (
      <MatchmakingStatePanel title={t("unavailable.title")} message={t("unavailable.message")} role="alert">
        <button
          type="button"
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
          onClick={onRetry}
          disabled={isPending}
          aria-label={t("actions.retry")}
        >
          {t("actions.retry")}
        </button>
      </MatchmakingStatePanel>
    );
  }

  if (status.status === "searching" && status.queue) {
    const started = searchStartedAt || status.queue.search_started_at;
    const delayed = clockMs > 0 && clockMs - new Date(started).getTime() > 30_000;
    return (
      <MatchmakingStatePanel
        title={delayed ? t("delayed.title") : t("searching.title")}
        message={delayed ? t("delayed.message") : t("searching.message")}
        indicator={t("searching.indicator")}
      >
        <p className="text-sm text-zinc-500" aria-live="polite">
          {t("searching.startedAt", { time: new Date(status.queue.search_started_at).toLocaleTimeString() })}
        </p>
        {error ? (
          <p className="text-sm text-red-600" role="alert">
            {error}
          </p>
        ) : null}
        <button
          type="button"
          className="rounded-md border border-zinc-300 px-4 py-2 text-sm font-medium text-zinc-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-100"
          onClick={onLeave}
          disabled={isPending || leaving}
          aria-label={t("actions.leave")}
          aria-busy={leaving || undefined}
        >
          {leaving ? t("actions.leaving") : t("actions.leave")}
        </button>
      </MatchmakingStatePanel>
    );
  }

  return (
    <MatchmakingStatePanel title={t("notQueued.title")} message={t("notQueued.message")}>
      {error ? (
        <p className="text-sm text-red-600" role="alert">
          {error}
        </p>
      ) : null}
      <button
        ref={focusTargetRef}
        type="button"
        className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
        onClick={onJoin}
        disabled={isPending}
        aria-label={t("actions.join")}
        aria-busy={isPending || undefined}
      >
        {isPending ? t("actions.joining") : t("actions.join")}
      </button>
    </MatchmakingStatePanel>
  );
}
