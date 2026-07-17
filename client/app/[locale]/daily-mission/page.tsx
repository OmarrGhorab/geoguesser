import { getTranslations, setRequestLocale } from "next-intl/server";
import { DailyMissionScreen } from "@/features/mission/components/daily-mission-screen";
import { PastDailyMissionEmptyScreen } from "@/features/mission/components/past-daily-mission-empty-screen";
import { PlayedTodayMissionScreen } from "@/features/mission/components/played-today-mission-screen";
import {
  resolveCompletedDailyGameId,
  shouldShowPastDailyEmptyState,
} from "@/features/mission/completed-state";
import type {
  DailyMissionCopy,
  DailyResultsCopy,
} from "@/features/mission/types";
import {
  getDailyGameResults,
  getDailyMission,
  getTodayDailyMission,
} from "@/features/mission/data";
import type { AppLocale } from "@/lib/i18n/routing";

type DailyMissionPageProps = Readonly<{
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ date?: string }>;
}>;

const monthKeys = [
  "jan",
  "feb",
  "mar",
  "apr",
  "may",
  "jun",
  "jul",
  "aug",
  "sep",
  "oct",
  "nov",
  "dec",
] as const;

export default async function DailyMissionPage({
  params,
  searchParams,
}: DailyMissionPageProps) {
  const [{ locale }, query, todayMission] = await Promise.all([
    params,
    searchParams,
    getTodayDailyMission(),
  ]);
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const todayDate = todayMission.challenge.challenge_date;
  const requestedDate = query.date;
  const selectedDate =
    requestedDate &&
    /^\d{4}-\d{2}-\d{2}$/.test(requestedDate) &&
    requestedDate <= todayDate
      ? requestedDate
      : todayDate;
  const mission =
    selectedDate === todayDate
      ? todayMission
      : await getDailyMission(selectedDate);
  const completedGameId = resolveCompletedDailyGameId(
    selectedDate,
    mission.challenge.challenge_date,
    mission.attempt_state ?? null,
    mission.last_completed_game_id,
  );
  const showPastEmptyState = shouldShowPastDailyEmptyState(
    selectedDate,
    todayDate,
    completedGameId,
  );
  const [t, completedResults] = await Promise.all([
    getTranslations("DailyMission"),
    completedGameId ? getDailyGameResults(completedGameId) : null,
  ]);

  const copy: DailyMissionCopy = {
    back: t("back"),
    today: t("today"),
    prevDay: t("prevDay"),
    nextDay: t("nextDay"),
    play: t("play"),
    resume: t("resume"),
    viewResults: t("viewResults"),
    todayOnly: t("todayOnly"),
    playedToday: t("playedToday", {
      count: mission.leaderboard_summary.participants,
    }),
    gamesProgress: t("gamesProgress", {
      completed: mission.daily_games_played,
      total: mission.daily_games_total,
    }),
    weekdays: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"].map((d) =>
      t(`weekdays.${d}`),
    ),
    months: monthKeys.map((key) => t(`months.${key}`)),
    cards: {
      locations: t("cards.locations"),
      compare: t("cards.compare"),
      daily: t("cards.daily"),
    },
    past: {
      eyebrow: t("past.eyebrow"),
      title: t("past.title"),
      noRounds: t("past.noRounds"),
      incomplete: t("past.incomplete"),
      statusNoRounds: t("past.statusNoRounds"),
      statusIncomplete: t("past.statusIncomplete"),
      todayHint: t("past.todayHint"),
      todayCta: t("past.todayCta"),
    },
    played: {
      completedTitle: t("played.completedTitle"),
      mapSummary: t("played.mapSummary"),
      score: t("played.score"),
      round: t("played.round"),
      rounds: t("played.rounds"),
      bestRound: t("played.bestRound"),
      totalDistance: t("played.totalDistance"),
      premiumTitle: t("played.premiumTitle"),
      premiumBody: t("played.premiumBody"),
      premiumBenefits: [
        t("played.premiumBenefits.adFree"),
        t("played.premiumBenefits.maps"),
        t("played.premiumBenefits.platforms"),
      ],
      mapsBenefit: t("played.benefits.mapsTitle"),
      mapsBenefitBody: t("played.benefits.mapsBody"),
      modesBenefit: t("played.benefits.modesTitle"),
      modesBenefitBody: t("played.benefits.modesBody"),
      friendsBenefit: t("played.benefits.friendsTitle"),
      friendsBenefitBody: t("played.benefits.friendsBody"),
      progressBenefit: t("played.benefits.progressTitle"),
      progressBenefitBody: t("played.benefits.progressBody"),
      completeTitle: t("played.completeTitle"),
      completeBody: t("played.completeBody"),
      yourGuess: t("played.yourGuess"),
      correctPosition: t("played.correctPosition"),
      mapUnavailable: t("played.mapUnavailable"),
      resultMap: t("played.resultMap"),
    },
    aria: {
      screen: t("aria.screen"),
      back: t("aria.back"),
      dayPicker: t("aria.dayPicker"),
    },
  };

  const resultsCopy: DailyResultsCopy = {
    title: t("game.title"),
    resultsTab: t("game.resultsTab"),
    mapTab: t("game.mapTab"),
    myGame: t("game.myGame"),
    friends: t("game.friends"),
    clubs: t("game.clubs"),
    country: t("game.country"),
    all: t("game.all"),
    player: t("game.player"),
    you: t("game.you"),
    total: t("game.total"),
    statusTitle: t("game.statusTitle"),
    statusBody: t("game.statusBody"),
    curious: t("game.curious"),
    comparisonUnavailable: t("game.comparisonUnavailable"),
    replay: t("game.replay"),
    finalScore: t("game.finalScore"),
    pointsOf: t("game.pointsOf"),
    breakdown: t("game.breakdown"),
    overview: t("game.overview"),
    next: t("game.next"),
    backToMission: t("game.backToMission"),
    round: t("game.round"),
    score: t("game.score"),
    distance: t("game.distance"),
    roundsCompleted: t("game.roundsCompleted"),
    totalDistance: t("game.totalDistance"),
    yourGuess: t("game.yourGuess"),
    correctPosition: t("game.correctPosition"),
    mapUnavailable: t("game.resultMapUnavailable"),
    aria: {
      resultMap: t("game.aria.resultMap"),
      showBreakdown: t("game.aria.showBreakdown"),
      showOverview: t("game.aria.showOverview"),
    },
  };

  if (completedResults) {
    return (
      <PlayedTodayMissionScreen
        locale={appLocale}
        copy={copy}
        resultsCopy={resultsCopy}
        results={completedResults}
        selectedDate={selectedDate}
        todayDate={todayDate}
        googleMapsApiKey={process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? ""}
        dailyGamesPlayed={mission.daily_games_played}
        dailyGamesTotal={mission.daily_games_total}
        isToday={selectedDate === todayDate}
      />
    );
  }

  if (showPastEmptyState) {
    return (
      <PastDailyMissionEmptyScreen
        locale={appLocale}
        copy={copy}
        selectedDate={selectedDate}
        todayDate={todayDate}
        hasStarted={Boolean(
          mission.attempt_state?.game_id || mission.attempt_state?.started_at,
        )}
      />
    );
  }

  return (
    <DailyMissionScreen
      locale={appLocale}
      copy={copy}
      mission={mission}
      selectedDate={selectedDate}
    />
  );
}
