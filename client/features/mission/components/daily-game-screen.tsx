"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, useTransition } from "react";
import {
  GuessMap,
  MapControls,
  RevealOverlay,
  RoundHud,
  SceneViewer,
  type GuessMapHandle,
  type SceneViewerHandle,
} from "@/features/gameplay/components";
import {
  expireDailyRoundAction,
  submitDailyGuessAction,
} from "@/features/mission/actions";
import { dailyMissionResultsHref } from "@/features/mission/routes";
import type {
  DailyGame,
  DailyGuessResult,
  DailyRound,
} from "@/features/mission/schemas";
import type { DailyGameCopy } from "@/features/mission/types";
import {
  formatRoundTimer,
  roundStatValue,
} from "@/features/mission/round-stats";
import type { AppLocale } from "@/lib/i18n/routing";

type DailyGameScreenProps = Readonly<{
  locale: AppLocale;
  game: DailyGame;
  initialRound: DailyRound | null;
  googleMapsApiKey: string;
  copy: DailyGameCopy;
  retryAction: () => Promise<void>;
}>;
type Pin = { latitude: number; longitude: number };
type PendingOutcome =
  | { kind: "next_round"; nextRound: DailyRound }
  | { kind: "completed" };

function remainingRoundSeconds(round: DailyRound | null, fallback: number) {
  if (!round?.ends_at) return fallback;
  return Math.max(
    0,
    Math.ceil((new Date(round.ends_at).getTime() - Date.now()) / 1_000),
  );
}

