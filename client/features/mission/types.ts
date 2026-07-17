export type DailyMissionCopy = {
  back: string;
  today: string;
  prevDay: string;
  nextDay: string;
  play: string;
  resume: string;
  viewResults: string;
  todayOnly: string;
  playedToday: string;
  gamesProgress: string;
  /** Mon–Sun short labels for the calendar grid */
  weekdays: string[];
  /** January–December localized month names */
  months: string[];
  cards: {
    locations: string;
    compare: string;
    daily: string;
  };
  past: {
    eyebrow: string;
    title: string;
    noRounds: string;
    incomplete: string;
    statusNoRounds: string;
    statusIncomplete: string;
    todayHint: string;
    todayCta: string;
  };
  played: {
    completedTitle: string;
    mapSummary: string;
    score: string;
    round: string;
    rounds: string;
    bestRound: string;
    totalDistance: string;
    premiumTitle: string;
    premiumBody: string;
    premiumBenefits: string[];
    mapsBenefit: string;
    mapsBenefitBody: string;
    modesBenefit: string;
    modesBenefitBody: string;
    friendsBenefit: string;
    friendsBenefitBody: string;
    progressBenefit: string;
    progressBenefitBody: string;
    completeTitle: string;
    completeBody: string;
    yourGuess: string;
    correctPosition: string;
    mapUnavailable: string;
    resultMap: string;
  };
  aria: {
    screen: string;
    back: string;
    dayPicker: string;
  };
};

export type DailyGameCopy = {
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
  backToMission: string;
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
  errors: {
    invalidGuess: string;
    unavailable: string;
  };
};

export type DailyResultsCopy = {
  title: string;
  resultsTab: string;
  mapTab: string;
  myGame: string;
  friends: string;
  clubs: string;
  country: string;
  all: string;
  player: string;
  you: string;
  total: string;
  statusTitle: string;
  statusBody: string;
  curious: string;
  comparisonUnavailable: string;
  replay: string;
  finalScore: string;
  pointsOf: string;
  breakdown: string;
  overview: string;
  next: string;
  backToMission: string;
  round: string;
  score: string;
  distance: string;
  roundsCompleted: string;
  totalDistance: string;
  yourGuess: string;
  correctPosition: string;
  mapUnavailable: string;
  aria: {
    resultMap: string;
    showBreakdown: string;
    showOverview: string;
  };
};
