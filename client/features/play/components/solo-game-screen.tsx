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
  expirePlayRoundAction,
  nextPracticeRoundAction,
  submitPlayGuessAction,
} from "@/features/play/actions";
import {
  getGameCapabilities,
  resolveSoloCapabilities,
  type PlayCapabilityMode,
} from "@/features/play/game-adapter";
import { gameResultsHref } from "@/features/play/routes";
import type {
  CurrentRound,
  Game,
  GuessResult,
} from "@/features/play/schemas";
import {
  formatRoundTimer,
  roundStatValue,
} from "@/features/mission/round-stats";
import type { AppLocale } from "@/lib/i18n/routing";
import type { Route } from "next";

export type SoloGameCopy = Readonly<{
  title: string;
  round: string;
  of: string;
  score: string;
  distance: string;
  placePin: string;
  submitGuess: string;
  submitting: string;
  next: string;
  finalScore: string;
  breakdown: string;
  backToPlay: string;
  loadingMap: string;
  mapUnavailable: string;
  retryMap: string;
  selectLocation: string;
  total: string;
  zoomIn: string;
  zoomOut: string;
  recenter: string;
  settings: string;
  openReactions: string;
  reactionsTitle: string;
  closeReactions: string;
  reactionsPrompt: string;
  reactionSent: string;
  expandMap: string;
  closeMap: string;
  mapLabel: string;
  timeExpired: string;
  correctLocation: string;
  endPractice: string;
  nextPracticeRound: string;
  errors: {
    invalidGuess: string;
    unavailable: string;
  };
}>;

type SoloGameScreenProps = Readonly<{
  locale: AppLocale;
  game: Game;
  initialRound: CurrentRound | null;
  googleMapsApiKey: string;
  copy: SoloGameCopy;
  backHref: Route | string;
  resultsHref?: Route | string;
  retryAction?: () => Promise<void>;
}>;

type Pin = { latitude: number; longitude: number };
type PendingOutcome =
  | { kind: "next_round"; nextRound: CurrentRound }
  | { kind: "completed" }
  | { kind: "practice_awaiting_next" };

function remainingRoundSeconds(round: CurrentRound | null, fallback: number) {
  if (!round?.ends_at) return fallback;
  return Math.max(
    0,
    Math.ceil((new Date(round.ends_at).getTime() - Date.now()) / 1_000),
  );
}

function resolveCapabilities(game: Game) {
  const mode = game.mode as PlayCapabilityMode;
  if (mode === "solo") {
    return resolveSoloCapabilities({ timerSeconds: game.timer_seconds });
  }
  if (mode === "quick_play" || mode === "practice" || mode === "daily") {
    return getGameCapabilities(mode);
  }
  return resolveSoloCapabilities({ timerSeconds: game.timer_seconds });
}