export function DailyGameScreen({
  locale,
  game,
  initialRound,
  googleMapsApiKey,
  copy,
  retryAction,
}: DailyGameScreenProps) {
  const router = useRouter();
  const sceneRef = useRef<SceneViewerHandle>(null);
  const guessMapRef = useRef<GuessMapHandle>(null);
  const expiredRoundRef = useRef<string | null>(null);
  const [round, setRound] = useState(initialRound);
  const [pin, setPin] = useState<Pin | null>(null);
  const [reveal, setReveal] = useState<DailyGuessResult | null>(null);
  const [outcome, setOutcome] = useState<PendingOutcome | null>(null);
  const [totalScore, setTotalScore] = useState(game.total_score);
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const [reactionOpen, setReactionOpen] = useState(false);
  const [reaction, setReaction] = useState<string | null>(null);
  const [mapExpanded, setMapExpanded] = useState(false);
  const [heading, setHeading] = useState(0);
  const [roundTimeLabels, setRoundTimeLabels] = useState<
    Record<number, string>
  >({});
  const roundTimerSeconds = game.timer_seconds ?? 180;
  const [secondsRemaining, setSecondsRemaining] = useState(() =>
    remainingRoundSeconds(initialRound, roundTimerSeconds),
  );

  useEffect(() => {
    if (!round || reveal) return;
    const timer = window.setInterval(
      () => setSecondsRemaining((remaining) => Math.max(0, remaining - 1)),
      1_000,
    );
    return () => window.clearInterval(timer);
  }, [reveal, round]);

  useEffect(() => {
    if (
      !round ||
      secondsRemaining !== 0 ||
      reveal ||
      isPending ||
      expiredRoundRef.current === round.id
    )
      return;
    expiredRoundRef.current = round.id;
    startTransition(async () => {
      const response = await expireDailyRoundAction({
        gameId: game.id,
        roundId: round.id,
      });
      if (!response.ok) {
        setError(copy.errors.unavailable);
        return;
      }
      setRoundTimeLabels((labels) => ({
        ...labels,
        [round.round_number]: "0:00",
      }));
      setReveal(response.reveal);
      setOutcome(
        response.kind === "completed"
          ? { kind: "completed" }
          : { kind: "next_round", nextRound: response.nextRound },
      );
    });
  }, [
    copy.errors.unavailable,
    game.id,
    isPending,
    reveal,
    round,
    secondsRemaining,
  ]);

  if (!round)
    return (
      <main className="grid min-h-dvh place-items-center bg-[#08051d] text-white">
        <p>{copy.errors.unavailable}</p>
      </main>
    );

  const submit = () => {
    if (secondsRemaining === 0) {
      setError(copy.timeExpired);
      return;
    }
    if (!pin) {
      setError(copy.selectLocation);
      return;
    }
    const submittedTimerLabel = formatRoundTimer(secondsRemaining);
    setError(null);
    startTransition(async () => {
      const response = await submitDailyGuessAction({
        gameId: game.id,
        roundId: round.id,
        latitude: pin.latitude,
        longitude: pin.longitude,
        idempotencyKey: crypto.randomUUID(),
      });
      if (!response.ok) {
        setError(
          response.code === "invalid_guess"
            ? copy.errors.invalidGuess
            : copy.errors.unavailable,
        );
        return;
      }
      setRoundTimeLabels((labels) => ({
        ...labels,
        [round.round_number]: submittedTimerLabel,
      }));
      setReveal(response.reveal);
      setTotalScore((score) => score + response.reveal.guess.score);
      setOutcome(
        response.kind === "completed"
          ? { kind: "completed" }
          : { kind: "next_round", nextRound: response.nextRound },
      );
    });
  };

  const advance = () => {
    if (!outcome) return;
    if (outcome.kind === "completed") {
      router.replace(dailyMissionResultsHref(locale, game.id));
      return;
    }
    guessMapRef.current?.clearRevealArtifacts();
    expiredRoundRef.current = null;
    setSecondsRemaining(
      remainingRoundSeconds(outcome.nextRound, roundTimerSeconds),
    );
    setRound(outcome.nextRound);
    setPin(null);
    setReveal(null);
    setOutcome(null);
    setError(null);
  };

  const timerLabel = formatRoundTimer(secondsRemaining);
  const roundStats = Array.from({ length: game.round_count }, (_, index) => {
    const roundNumber = index + 1;
    return {
      roundNumber,
      active: roundNumber === round.round_number,
      value: roundStatValue(
        roundNumber,
        round.round_number,
        timerLabel,
        roundTimeLabels,
      ),
    };
  });

  return (
    <main className="relative h-dvh overflow-hidden bg-[#08051d] font-sans text-white">
      <SceneViewer
        ref={sceneRef}
        media={round.media}
        googleMapsApiKey={googleMapsApiKey}
        copy={copy}
        retryAction={retryAction}
        onHeadingChange={setHeading}
      />

      <RoundHud
        locale={locale}
        backHref={`/${locale}/daily-mission`}
        copy={copy}
        heading={heading}
        secondsRemaining={secondsRemaining}
        roundStats={roundStats}
        totalScore={totalScore}
      />

      <MapControls
        copy={copy}
        reactionOpen={reactionOpen}
        onZoomIn={() => {
          sceneRef.current?.changeZoom(1);
          guessMapRef.current?.changeZoom(1);
        }}
        onZoomOut={() => {
          sceneRef.current?.changeZoom(-1);
          guessMapRef.current?.changeZoom(-1);
        }}
        onRecenter={() => {
          sceneRef.current?.recenter();
          guessMapRef.current?.recenter();
        }}
        onOpenReactions={() => setReactionOpen(true)}
      />

      <GuessMap
        ref={guessMapRef}
        roundKey={round.id}
        googleMapsApiKey={googleMapsApiKey}
        locale={locale}
        copy={copy}
        pin={pin}
        onPinChange={setPin}
        pinLocked={Boolean(reveal)}
        reveal={reveal}
        expanded={mapExpanded}
        onExpandedChange={setMapExpanded}
        isPending={isPending}
        error={error}
        onSubmit={submit}
        onAdvance={advance}
      />

      <RevealOverlay
        open={reactionOpen}
        copy={copy}
        reaction={reaction}
        onClose={() => setReactionOpen(false)}
        onReaction={(emoji) => {
          setReaction(emoji);
          setReactionOpen(false);
        }}
      />
    </main>
  );
}
