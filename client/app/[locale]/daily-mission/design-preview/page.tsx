import { Suspense } from "react";
import { notFound } from "next/navigation";
import { PlayedTodayMissionScreen } from "@/features/mission/components/played-today-mission-screen";
import type { DailyGameResults } from "@/features/mission/schemas";
import type {
  DailyMissionCopy,
  DailyResultsCopy,
} from "@/features/mission/types";

const ids = [
  "11111111-1111-4111-8111-111111111111",
  "22222222-2222-4222-8222-222222222222",
  "33333333-3333-4333-8333-333333333333",
  "44444444-4444-4444-8444-444444444444",
  "55555555-5555-4555-8555-555555555555",
] as const;

const results: DailyGameResults = {
  game: {
    id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
    mode: "daily",
    status: "completed",
    map_id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
    round_count: 5,
    scoring_version: 1,
    total_score: 2708,
  },
  players: [
    {
      id: "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
      display_name: "RoadmanUAE123",
      role: "player",
      status: "active",
      total_score: 2708,
    },
  ],
  rounds: [
    [19, 8302800, 40, -74, 51.5, -0.1],
    [1, 12985800, -23.5, -46.6, 35.7, 139.7],
    [2382, 1106100, 30, 31, 59.3, 18.1],
    [101, 5820600, 28.6, 77.2, -1.3, 36.8],
    [205, 4767000, 1.3, 103.8, 48.8, 2.3],
  ].map(([score, distance, glat, glng, alat, alng], index) => ({
    round_id: ids[index]!,
    round_number: index + 1,
    actual_location: {
      latitude: alat!,
      longitude: alng!,
      country_code: "EG",
    },
    guesses: [
      {
        id: ids[index]!,
        latitude: glat!,
        longitude: glng!,
        distance_meters: distance!,
        score: score!,
        submitted_at: "2026-07-16T10:00:00Z",
        timed_out: false,
      },
    ],
  })),
};

const copy: DailyMissionCopy = {
  back: "Back",
  today: "Today",
  prevDay: "Previous day",
  nextDay: "Next day",
  play: "Play",
  resume: "Resume",
  viewResults: "View results",
  todayOnly: "Today only",
  playedToday: "1 has played today",
  gamesProgress: "Daily games: 4/5",
  weekdays: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
  months: [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
  ],
  cards: { locations: "", compare: "", daily: "" },
  past: {
    eyebrow: "Past challenge",
    title: "This daily mission has passed",
    noRounds: "You didn't play any rounds on this date.",
    incomplete: "This mission was not completed.",
    statusNoRounds: "No rounds recorded",
    statusIncomplete: "Mission not completed",
    todayHint: "Today's five new locations are ready.",
    todayCta: "Go to today's mission",
  },
  played: {
    completedTitle: "Today's mission complete",
    mapSummary: "Your five-round route",
    score: "Your score",
    round: "Round",
    rounds: "Rounds completed",
    bestRound: "Best round",
    totalDistance: "Total distance",
    premiumTitle: "Subscribe to play without limits!",
    premiumBody: "",
    premiumBenefits: [
      "Ad free, unlock all game modes",
      "Thousands of maps",
      "Play on web, apps or desktop",
    ],
    mapsBenefit: "1000+",
    mapsBenefitBody: "Maps worldwide",
    modesBenefit: "10+",
    modesBenefitBody: "Game modes",
    friendsBenefit: "Compete",
    friendsBenefitBody: "With friends",
    progressBenefit: "Track",
    progressBenefitBody: "Your progress",
    completeTitle: "Daily challenge complete",
    completeBody:
      "Your result is saved. Return tomorrow for five new locations.",
    yourGuess: "Your guess",
    correctPosition: "Correct position",
    mapUnavailable: "Map unavailable",
    resultMap: "Five-round route map",
  },
  aria: {
    screen: "Daily mission",
    back: "Back to home",
    dayPicker: "Select mission day",
  },
};

const resultsCopy: DailyResultsCopy = {
  title: "Daily mission",
  resultsTab: "Results",
  mapTab: "Map",
  myGame: "My game",
  friends: "Friends",
  clubs: "Clubs",
  country: "Egypt",
  all: "All",
  player: "Player",
  you: "You",
  total: "Total",
  statusTitle: "Daily mission",
  statusBody: "Complete!",
  curious: "Curious how others play? Take a peek at other players’ games!",
  comparisonUnavailable:
    "Other players’ round breakdowns are not available yet.",
  replay: "Replay other players’ games",
  finalScore: "Your score",
  pointsOf: "of 25,000 pts",
  breakdown: "Breakdown",
  overview: "Score overview",
  next: "Next",
  backToMission: "Back to daily mission",
  round: "Round",
  score: "Score",
  distance: "Distance",
  roundsCompleted: "Rounds completed",
  totalDistance: "Total distance",
  yourGuess: "Your guess",
  correctPosition: "Correct position",
  mapUnavailable: "Map unavailable",
  aria: {
    resultMap: "Result map",
    showBreakdown: "Show breakdown",
    showOverview: "Show overview",
  },
};

export default function DesignPreviewPage() {
  if (process.env.NODE_ENV !== "development") {
    notFound();
  }

  return (
    <Suspense
      fallback={<main className="min-h-dvh bg-[#07051d]" aria-hidden="true" />}
    >
      <PlayedTodayMissionScreen
        locale="en"
        copy={copy}
        resultsCopy={resultsCopy}
        results={results}
        selectedDate="2026-07-16"
        todayDate="2026-07-16"
        googleMapsApiKey=""
        dailyGamesPlayed={4}
        dailyGamesTotal={5}
        isToday
      />
    </Suspense>
  );
}