export function SoloGameScreen({
  locale,
  game,
  initialRound,
  googleMapsApiKey,
  copy,
  backHref,
  resultsHref,
  retryAction,
}: SoloGameScreenProps) {
  const router = useRouter();
  const capabilities = resolveCapabilities(game);
  const sceneRef = useRef<SceneViewerHandle>(null);
  const guessMapRef = useRef<GuessMapHandle>(null);
  const expiredRoundRef = useRef<string | null>(null);
  const [round, setRound] = useState(initialRound);
  const [pin, setPin] = useState<Pin | null>(null);
  const [reveal, setReveal] = useState<GuessResult | null>(null);
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
    capabilities.timed
      ? remainingRoundSeconds(initialRound, roundTimerSeconds)
      : roundTimerSeconds,
  );

  const destinationResults =
    resultsHref ?? gameResultsHref(locale, game.id);

  useEffect(() => {
    if (!capabilities.timed || !round || reveal) return;
    const timer = window.setInterval(
      () => setSecondsRemaining((remaining) => Math.max(0, remaining - 1)),
      1_000,
    );
    return () => window.clearInterval(timer);
  }, [capabilities.timed, reveal, round]);

  useEffect(() => {
    if (
      !capabilities.allowTimeout ||
      !round ||
      secondsRemaining !== 0 ||
      reveal ||
      isPending ||
      expiredRoundRef.current === round.id
    ) {
      return;
    }
    expiredRoundRef.current = round.id;
    startTransition(async () => {
      const response = await expirePlayRoundAction({
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
      if (response.kind === "completed") {
        setOutcome({ kind: "completed" });
      } else if (capabilities.openEnded) {
        setOutcome({ kind: "practice_awaiting_next" });
      } else {
        setOutcome({ kind: "next_round", nextRound: response.nextRound });
      }
    });
  }, [
    capabilities.allowTimeout,
    capabilities.openEnded,
    copy.errors.unavailable,
    game.id,
    isPending,
    reveal,
    round,
    secondsRemaining,
  ]);

  if (!round) {
    return (
      <main className="grid min-h-dvh place-items-center bg-[#08051d] text-white">
        <p>{copy.errors.unavailable}</p>
      </main>
    );
  }

  const submit = () => {
    if (capabilities.timed && secondsRemaining === 0) {
      setError(copy.timeExpired);
      return;
    }
    if (!pin) {
      setError(copy.selectLocation);
      return;
    }
    const submittedTimerLabel = capabilities.timed
      ? formatRoundTimer(secondsRemaining)
      : "—";
    setError(null);
    startTransition(async () => {
      const response = await submitPlayGuessAction({
        gameId: game.id,
        roundId: round.id,
        latitude: pin.latitude,
        longitude: pin.longitude,
        idempotencyKey: crypto.randomUUID(),
      });
      if (!response.ok) {
        setError(
          response.code === "invalid_guess" ||
            response.code === "validation_failed"
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
      if (response.kind === "completed") {
        setOutcome({ kind: "completed" });
      } else if (capabilities.openEnded) {
        setOutcome({ kind: "practice_awaiting_next" });
      } else {
        setOutcome({ kind: "next_round", nextRound: response.nextRound });
      }
    });
  };

  const advance = () => {
    if (!outcome) return;
    if (outcome.kind === "completed") {
      router.replace(destinationResults as Route);
      return;
    }
    if (outcome.kind === "practice_awaiting_next") {
      startTransition(async () => {
        const response = await nextPracticeRoundAction({
          gameId: game.id,
          idempotencyKey: `practice-next-${game.id}-${crypto.randomUUID()}`,
        });
        if (!response.ok) {
          setError(copy.errors.unavailable);
          return;
        }
        guessMapRef.current?.clearRevealArtifacts();
        expiredRoundRef.current = null;
        setRound(response.round);
        setPin(null);
        setReveal(null);
        setOutcome(null);
        setError(null);
        if (capabilities.timed) {
          setSecondsRemaining(
            remainingRoundSeconds(response.round, roundTimerSeconds),
          );
        }
      });
      return;
    }
    guessMapRef.current?.clearRevealArtifacts();
    expiredRoundRef.current = null;
    if (capabilities.timed) {
      setSecondsRemaining(
        remainingRoundSeconds(outcome.nextRound, roundTimerSeconds),
      );
    }
    setRound(outcome.nextRound);
    setPin(null);
    setReveal(null);
    setOutcome(null);
    setError(null);
  };

  const timerLabel = capabilities.timed
    ? formatRoundTimer(secondsRemaining)
    : "∞";
  const displayRoundCount = capabilities.fixedRounds
    ? game.round_count
    : Math.max(game.round_count, round.round_number);
  const roundStats = Array.from({ length: displayRoundCount }, (_, index) => {
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

  const hudCopy = {
    ...copy,
    backToMission: copy.backToPlay,
  };

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
        backHref={backHref}
        copy={hudCopy}
        heading={heading}
        secondsRemaining={capabilities.timed ? secondsRemaining : 9999}
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
        reveal={
          reveal?.actual_location
            ? {
                guess: reveal.guess,
                actual_location: reveal.actual_location,
                max_score: reveal.max_score,
                score_percent: reveal.score_percent,
              }
            : null
        }
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
